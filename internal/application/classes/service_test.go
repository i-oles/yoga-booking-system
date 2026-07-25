package classes

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	domainModels "main/internal/domain/models"
	"main/internal/domain/repositories"
	repositoryError "main/internal/infrastructure/errs"
	"main/mock"
	"main/pkg/optional"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

var (
	now         = time.Now()
	pastTime    = now.Add(-2 * time.Hour)
	futureTime1 = now.Add(1 * time.Hour)
	futureTime2 = now.Add(2 * time.Hour)
	futureTime3 = now.Add(3 * time.Hour)

	classID1 = uuid.New()
	classID2 = uuid.New()
	classID3 = uuid.New()
	classID4 = uuid.New()

	bookingID2 = uuid.New()

	passID1 = 1234

	otoJogaStudio = "studio otojoga"
)

var futureClass = domainModels.Class{
	ID:          classID1,
	StartTime:   futureTime1,
	ClassLevel:  "Beginner",
	ClassName:   "Vinyasa",
	MaxCapacity: 5,
	Location:    "Studio A",
}

var futureClassPresentation = ClassPresentation{
	ID:              futureClass.ID,
	StartTime:       futureClass.StartTime,
	ClassLevel:      futureClass.ClassLevel,
	ClassName:       futureClass.ClassName,
	CurrentCapacity: 3,
	MaxCapacity:     futureClass.MaxCapacity,
	Location:        futureClass.Location,
	LocationLink:    "link-a",
}

var pastClass = domainModels.Class{
	ID:          classID2,
	StartTime:   pastTime,
	ClassLevel:  "Beginner",
	ClassName:   "Vinyasa",
	MaxCapacity: 5,
	Location:    "Studio A",
}

var futureClasses = []domainModels.Class{
	{
		ID:          classID1,
		StartTime:   futureTime1,
		ClassLevel:  "Intermediate",
		ClassName:   "Ashtanga",
		MaxCapacity: 15,
		Location:    "Studio B",
	},
	{
		ID:          classID3,
		StartTime:   futureTime2,
		ClassLevel:  "Advanced",
		ClassName:   "Vinyasa",
		MaxCapacity: 12,
		Location:    "Studio C",
	},
}

var futureClassesPresentations = []ClassPresentation{
	{
		ID:              futureClasses[0].ID,
		StartTime:       futureClasses[0].StartTime,
		ClassLevel:      futureClasses[0].ClassLevel,
		ClassName:       futureClasses[0].ClassName,
		CurrentCapacity: 12,
		MaxCapacity:     futureClasses[0].MaxCapacity,
		Location:        futureClasses[0].Location,
		LocationLink:    "link-b",
	},
	{
		ID:              futureClasses[1].ID,
		StartTime:       futureClasses[1].StartTime,
		ClassLevel:      futureClasses[1].ClassLevel,
		ClassName:       futureClasses[1].ClassName,
		CurrentCapacity: 7,
		MaxCapacity:     futureClasses[1].MaxCapacity,
		Location:        futureClasses[1].Location,
		LocationLink:    "link-c",
	},
}

var futureClassesPresentationsWithoutBookings = []ClassPresentation{
	{
		ID:              futureClasses[0].ID,
		StartTime:       futureClasses[0].StartTime,
		ClassLevel:      futureClasses[0].ClassLevel,
		ClassName:       futureClasses[0].ClassName,
		CurrentCapacity: futureClasses[0].MaxCapacity,
		MaxCapacity:     futureClasses[0].MaxCapacity,
		Location:        futureClasses[0].Location,
		LocationLink:    "link-b",
	},
	{
		ID:              futureClasses[1].ID,
		StartTime:       futureClasses[1].StartTime,
		ClassLevel:      futureClasses[1].ClassLevel,
		ClassName:       futureClasses[1].ClassName,
		CurrentCapacity: futureClasses[1].MaxCapacity,
		MaxCapacity:     futureClasses[1].MaxCapacity,
		Location:        futureClasses[1].Location,
		LocationLink:    "link-c",
	},
}

var pastAndFutureClasses = []domainModels.Class{
	{
		ID:          classID1,
		StartTime:   pastTime,
		ClassLevel:  "Beginner",
		ClassName:   "Morning Yoga",
		MaxCapacity: 10,
		Location:    "Studio A",
	},
	{
		ID:          classID2,
		StartTime:   futureTime1,
		ClassLevel:  "Intermediate",
		ClassName:   "Afternoon Yoga",
		MaxCapacity: 15,
		Location:    "Studio B",
	},
	{
		ID:          classID3,
		StartTime:   futureTime2,
		ClassLevel:  "Advanced",
		ClassName:   "Evening Yoga",
		MaxCapacity: 12,
		Location:    "Studio C",
	},
	{
		ID:          classID4,
		StartTime:   futureTime3,
		ClassLevel:  "Beginner",
		ClassName:   "Night Yoga",
		MaxCapacity: 20,
		Location:    "Studio D",
	},
}

