package dto

import (
	"fmt"

	appModels "main/internal/application/models"
	"main/pkg/converter"
	"main/pkg/translator"

	"github.com/google/uuid"
)

type ClassView struct {
	ID              uuid.UUID `json:"id"`
	WeekDay         string    `json:"week_day"`
	StartDate       string    `json:"start_date"`
	StartHour       string    `json:"start_hour"`
	ClassLevel      string    `json:"class_level"`
	ClassName       string    `json:"class_name"`
	CurrentCapacity int       `json:"current_capacity"`
	MaxCapacity     int       `json:"max_capacity"`
	Location        string    `json:"location"`
	LocationLink    string    `json:"location_link"`
}

func FromClassPresentation(class appModels.ClassPresentation) (ClassView, error) {
	warsawTime, err := converter.ConvertToWarsawTime(class.StartTime)
	if err != nil {
		return ClassView{},
			fmt.Errorf("error while converting time to warsaw time: %w", err)
	}

	weekday, err := translator.TranslateToWeekDayToPolish(warsawTime.Weekday())
	if err != nil {
		return ClassView{},
			fmt.Errorf("error while translating week day to polish: %w", err)
	}

	return ClassView{
		ID:              class.ID,
		WeekDay:         weekday,
		StartDate:       warsawTime.Format(converter.DateLayout),
		StartHour:       warsawTime.Format(converter.HourLayout),
		ClassLevel:      class.ClassLevel,
		ClassName:       class.ClassName,
		CurrentCapacity: class.CurrentCapacity,
		MaxCapacity:     class.MaxCapacity,
		Location:        class.Location,
		LocationLink:    class.LocationLink,
	}, nil
}

func FromClassPresentations(classes []appModels.ClassPresentation) ([]ClassView, error) {
	classesView := make([]ClassView, len(classes))

	for idx, class := range classes {
		classView, err := FromClassPresentation(class)
		if err != nil {
			return nil, fmt.Errorf("could not convert classListItem to ClassView : %w", err)
		}

		classesView[idx] = classView
	}

	return classesView, nil
}

type BookingCancellationClassView struct {
	ID         uuid.UUID `json:"id"`
	WeekDay    string    `json:"week_day"`
	StartDate  string    `json:"start_date"`
	StartHour  string    `json:"start_hour"`
	ClassLevel string    `json:"class_level"`
	ClassName  string    `json:"class_name"`
	Location   string    `json:"location"`
}

func FromBookingCancellationClass(
	class appModels.BookingCancellationClass,
) (BookingCancellationClassView, error) {
	warsawTime, err := converter.ConvertToWarsawTime(class.StartTime)
	if err != nil {
		return BookingCancellationClassView{},
			fmt.Errorf("error while converting time to warsaw time: %w", err)
	}

	weekday, err := translator.TranslateToWeekDayToPolish(warsawTime.Weekday())
	if err != nil {
		return BookingCancellationClassView{},
			fmt.Errorf("error while translating week day to polish: %w", err)
	}

	return BookingCancellationClassView{
		ID:         class.ID,
		WeekDay:    weekday,
		StartDate:  warsawTime.Format(converter.DateLayout),
		StartHour:  warsawTime.Format(converter.HourLayout),
		ClassLevel: class.ClassLevel,
		ClassName:  class.ClassName,
		Location:   class.Location,
	}, nil
}
