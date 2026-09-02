package classes

import (
	"context"
	"errors"
	"fmt"
	"time"

	"main/internal/application/location"
	"main/internal/domain/errs/api"
	"main/internal/domain/models"
	"main/internal/domain/notifier"
	"main/internal/domain/repositories"
	"main/internal/domain/services/passes"
	repositoryError "main/internal/infrastructure/errs"

	"github.com/google/uuid"
)

type service struct {
	classesRepo      repositories.IClasses
	bookingsRepo     repositories.IBookings
	unitOfWork       repositories.IUnitOfWork
	notifier         notifier.INotifier
	locationResolver location.ILinkProvider
	domainAddr       string
}

func NewService(
	classesRepo repositories.IClasses,
	bookingsRepo repositories.IBookings,
	unitOfWork repositories.IUnitOfWork,
	notifier notifier.INotifier,
	locationResolver location.ILinkProvider,
	domainAddr string,
) *service {
	return &service{
		classesRepo:      classesRepo,
		bookingsRepo:     bookingsRepo,
		unitOfWork:       unitOfWork,
		notifier:         notifier,
		locationResolver: locationResolver,
		domainAddr:       domainAddr,
	}
}

func (s *service) ListClasses(
	ctx context.Context,
	onlyUpcomingClasses bool,
	classesLimit *int,
) ([]ClassPresentation, error) {
	if classesLimit != nil && *classesLimit < 0 {
		return nil, api.ErrValidation(
			fmt.Errorf("classes_limit must be greater than or equal to 0, got: %d", *classesLimit),
		)
	}

	classes, err := s.classesRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get all classes: %w", err)
	}

	classPresentations, err := s.buildClassPresentations(ctx, classes)
	if err != nil {
		return nil, fmt.Errorf("could build classPresentations: %w", err)
	}

	if onlyUpcomingClasses {
		filtered := make([]ClassPresentation, 0, len(classPresentations))

		now := time.Now()
		for _, class := range classPresentations {
			if class.StartTime.After(now) {
				filtered = append(filtered, class)
			}
		}

		classPresentations = filtered
	}

	if classesLimit != nil {
		limit := min(*classesLimit, len(classPresentations))
		classPresentations = classPresentations[:limit]
	}

	return classPresentations, nil
}

func (s *service) buildClassPresentations(
	ctx context.Context, classes []models.Class,
) ([]ClassPresentation, error) {
	classPresentations := make([]ClassPresentation, 0, len(classes))

	for _, class := range classes {
		bookingCount, err := s.bookingsRepo.CountForClassID(ctx, class.ID)
		if err != nil {
			return nil, fmt.Errorf("could not get bookings for class %v: %w", class.ID, err)
		}

		locationLink, err := s.locationResolver.GetLink(class.Location)
		if err != nil {
			return nil, fmt.Errorf("could not get location link for: %s, err: %w", class.Location, err)
		}

		classPresentations = append(classPresentations, ClassPresentation{
			ID:              class.ID,
			StartTime:       class.StartTime,
			ClassLevel:      class.ClassLevel,
			ClassName:       class.ClassName,
			CurrentCapacity: class.MaxCapacity - bookingCount,
			MaxCapacity:     class.MaxCapacity,
			Location:        class.Location,
			LocationLink:    locationLink,
		})
	}

	return classPresentations, nil
}

func (s *service) CreateClasses(
	ctx context.Context, newClasses []models.Class,
) ([]models.Class, error) {
	existingClasses, err := s.classesRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get existing classes: %w", err)
	}

	err = validateClasses(newClasses, existingClasses)
	if err != nil {
		return nil, api.ErrValidation(err)
	}

	insertedClasses, err := s.classesRepo.Insert(ctx, newClasses)
	if err != nil {
		return nil, fmt.Errorf("could not insert classes: %w", err)
	}

	return insertedClasses, nil
}