var pastAndFutureClassPresentations = []ClassPresentation{
	{
		ID:              classID1,
		StartTime:       pastTime,
		ClassLevel:      "Beginner",
		ClassName:       "Morning Yoga",
		CurrentCapacity: 9,
		MaxCapacity:     10,
		Location:        "Studio A",
		LocationLink:    "link",
	},
	{
		ID:              classID2,
		StartTime:       futureTime1,
		ClassLevel:      "Intermediate",
		ClassName:       "Afternoon Yoga",
		CurrentCapacity: 14,
		MaxCapacity:     15,
		Location:        "Studio B",
		LocationLink:    "link",
	},
	{
		ID:              classID3,
		StartTime:       futureTime2,
		ClassLevel:      "Advanced",
		ClassName:       "Evening Yoga",
		CurrentCapacity: 11,
		MaxCapacity:     12,
		Location:        "Studio C",
		LocationLink:    "link",
	},
	{
		ID:              classID4,
		StartTime:       futureTime3,
		ClassLevel:      "Beginner",
		ClassName:       "Night Yoga",
		CurrentCapacity: 19,
		MaxCapacity:     20,
		Location:        "Studio D",
		LocationLink:    "link",
	},
}

var bookingWithPass = domainModels.Booking{
	ID:                uuid.MustParse("7c9b4c3e-2a6f-4b9d-9c8f-6f1a3e0b5d42"),
	ClassID:           classID1,
	Class:             futureClass,
	PassID:            optional.Of(passID1),
	Pass:              optional.Of(pass1),
	FirstName:         "Jan",
	LastName:          "Kowalski",
	Email:             "jan.kowalski@example.com",
	CreatedAt:         time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC),
	ConfirmationToken: "confirm_abc123xyz",
}

var bookingWithoutPass = domainModels.Booking{
	ID:                uuid.MustParse("7c9b4c3e-2a6f-4b9d-9c8f-6f1a3e0b5d42"),
	ClassID:           bookingID2,
	Class:             futureClass,
	PassID:            optional.Empty[int](),
	Pass:              optional.Empty[domainModels.Pass](),
	FirstName:         "Jan",
	LastName:          "Kowalski",
	Email:             "jan.kowalski@example.com",
	CreatedAt:         time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC),
	ConfirmationToken: "confirm_abc123xyz",
}

var pass1 = domainModels.Pass{
	ID:         passID1,
	Email:      "first@test.com",
	TotalSlots: 5,
	UpdatedAt:  time.Now(),
	CreatedAt:  time.Date(2026, 1, 0o7, 12, 0, 0, 0, time.UTC),
}

func TestService_ListClasses(t *testing.T) {
	t.Parallel()

	limitOne := 1
	negativeLimit := -1

	tests := []struct {
		name                string
		onlyUpcomingClasses bool
		classesLimit        *int
		mocks               func(
			classRepo *mock.MockIClasses,
			bookingsRepo *mock.MockIBookings,
			locationLinkProvider *mock.MockILinkProvider,
		)
		want          []ClassPresentation
		wantError     bool
		errorContains string
	}{
		{
			name:                "List one class",
			onlyUpcomingClasses: false,
			classesLimit:        nil,
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classRepo.EXPECT().
					List(gomock.Any()).
					Return([]domainModels.Class{futureClass}, nil)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), futureClass.ID).
					Return(2, nil)

				locationLinkProvider.EXPECT().
					GetLink(futureClass.Location).
					Return("link-a", nil)
			},
			want: []ClassPresentation{futureClassPresentation},
		},
		{
			name:                "List classes",
			onlyUpcomingClasses: false,
			classesLimit:        nil,
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classRepo.EXPECT().
					List(gomock.Any()).
					Return(futureClasses, nil)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), futureClasses[0].ID).
					Return(3, nil)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), futureClasses[1].ID).
					Return(5, nil)

				locationLinkProvider.EXPECT().
					GetLink(futureClasses[0].Location).
					Return("link-b", nil)

				locationLinkProvider.EXPECT().
					GetLink(futureClasses[1].Location).
					Return("link-c", nil)
			},
			want: futureClassesPresentations,
		},
		{
			name:                "Only upcoming classes",
			onlyUpcomingClasses: true,
			classesLimit:        nil,
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classRepo.EXPECT().
					List(gomock.Any()).
					Return(pastAndFutureClasses, nil)

				for _, class := range pastAndFutureClasses {
					bookingsRepo.EXPECT().
						CountForClassID(gomock.Any(), class.ID).
						Return(1, nil)

					locationLinkProvider.EXPECT().
						GetLink(class.Location).
						Return("link", nil)
				}
			},
			want: pastAndFutureClassPresentations[1:],
		},
		{
			name:                "Limit classes",
			onlyUpcomingClasses: false,
			classesLimit:        &limitOne,
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classRepo.EXPECT().
					List(gomock.Any()).
					Return(futureClasses, nil)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), futureClasses[0].ID).
					Return(0, nil)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), futureClasses[1].ID).
					Return(0, nil)

				locationLinkProvider.EXPECT().
					GetLink(futureClasses[0].Location).
					Return("link-b", nil)

				locationLinkProvider.EXPECT().
					GetLink(futureClasses[1].Location).
					Return("link-c", nil)
			},
			want: futureClassesPresentationsWithoutBookings[:1],
		},
		{
			name:                "Validation error - negative limit",
			onlyUpcomingClasses: false,
			classesLimit:        &negativeLimit,
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
			) {
			},
			wantError:     true,
			errorContains: "classes_limit",
		},
		{
			name:                "Repository list error",
			onlyUpcomingClasses: false,
			classesLimit:        nil,
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classRepo.EXPECT().
					List(gomock.Any()).
					Return(nil, errors.New("db error"))
			},
			wantError:     true,
			errorContains: "could not get all classes",
		},
		{
			name:                "Repository bookings count error",
			onlyUpcomingClasses: false,
			classesLimit:        nil,
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classRepo.EXPECT().
					List(gomock.Any()).
					Return([]domainModels.Class{futureClass}, nil)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), futureClass.ID).
					Return(0, errors.New("db error"))
			},
			wantError:     true,
			errorContains: "could not get bookings for class",
		},
		{
			name:                "Location resolver error",
			onlyUpcomingClasses: false,
			classesLimit:        nil,
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classRepo.EXPECT().
					List(gomock.Any()).
					Return([]domainModels.Class{futureClass}, nil)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), futureClass.ID).
					Return(0, nil)

				locationLinkProvider.EXPECT().
					GetLink(futureClass.Location).
					Return("", errors.New("maps error"))
			},
			wantError:     true,
			errorContains: "could not get location link",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			classRepo := mock.NewMockIClasses(ctrl)
			bookingsRepo := mock.NewMockIBookings(ctrl)
			locationLinkProvider := mock.NewMockILinkProvider(ctrl)

			tt.mocks(classRepo, bookingsRepo, locationLinkProvider)

			service := NewService(
				classRepo,
				bookingsRepo,
				mock.NewMockIUnitOfWork(ctrl),
				mock.NewMockINotifier(ctrl),
				locationLinkProvider,
				"testDomainAddr",
			)

			result, err := service.ListClasses(
				context.Background(),
				tt.onlyUpcomingClasses,
				tt.classesLimit,
			)

			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.errorContains)
				}

				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("error: %s do not contains %s", err.Error(), tt.errorContains)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("expected: %+v, got %+v", tt.want, result)
			}
		})
	}
}

