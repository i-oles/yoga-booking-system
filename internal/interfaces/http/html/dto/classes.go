package dto

import (
	"fmt"

	"main/internal/application/classes"
	"main/pkg/converter"

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

func ToClassView(class classes.ClassPresentation) (ClassView, error) {
	weekDay, startDate, startHour, err := converter.ConvertClassTime(class.StartTime)
	if err != nil {
		return ClassView{},
			fmt.Errorf("error while converting to class time: %w", err)
	}

	return ClassView{
		ID:              class.ID,
		WeekDay:         weekDay,
		StartDate:       startDate,
		StartHour:       startHour,
		ClassLevel:      class.ClassLevel,
		ClassName:       class.ClassName,
		CurrentCapacity: class.CurrentCapacity,
		MaxCapacity:     class.MaxCapacity,
		Location:        class.Location,
		LocationLink:    class.LocationLink,
	}, nil
}

func ToClassViews(classes []classes.ClassPresentation) ([]ClassView, error) {
	classesView := make([]ClassView, len(classes))

	for idx, class := range classes {
		classView, err := ToClassView(class)
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

func ToBookingCancellationClassView(
	class classes.BookingCancellationClass,
) (BookingCancellationClassView, error) {
	weekDay, startDate, startHour, err := converter.ConvertClassTime(class.StartTime)
	if err != nil {
		return BookingCancellationClassView{},
			fmt.Errorf("error while converting class time: %w", err)
	}

	return BookingCancellationClassView{
		ID:         class.ID,
		WeekDay:    weekDay,
		StartDate:  startDate,
		StartHour:  startHour,
		ClassLevel: class.ClassLevel,
		ClassName:  class.ClassName,
		Location:   class.Location,
	}, nil
}