func (s *service) DeleteClass(ctx context.Context, classID uuid.UUID, msg *string) error {
	var notifierParamsList []models.NotifierParams

	err := s.unitOfWork.WithTransaction(ctx, func(repos repositories.Repositories) error {
		bookings, err := repos.Bookings.ListByClassID(ctx, classID)
		if err != nil {
			return fmt.Errorf("could not get classes for classID %v: %w", classID, err)
		}

		if err := validateDeleteMessage(bookings, msg); err != nil {
			return fmt.Errorf("could not validate delete message for classID %v: %w", classID, err)
		}

		locationLink, err := s.getLocationLink(bookings)
		if err != nil {
			return fmt.Errorf("could not get locationLink: %w", err)
		}

		for _, booking := range bookings {
			err := repos.Bookings.Delete(ctx, booking.ID)
			if err != nil {
				return fmt.Errorf("could not delete booking for id %v: %w", booking.ID, err)
			}

			notifierParams, err := s.buildNotifierParamsForDelete(ctx, repos, booking, locationLink)
			if err != nil {
				return fmt.Errorf("could not build notifierParams for delete: %w", err)
			}

			notifierParamsList = append(notifierParamsList, notifierParams)
		}

		return deleteClassRow(ctx, repos, classID)
	})
	if err != nil {
		return fmt.Errorf("delete class transaction failed: %w", err)
	}

	err = s.notifyClassCancellation(notifierParamsList, msg)
	if err != nil {
		return fmt.Errorf("could not notify class cancellation: %w", err)
	}

	return nil
}

func deleteClassRow(ctx context.Context, repos repositories.Repositories, classID uuid.UUID) error {
	err := repos.Classes.Delete(ctx, classID)
	if err != nil {
		if errors.Is(err, repositoryError.ErrNoRowsAffected) {
			return api.ErrNotFound(err)
		}

		return fmt.Errorf("could not delete class: %w", err)
	}

	return nil
}

func validateDeleteMessage(bookings []models.Booking, msg *string) error {
	if len(bookings) > 0 && msg == nil {
		return api.ErrValidation(
			errors.New("reason msg can not be empty, when classes has bookings"),
		)
	}

	return nil
}

func (s *service) getLocationLink(bookings []models.Booking) (string, error) {
	if len(bookings) == 0 {
		return "", nil
	}

	location := bookings[0].Class.Location

	locationLink, err := s.locationResolver.GetLink(location)
	if err != nil {
		return "", fmt.Errorf("could not get location link for location %q: %w", location, err)
	}

	return locationLink, nil
}

func (s *service) buildNotifierParamsForDelete(
	ctx context.Context,
	repos repositories.Repositories,
	booking models.Booking,
	locationLink string,
) (models.NotifierParams, error) {
	notifierParams := models.NotifierParams{
		RecipientFirstName: booking.FirstName,
		RecipientLastName:  booking.LastName,
		RecipientEmail:     booking.Email,
		ClassName:          booking.Class.ClassName,
		ClassLevel:         booking.Class.ClassLevel,
		StartTime:          booking.Class.StartTime,
		Location:           booking.Class.Location,
		LocationLink:       locationLink,
	}

	if booking.Pass.Exists() {
		pass := booking.Pass.Get()

		usedBookings, err := repos.Bookings.ListByPassID(ctx, pass.ID)
		if err != nil {
			return models.NotifierParams{},
				fmt.Errorf("could not get bookings for pass id %d: %w", pass.ID, err)
		}

		notifierParams.PassSlots = passes.BuildPassSlots(usedBookings, pass.TotalSlots, time.Now())
	}

	return notifierParams, nil
}

func (s *service) notifyClassCancellation(
	paramsList []models.NotifierParams,
	msg *string,
) error {
	for _, params := range paramsList {
		if err := s.notifier.NotifyClassCancellation(params, *msg); err != nil {
			return fmt.Errorf("could not notify class cancellation with %+v: %w", params, err)
		}
	}

	return nil
}

func (s *service) UpdateClass(
	ctx context.Context, classID uuid.UUID, update UpdateClassCommand,
) (ClassData, error) {
	err := s.ensureClassUpdate(ctx, classID, update)
	if err != nil {
		return ClassData{}, fmt.Errorf("update class not possible: %w", err)
	}

	change, err := getDataForClassUpdate(update)
	if err != nil {
		return ClassData{}, fmt.Errorf("could not get data for class update: %w", err)
	}

	updatedClass, err := s.classesRepo.Update(ctx, classID, change)
	if err != nil {
		return ClassData{}, fmt.Errorf("could not update class: %w", err)
	}

	err = s.sendInformationAboutClassUpdateToUsers(ctx, update, updatedClass)
	if err != nil {
		return ClassData{}, fmt.Errorf("could not get class after update: %w", err)
	}

	classData, err := s.buildClassData(ctx, updatedClass)
	if err != nil {
		return ClassData{}, fmt.Errorf("could not build classData: %w", err)
	}

	return classData, nil
}