func TestService_CreateClasses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		newClasses    []domainModels.Class
		mocks         func(classRepo *mock.MockIClasses)
		want          []domainModels.Class
		wantError     bool
		errorContains string
	}{
		{
			name:       "Create one valid class",
			newClasses: []domainModels.Class{futureClass},
			mocks: func(classRepo *mock.MockIClasses) {
				classRepo.EXPECT().List(gomock.Any()).Return([]domainModels.Class{pastClass}, nil)
				classRepo.EXPECT().Insert(gomock.Any(), []domainModels.Class{futureClass}).Return(
					[]domainModels.Class{futureClass},
					nil,
				)
			},
			want: []domainModels.Class{futureClass},
		},
		{
			name:       "Create valid classes",
			newClasses: futureClasses,
			mocks: func(classRepo *mock.MockIClasses) {
				classRepo.EXPECT().List(gomock.Any()).Return([]domainModels.Class{pastClass}, nil)
				classRepo.EXPECT().Insert(gomock.Any(), futureClasses).Return(futureClasses, nil)
			},
			want: futureClasses,
		},
		{
			name:       "Validation error - expired class",
			newClasses: []domainModels.Class{pastClass},
			mocks: func(classRepo *mock.MockIClasses) {
				classRepo.EXPECT().List(gomock.Any()).Return(futureClasses, nil)
			},
			wantError:     true,
			errorContains: "expired",
		},
		{
			name:       "Validation error - all class should start in future",
			newClasses: pastAndFutureClasses,
			mocks: func(classRepo *mock.MockIClasses) {
				classRepo.EXPECT().List(gomock.Any()).Return(futureClasses, nil)
			},
			wantError:     true,
			errorContains: "expired",
		},
		{
			name:       "Validation error - exists class with the same time",
			newClasses: futureClasses,
			mocks: func(classRepo *mock.MockIClasses) {
				classRepo.EXPECT().List(gomock.Any()).Return(futureClasses, nil)
			},
			wantError:     true,
			errorContains: "already exists",
		},
		{
			name:       "Repository list error",
			newClasses: futureClasses,
			mocks: func(classRepo *mock.MockIClasses) {
				classRepo.EXPECT().List(gomock.Any()).
					Return(
						[]domainModels.Class{},
						fmt.Errorf("could not get existing classes: %w", errors.New("db error")),
					)
			},
			wantError:     true,
			errorContains: "could not get existing classes",
		},
		{
			name:       "Repository insert error",
			newClasses: futureClasses,
			mocks: func(classRepo *mock.MockIClasses) {
				classRepo.EXPECT().List(gomock.Any()).Return([]domainModels.Class{pastClass}, nil)
				classRepo.EXPECT().Insert(gomock.Any(), futureClasses).
					Return(
						[]domainModels.Class{},
						fmt.Errorf("could not insert classes: %w", errors.New("db error")),
					)
			},
			wantError:     true,
			errorContains: "could not insert classes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			classRepo := mock.NewMockIClasses(ctrl)
			tt.mocks(classRepo)

			service := NewService(
				classRepo,
				mock.NewMockIBookings(ctrl),
				mock.NewMockIUnitOfWork(ctrl),
				mock.NewMockINotifier(ctrl),
				mock.NewMockILinkProvider(ctrl),
				"testDomainAddr",
			)

			ctx := context.Background()

			result, err := service.CreateClasses(ctx, tt.newClasses)

			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.errorContains)
				}

				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("error: %s do not contains %s", err.Error(), tt.errorContains)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("expected: %v, got %v", tt.want, result)
			}
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestService_UpdateClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		update UpdateClassCommand
		mocks  func(
			classRepo *mock.MockIClasses,
			bookingsRepo *mock.MockIBookings,
			notifier *mock.MockINotifier,
			locationLinkProvider *mock.MockILinkProvider,
		)
		want          ClassData
		wantError     bool
		errorContains string
	}{
		{
			name: "Update class name",
			update: UpdateClassCommand{
				ClassName: ptr("Power Yoga"),
			},
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				updatedClass := futureClass
				updatedClass.ClassName = "Power Yoga"

				classRepo.EXPECT().
					List(gomock.Any()).
					Return([]domainModels.Class{pastClass}, nil)

				classRepo.EXPECT().
					Get(gomock.Any(), futureClass.ID).
					Return(futureClass, nil)

				classRepo.EXPECT().
					Update(
						gomock.Any(),
						futureClass.ID,
						map[string]any{
							"class_name": "Power Yoga",
						},
					).
					Return(updatedClass, nil)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), futureClass.ID).
					Return(2, nil)
			},
			want: ClassData{
				ID:              futureClass.ID,
				StartTime:       futureClass.StartTime,
				ClassLevel:      futureClass.ClassLevel,
				ClassName:       "Power Yoga",
				CurrentCapacity: 3,
				MaxCapacity:     futureClass.MaxCapacity,
				Location:        futureClass.Location,
			},
		},
		{
			name: "Update class start time",
			update: UpdateClassCommand{
				StartTime: ptr(futureTime3),
			},
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				updatedClass := futureClass
				updatedClass.StartTime = futureTime3

				classRepo.EXPECT().
					List(gomock.Any()).
					Return([]domainModels.Class{pastClass}, nil)

				classRepo.EXPECT().
					Get(gomock.Any(), futureClass.ID).
					Return(futureClass, nil)

				classRepo.EXPECT().
					Update(
						gomock.Any(),
						futureClass.ID,
						map[string]any{
							"start_time": futureTime3,
						},
					).
					Return(updatedClass, nil)

				bookingsRepo.EXPECT().
					ListByClassID(gomock.Any(), futureClass.ID).
					Return([]domainModels.Booking{bookingWithoutPass}, nil)

				locationLinkProvider.EXPECT().
					GetLink(updatedClass.Location).
					Return("link-a", nil)

				notifier.EXPECT().
					NotifyClassUpdate(
						gomock.Any(),
						"Wyjątkowo musiałem zmienić czas rozpoczęcia zajęć.",
						gomock.Any(),
					).
					Return(nil)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), futureClass.ID).
					Return(2, nil)
			},
			want: ClassData{
				ID:              futureClass.ID,
				StartTime:       futureTime3,
				ClassLevel:      futureClass.ClassLevel,
				ClassName:       futureClass.ClassName,
				CurrentCapacity: 3,
				MaxCapacity:     futureClass.MaxCapacity,
				Location:        futureClass.Location,
			},
		},
		{
			name: "Update class location",
			update: UpdateClassCommand{
				Location: ptr(otoJogaStudio),
			},
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				updatedClass := futureClass
				updatedClass.Location = otoJogaStudio

				classRepo.EXPECT().
					List(gomock.Any()).
					Return([]domainModels.Class{pastClass}, nil)

				classRepo.EXPECT().
					Get(gomock.Any(), futureClass.ID).
					Return(futureClass, nil)

				classRepo.EXPECT().
					Update(
						gomock.Any(),
						futureClass.ID,
						map[string]any{
							"location": otoJogaStudio,
						},
					).
					Return(updatedClass, nil)

				bookingsRepo.EXPECT().
					ListByClassID(gomock.Any(), futureClass.ID).
					Return([]domainModels.Booking{bookingWithoutPass}, nil)

				locationLinkProvider.EXPECT().
					GetLink(otoJogaStudio).
					Return("link-x", nil)

				notifier.EXPECT().
					NotifyClassUpdate(
						gomock.Any(),
						"Wyjątkowo musiałem zmienić lokalizację zajęć.",
						gomock.Any(),
					).
					Return(nil)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), futureClass.ID).
					Return(2, nil)
			},
			want: ClassData{
				ID:              futureClass.ID,
				StartTime:       futureClass.StartTime,
				ClassLevel:      futureClass.ClassLevel,
				ClassName:       futureClass.ClassName,
				CurrentCapacity: 3,
				MaxCapacity:     futureClass.MaxCapacity,
				Location:        otoJogaStudio,
			},
		},
		{
			name: "Update class start time and location",
			update: UpdateClassCommand{
				StartTime: ptr(futureTime3),
				Location:  ptr(otoJogaStudio),
			},
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				updatedClass := futureClass
				updatedClass.StartTime = futureTime3
				updatedClass.Location = otoJogaStudio

				classRepo.EXPECT().
					List(gomock.Any()).
					Return([]domainModels.Class{pastClass}, nil)

				classRepo.EXPECT().
					Get(gomock.Any(), futureClass.ID).
					Return(futureClass, nil)

				classRepo.EXPECT().
					Update(
						gomock.Any(),
						futureClass.ID,
						map[string]any{
							"start_time": futureTime3,
							"location":   otoJogaStudio,
						},
					).
					Return(updatedClass, nil)

				bookingsRepo.EXPECT().
					ListByClassID(gomock.Any(), futureClass.ID).
					Return([]domainModels.Booking{bookingWithoutPass}, nil)

				locationLinkProvider.EXPECT().
					GetLink(otoJogaStudio).
					Return("link-x", nil)

				notifier.EXPECT().
					NotifyClassUpdate(
						gomock.Any(),
						"Wyjątkowo musiałem zmienić lokalizację i czas rozpoczęcia zajęć.",
						gomock.Any(),
					).
					Return(nil)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), futureClass.ID).
					Return(2, nil)
			},
			want: ClassData{
				ID:              futureClass.ID,
				StartTime:       futureTime3,
				ClassLevel:      futureClass.ClassLevel,
				ClassName:       futureClass.ClassName,
				CurrentCapacity: 3,
				MaxCapacity:     futureClass.MaxCapacity,
				Location:        otoJogaStudio,
			},
		},
		{
			name:   "Validation error - empty update",
			update: UpdateClassCommand{},
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classRepo.EXPECT().
					List(gomock.Any()).
					Return([]domainModels.Class{pastClass}, nil)

				classRepo.EXPECT().
					Get(gomock.Any(), futureClass.ID).
					Return(futureClass, nil)
			},
			wantError:     true,
			errorContains: "no fields to update class",
		},
		{
			name: "Validation error - expired class start time",
			update: UpdateClassCommand{
				StartTime: ptr(pastTime),
			},
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classRepo.EXPECT().
					List(gomock.Any()).
					Return([]domainModels.Class{}, nil)
			},
			wantError:     true,
			errorContains: "expired",
		},
		{
			name: "Validation error - class with the same start time already exists",
			update: UpdateClassCommand{
				StartTime: ptr(futureTime2),
			},
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classRepo.EXPECT().
					List(gomock.Any()).
					Return(futureClasses, nil)
			},
			wantError:     true,
			errorContains: "already exists",
		},
		{
			name: "Repository list error",
			update: UpdateClassCommand{
				ClassName: ptr("Power Yoga"),
			},
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classRepo.EXPECT().
					List(gomock.Any()).
					Return(nil, errors.New("db error"))
			},
			wantError:     true,
			errorContains: "could not get existing classes",
		},
		{
			name: "Repository get not found",
			update: UpdateClassCommand{
				ClassName: ptr("Power Yoga"),
			},
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classRepo.EXPECT().
					List(gomock.Any()).
					Return([]domainModels.Class{pastClass}, nil)

				classRepo.EXPECT().
					Get(gomock.Any(), futureClass.ID).
					Return(domainModels.Class{}, repositoryError.ErrNotFound)
			},
			wantError:     true,
			errorContains: "not found",
		},
		{
			name: "Repository get error",
			update: UpdateClassCommand{
				ClassName: ptr("Power Yoga"),
			},
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classRepo.EXPECT().
					List(gomock.Any()).
					Return([]domainModels.Class{pastClass}, nil)

				classRepo.EXPECT().
					Get(gomock.Any(), futureClass.ID).
					Return(domainModels.Class{}, errors.New("db error"))
			},
			wantError:     true,
			errorContains: "could not get class",
		},
		{
			name: "Repository update error",
			update: UpdateClassCommand{
				ClassName: ptr("Power Yoga"),
			},
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classRepo.EXPECT().
					List(gomock.Any()).
					Return([]domainModels.Class{pastClass}, nil)

				classRepo.EXPECT().
					Get(gomock.Any(), futureClass.ID).
					Return(futureClass, nil)

				classRepo.EXPECT().
					Update(
						gomock.Any(),
						futureClass.ID,
						map[string]any{
							"class_name": "Power Yoga",
						},
					).
					Return(domainModels.Class{}, errors.New("db error"))
			},
			wantError:     true,
			errorContains: "could not update class",
		},
		{
			name: "Repository bookings count error",
			update: UpdateClassCommand{
				ClassName: ptr("Power Yoga"),
			},
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				updatedClass := futureClass
				updatedClass.ClassName = "Power Yoga"

				classRepo.EXPECT().
					List(gomock.Any()).
					Return([]domainModels.Class{pastClass}, nil)

				classRepo.EXPECT().
					Get(gomock.Any(), futureClass.ID).
					Return(futureClass, nil)

				classRepo.EXPECT().
					Update(
						gomock.Any(),
						futureClass.ID,
						map[string]any{
							"class_name": "Power Yoga",
						},
					).
					Return(updatedClass, nil)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), futureClass.ID).
					Return(0, errors.New("db error"))
			},
			wantError:     true,
			errorContains: "could not get bookings for class",
		},
		{
			name: "Repository list bookings error",
			update: UpdateClassCommand{
				StartTime: ptr(futureTime3),
			},
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				updatedClass := futureClass
				updatedClass.StartTime = futureTime3

				classRepo.EXPECT().
					List(gomock.Any()).
					Return([]domainModels.Class{pastClass}, nil)

				classRepo.EXPECT().
					Get(gomock.Any(), futureClass.ID).
					Return(futureClass, nil)

				classRepo.EXPECT().
					Update(
						gomock.Any(),
						futureClass.ID,
						map[string]any{
							"start_time": futureTime3,
						},
					).
					Return(updatedClass, nil)

				bookingsRepo.EXPECT().
					ListByClassID(gomock.Any(), futureClass.ID).
					Return(nil, errors.New("db error"))
			},
			wantError:     true,
			errorContains: "could not get class after update",
		},
		{
			name: "Location resolver error",
			update: UpdateClassCommand{
				Location: ptr(otoJogaStudio),
			},
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				updatedClass := futureClass
				updatedClass.Location = otoJogaStudio

				classRepo.EXPECT().
					List(gomock.Any()).
					Return([]domainModels.Class{pastClass}, nil)

				classRepo.EXPECT().
					Get(gomock.Any(), futureClass.ID).
					Return(futureClass, nil)

				classRepo.EXPECT().
					Update(
						gomock.Any(),
						futureClass.ID,
						map[string]any{
							"location": otoJogaStudio,
						},
					).
					Return(updatedClass, nil)

				bookingsRepo.EXPECT().
					ListByClassID(gomock.Any(), futureClass.ID).
					Return([]domainModels.Booking{bookingWithoutPass}, nil)

				locationLinkProvider.EXPECT().
					GetLink(otoJogaStudio).
					Return("", errors.New("maps error"))
			},
			wantError:     true,
			errorContains: "could not get class after update",
		},
		{
			name: "Notifier error",
			update: UpdateClassCommand{
				StartTime: ptr(futureTime3),
			},
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				updatedClass := futureClass
				updatedClass.StartTime = futureTime3

				classRepo.EXPECT().
					List(gomock.Any()).
					Return([]domainModels.Class{pastClass}, nil)

				classRepo.EXPECT().
					Get(gomock.Any(), futureClass.ID).
					Return(futureClass, nil)

				classRepo.EXPECT().
					Update(
						gomock.Any(),
						futureClass.ID,
						map[string]any{
							"start_time": futureTime3,
						},
					).
					Return(updatedClass, nil)

				bookingsRepo.EXPECT().
					ListByClassID(gomock.Any(), futureClass.ID).
					Return([]domainModels.Booking{bookingWithoutPass}, nil)

				locationLinkProvider.EXPECT().
					GetLink(updatedClass.Location).
					Return("link-a", nil)

				notifier.EXPECT().
					NotifyClassUpdate(
						gomock.Any(),
						"Wyjątkowo musiałem zmienić czas rozpoczęcia zajęć.",
						gomock.Any(),
					).
					Return(errors.New("smtp error"))
			},
			wantError:     true,
			errorContains: "could not get class after update",
		},
		{
			name: "Update class level",
			update: UpdateClassCommand{
				ClassLevel: ptr("Advanced"),
			},
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				updatedClass := futureClass
				updatedClass.ClassLevel = "Advanced"

				classRepo.EXPECT().
					List(gomock.Any()).
					Return([]domainModels.Class{pastClass}, nil)

				classRepo.EXPECT().
					Get(gomock.Any(), futureClass.ID).
					Return(futureClass, nil)

				classRepo.EXPECT().
					Update(
						gomock.Any(),
						futureClass.ID,
						map[string]any{
							"class_level": "Advanced",
						},
					).
					Return(updatedClass, nil)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), futureClass.ID).
					Return(2, nil)
			},
			want: ClassData{
				ID:              futureClass.ID,
				StartTime:       futureClass.StartTime,
				ClassLevel:      "Advanced",
				ClassName:       futureClass.ClassName,
				CurrentCapacity: 3,
				MaxCapacity:     futureClass.MaxCapacity,
				Location:        futureClass.Location,
			},
		},
		{
			name: "Update max capacity",
			update: UpdateClassCommand{
				MaxCapacity: ptr(10),
			},
			mocks: func(
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				updatedClass := futureClass
				updatedClass.MaxCapacity = 10

				classRepo.EXPECT().
					List(gomock.Any()).
					Return([]domainModels.Class{pastClass}, nil)

				classRepo.EXPECT().
					Get(gomock.Any(), futureClass.ID).
					Return(futureClass, nil)

				classRepo.EXPECT().
					Update(
						gomock.Any(),
						futureClass.ID,
						map[string]any{
							"max_capacity": 10,
						},
					).
					Return(updatedClass, nil)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), futureClass.ID).
					Return(2, nil)
			},
			want: ClassData{
				ID:              futureClass.ID,
				StartTime:       futureClass.StartTime,
				ClassLevel:      futureClass.ClassLevel,
				ClassName:       futureClass.ClassName,
				CurrentCapacity: 8,
				MaxCapacity:     10,
				Location:        futureClass.Location,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			classRepo := mock.NewMockIClasses(ctrl)
			bookingsRepo := mock.NewMockIBookings(ctrl)
			notifier := mock.NewMockINotifier(ctrl)
			locationLinkProvider := mock.NewMockILinkProvider(ctrl)

			tt.mocks(
				classRepo,
				bookingsRepo,
				notifier,
				locationLinkProvider,
			)

			service := NewService(
				classRepo,
				bookingsRepo,
				mock.NewMockIUnitOfWork(ctrl),
				notifier,
				locationLinkProvider,
				"testDomainAddr",
			)

			result, err := service.UpdateClass(
				context.Background(),
				futureClass.ID,
				tt.update,
			)

			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.errorContains)
				}

				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf(
						"error: %s does not contain %s",
						err.Error(),
						tt.errorContains,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("expected: %+v, got %+v", tt.want, result)
			}
		})
	}
}