func (s *service) ensureClassUpdate(
	ctx context.Context, classID uuid.UUID, update UpdateClassCommand,
) error {
	if (update.Location != nil || update.StartTime != nil) && update.Message == nil {
		return api.ErrValidation(
			errors.New("message cannot be empty when updating location or class startTime"),
		)
	}

	existingClasses, err := s.classesRepo.List(ctx)
	if err != nil {
		if errors.Is(err, repositoryError.ErrNotFound) {
			return api.ErrNotFound(err)
		}

		return fmt.Errorf("could not get existing classes: %w", err)
	}

	if update.StartTime != nil {
		err := validateClassStartTime(*update.StartTime, existingClasses)
		if err != nil {
			return api.ErrValidation(err)
		}
	}

	_, err = s.classesRepo.Get(ctx, classID)
	if err != nil {
		if errors.Is(err, repositoryError.ErrNotFound) {
			return api.ErrNotFound(err)
		}

		return fmt.Errorf("could not get class for class_id %v: %w", classID, err)
	}

	return nil
}

func (s *service) sendInformationAboutClassUpdateToUsers(
	ctx context.Context, update UpdateClassCommand, updatedClass models.Class,
) error {
	if update.Location == nil && update.StartTime == nil {
		return nil
	}

	bookings, err := s.bookingsRepo.ListByClassID(ctx, updatedClass.ID)
	if err != nil {
		return fmt.Errorf("could not get bookings for class %v: %w", updatedClass.ID, err)
	}

	locationLink, err := s.locationResolver.GetLink(updatedClass.Location)
	if err != nil {
		return fmt.Errorf("could not get location link for location: %s", updatedClass.Location)
	}

	for _, booking := range bookings {
		notifierParams := models.NotifierParams{
			RecipientEmail:     booking.Email,
			RecipientFirstName: booking.FirstName,
			RecipientLastName:  booking.LastName,
			ClassName:          updatedClass.ClassName,
			ClassLevel:         updatedClass.ClassLevel,
			StartTime:          updatedClass.StartTime,
			Location:           updatedClass.Location,
			LocationLink:       locationLink,
		}

		cancellationLink := fmt.Sprintf(
			"%s/bookings/%s/cancel_form?token=%s", s.domainAddr, booking.ID, booking.ConfirmationToken,
		)

		err = s.notifier.NotifyClassUpdate(notifierParams, *update.Message, cancellationLink)
		if err != nil {
			return fmt.Errorf("could not notify class update to with %+v: %w", notifierParams, err)
		}
	}

	return nil
}

func (s *service) buildClassData(ctx context.Context, class models.Class) (ClassData, error) {
	bookingCount, err := s.bookingsRepo.CountForClassID(ctx, class.ID)
	if err != nil {
		return ClassData{}, fmt.Errorf("could not get bookings for class %v: %w", class.ID, err)
	}

	return ClassData{
		ID:              class.ID,
		StartTime:       class.StartTime,
		ClassLevel:      class.ClassLevel,
		ClassName:       class.ClassName,
		MaxCapacity:     class.MaxCapacity,
		CurrentCapacity: class.MaxCapacity - bookingCount,
		Location:        class.Location,
	}, nil
}

func getDataForClassUpdate(update UpdateClassCommand) (map[string]any, error) {
	updateData := map[string]any{}
	if update.StartTime != nil {
		updateData["start_time"] = *update.StartTime
	}

	if update.ClassLevel != nil {
		updateData["class_level"] = *update.ClassLevel
	}

	if update.ClassName != nil {
		updateData["class_name"] = *update.ClassName
	}

	if update.MaxCapacity != nil {
		updateData["max_capacity"] = *update.MaxCapacity
	}

	if update.Location != nil {
		updateData["location"] = *update.Location
	}

	if len(updateData) == 0 {
		return nil, api.ErrValidation(errors.New("no fields to update class"))
	}

	return updateData, nil
}

func validateClasses(newClasses, existingClasses []models.Class) error {
	for _, class := range newClasses {
		err := validateClassStartTime(class.StartTime, existingClasses)
		if err != nil {
			return fmt.Errorf("startTime validation failed %w", err)
		}
	}

	return nil
}

func validateClassStartTime(startTime time.Time, existingClasses []models.Class) error {
	if startTime.Before(time.Now()) {
		return fmt.Errorf("class startTime: %v expired", startTime)
	}

	for _, existingClass := range existingClasses {
		if startTime.Equal(existingClass.StartTime) {
			return fmt.Errorf("class with startTime %v already exists", startTime)
		}
	}

	return nil
}