func mockTransaction(
	uow *mock.MockIUnitOfWork,
	classRepo *mock.MockIClasses,
	bookingsRepo *mock.MockIBookings,
	passRepo *mock.MockIPasses,
) {
	uow.EXPECT().
		WithTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			ctx context.Context,
			fn func(repositories.Repositories) error,
		) error {
			return fn(repositories.Repositories{
				Classes:  classRepo,
				Bookings: bookingsRepo,
				Passes:   passRepo,
			})
		})
}

func TestService_DeleteClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		msg   *string
		mocks func(
			uow *mock.MockIUnitOfWork,
			classRepo *mock.MockIClasses,
			bookingsRepo *mock.MockIBookings,
			passRepo *mock.MockIPasses,
			notifier *mock.MockINotifier,
			locationLinkProvider *mock.MockILinkProvider,
		)
		wantError     bool
		errorContains string
	}{
		{
			name: "Delete class without bookings",
			msg:  nil,
			mocks: func(
				uow *mock.MockIUnitOfWork,
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				passRepo *mock.MockIPasses,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				mockTransaction(
					uow,
					classRepo,
					bookingsRepo,
					passRepo,
				)

				bookingsRepo.EXPECT().
					ListByClassID(gomock.Any(), futureClass.ID).
					Return([]domainModels.Booking{}, nil)

				classRepo.EXPECT().
					Delete(gomock.Any(), futureClass.ID).
					Return(nil)
			},
		},
		{
			name: "Delete class with booking",
			msg:  ptr("Class cancelled"),
			mocks: func(
				uow *mock.MockIUnitOfWork,
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				passRepo *mock.MockIPasses,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				mockTransaction(
					uow,
					classRepo,
					bookingsRepo,
					passRepo,
				)

				bookingsRepo.EXPECT().
					ListByClassID(gomock.Any(), futureClass.ID).
					Return([]domainModels.Booking{bookingWithoutPass}, nil)

				locationLinkProvider.EXPECT().
					GetLink(futureClass.Location).
					Return("link-a", nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), bookingWithoutPass.ID).
					Return(nil)

				classRepo.EXPECT().
					Delete(gomock.Any(), futureClass.ID).
					Return(nil)

				notifier.EXPECT().
					NotifyClassCancellation(
						gomock.AssignableToTypeOf(domainModels.NotifierParams{}),
						"Class cancelled",
					).
					Return(nil)
			},
		},
		{
			name: "Delete class with booking and pass",
			msg:  ptr("Class cancelled"),
			mocks: func(
				uow *mock.MockIUnitOfWork,
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				passRepo *mock.MockIPasses,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				mockTransaction(
					uow,
					classRepo,
					bookingsRepo,
					passRepo,
				)

				bookingsRepo.EXPECT().
					ListByClassID(gomock.Any(), futureClass.ID).
					Return([]domainModels.Booking{bookingWithPass}, nil)

				locationLinkProvider.EXPECT().
					GetLink(futureClass.Location).
					Return("link-a", nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), bookingWithPass.ID).
					Return(nil)

				bookingsRepo.EXPECT().
					ListByPassID(gomock.Any(), pass1.ID).
					Return([]domainModels.Booking{bookingWithPass}, nil)

				classRepo.EXPECT().
					Delete(gomock.Any(), futureClass.ID).
					Return(nil)

				notifier.EXPECT().
					NotifyClassCancellation(
						gomock.AssignableToTypeOf(domainModels.NotifierParams{}),
						"Class cancelled",
					).
					Return(nil)
			},
		},
		{
			name: "Validation error - bookings exists but msg is nil",
			msg:  nil,
			mocks: func(
				uow *mock.MockIUnitOfWork,
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				passRepo *mock.MockIPasses,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				mockTransaction(
					uow,
					classRepo,
					bookingsRepo,
					passRepo,
				)

				bookingsRepo.EXPECT().
					ListByClassID(gomock.Any(), futureClass.ID).
					Return([]domainModels.Booking{bookingWithoutPass}, nil)
			},
			wantError:     true,
			errorContains: "reason msg can not be empty",
		},
		{
			name: "Repository list bookings error",
			msg:  ptr("Cancelled"),
			mocks: func(
				uow *mock.MockIUnitOfWork,
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				passRepo *mock.MockIPasses,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				mockTransaction(
					uow,
					classRepo,
					bookingsRepo,
					passRepo,
				)

				bookingsRepo.EXPECT().
					ListByClassID(gomock.Any(), futureClass.ID).
					Return(nil, errors.New("db error"))
			},
			wantError:     true,
			errorContains: "delete class transaction failed",
		},
		{
			name: "Location resolver error",
			msg:  ptr("Cancelled"),
			mocks: func(
				uow *mock.MockIUnitOfWork,
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				passRepo *mock.MockIPasses,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				mockTransaction(
					uow,
					classRepo,
					bookingsRepo,
					passRepo,
				)

				bookingsRepo.EXPECT().
					ListByClassID(gomock.Any(), futureClass.ID).
					Return([]domainModels.Booking{bookingWithoutPass}, nil)

				locationLinkProvider.EXPECT().
					GetLink(futureClass.Location).
					Return("", errors.New("maps error"))
			},
			wantError:     true,
			errorContains: "delete class transaction failed",
		},
		{
			name: "Repository delete booking error",
			msg:  ptr("Cancelled"),
			mocks: func(
				uow *mock.MockIUnitOfWork,
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				passRepo *mock.MockIPasses,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				mockTransaction(
					uow,
					classRepo,
					bookingsRepo,
					passRepo,
				)

				bookingsRepo.EXPECT().
					ListByClassID(gomock.Any(), futureClass.ID).
					Return([]domainModels.Booking{bookingWithoutPass}, nil)

				locationLinkProvider.EXPECT().
					GetLink(futureClass.Location).
					Return("link", nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), bookingWithoutPass.ID).
					Return(errors.New("db error"))
			},
			wantError:     true,
			errorContains: "delete class transaction failed",
		},
		{
			name: "Repository list bookings by pass error",
			msg:  ptr("Cancelled"),
			mocks: func(
				uow *mock.MockIUnitOfWork,
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				passRepo *mock.MockIPasses,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				mockTransaction(
					uow,
					classRepo,
					bookingsRepo,
					passRepo,
				)

				bookingsRepo.EXPECT().
					ListByClassID(gomock.Any(), futureClass.ID).
					Return([]domainModels.Booking{bookingWithPass}, nil)

				locationLinkProvider.EXPECT().
					GetLink(futureClass.Location).
					Return("link", nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), bookingWithPass.ID).
					Return(nil)

				bookingsRepo.EXPECT().
					ListByPassID(gomock.Any(), pass1.ID).
					Return(nil, errors.New("db error"))
			},
			wantError:     true,
			errorContains: "delete class transaction failed",
		},
		{
			name: "Repository delete class error",
			msg:  ptr("Cancelled"),
			mocks: func(
				uow *mock.MockIUnitOfWork,
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				passRepo *mock.MockIPasses,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				mockTransaction(
					uow,
					classRepo,
					bookingsRepo,
					passRepo,
				)

				bookingsRepo.EXPECT().
					ListByClassID(gomock.Any(), futureClass.ID).
					Return([]domainModels.Booking{}, nil)

				classRepo.EXPECT().
					Delete(gomock.Any(), futureClass.ID).
					Return(errors.New("db error"))
			},
			wantError:     true,
			errorContains: "delete class transaction failed",
		},
		{
			name: "Unit of work error",
			msg:  ptr("Cancelled"),
			mocks: func(
				uow *mock.MockIUnitOfWork,
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				passRepo *mock.MockIPasses,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				uow.EXPECT().
					WithTransaction(gomock.Any(), gomock.Any()).
					Return(errors.New("transaction failed"))
			},
			wantError:     true,
			errorContains: "delete class transaction failed",
		},

		{
			name: "Notifier error",
			msg:  ptr("Cancelled"),
			mocks: func(
				uow *mock.MockIUnitOfWork,
				classRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				passRepo *mock.MockIPasses,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				mockTransaction(
					uow,
					classRepo,
					bookingsRepo,
					passRepo,
				)

				bookingsRepo.EXPECT().
					ListByClassID(gomock.Any(), futureClass.ID).
					Return([]domainModels.Booking{bookingWithoutPass}, nil)

				locationLinkProvider.EXPECT().
					GetLink(futureClass.Location).
					Return("link", nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), bookingWithoutPass.ID).
					Return(nil)

				classRepo.EXPECT().
					Delete(gomock.Any(), futureClass.ID).
					Return(nil)

				notifier.EXPECT().
					NotifyClassCancellation(
						gomock.AssignableToTypeOf(domainModels.NotifierParams{}),
						"Cancelled",
					).
					Return(errors.New("smtp error"))
			},
			wantError:     true,
			errorContains: "smtp error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uow := mock.NewMockIUnitOfWork(ctrl)
			classRepo := mock.NewMockIClasses(ctrl)
			bookingsRepo := mock.NewMockIBookings(ctrl)
			passRepo := mock.NewMockIPasses(ctrl)
			notifier := mock.NewMockINotifier(ctrl)
			locationLinkProvider := mock.NewMockILinkProvider(ctrl)

			tt.mocks(
				uow,
				classRepo,
				bookingsRepo,
				passRepo,
				notifier,
				locationLinkProvider,
			)

			service := NewService(
				classRepo,
				bookingsRepo,
				uow,
				notifier,
				locationLinkProvider,
				"testDomainAddr",
			)

			err := service.DeleteClass(
				context.Background(),
				futureClass.ID,
				tt.msg,
			)

			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.errorContains)
				}

				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
