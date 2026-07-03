package dto

import (
	"fmt"
	"time"

	appModels "main/internal/application/models"
	"main/internal/domain/models"
	"main/pkg/converter"
	"main/pkg/translator"

	"github.com/google/uuid"
)

type CreateClassRequest struct {
	StartTime   time.Time `binding:"required" json:"start_time"  time_format:"2006-01-02T15:04:05Z07:00"` //nolint
	ClassLevel  string    `binding:"required,min=3,max=40" json:"class_level"`
	ClassName   string    `binding:"required,min=3,max=60" json:"class_name"`
	MaxCapacity int       `binding:"gte=1" json:"max_capacity"`
	Location    string    `binding:"required" json:"location"`
}

type ListClassesRequest struct {
	OnlyUpcomingClasses bool `json:"only_upcoming_classes"`
	ClassesLimit        *int `json:"classes_limit"`
}

type DeleteClassRequest struct {
	Message *string `binding:"omitempty,min=1,max=250" json:"message"`
}

type UpdateClassRequest struct {
	StartTime   *time.Time `json:"start_time"`
	ClassLevel  *string    `json:"class_level"`
	ClassName   *string    `json:"class_name"`
	MaxCapacity *int       `json:"max_capacity"`
	Location    *string    `json:"location"`
}

type UpdateClassURI struct {
	ClassID string `binding:"required" uri:"class_id"`
}

type ClassResponse struct {
	ID          uuid.UUID `json:"id"`
	WeekDay     string    `json:"week_day"`
	StartDate   string    `json:"start_date"`
	StartHour   string    `json:"start_hour"`
	ClassLevel  string    `json:"class_level"`
	ClassName   string    `json:"class_name"`
	MaxCapacity int       `json:"max_capacity"`
	Location    string    `json:"location"`
}

func FromClass(class models.Class) (ClassResponse, error) {
	warsawTime, err := converter.ConvertToWarsawTime(class.StartTime)
	if err != nil {
		return ClassResponse{}, fmt.Errorf("error while converting time to warsaw time: %w", err)
	}

	weekday, err := translator.TranslateToWeekDayToPolish(warsawTime.Weekday())
	if err != nil {
		return ClassResponse{}, fmt.Errorf("error while translating week day to polish: %w", err)
	}

	return ClassResponse{
		ID:          class.ID,
		WeekDay:     weekday,
		StartDate:   warsawTime.Format(converter.DateLayout),
		StartHour:   warsawTime.Format(converter.HourLayout),
		ClassLevel:  class.ClassLevel,
		ClassName:   class.ClassName,
		MaxCapacity: class.MaxCapacity,
		Location:    class.Location,
	}, nil
}

func FromClasses(classes []models.Class) ([]ClassResponse, error) {
	classesResponse := make([]ClassResponse, len(classes))

	for idx, class := range classes {
		classResponse, err := FromClass(class)
		if err != nil {
			return nil, fmt.Errorf("could not convert class to classResponse: %w", err)
		}

		classesResponse[idx] = classResponse
	}

	return classesResponse, nil
}

type ClassDataResponse struct {
	ID              uuid.UUID `json:"id"`
	WeekDay         string    `json:"week_day"`
	StartDate       string    `json:"start_date"`
	StartHour       string    `json:"start_hour"`
	ClassLevel      string    `json:"class_level"`
	ClassName       string    `json:"class_name"`
	CurrentCapacity int       `json:"current_capacity"`
	MaxCapacity     int       `json:"max_capacity"`
	Location        string    `json:"location"`
}

func FromClassData(class appModels.ClassData) (ClassDataResponse, error) {
	warsawTime, err := converter.ConvertToWarsawTime(class.StartTime)
	if err != nil {
		return ClassDataResponse{}, fmt.Errorf("error while converting time to warsaw time: %w", err)
	}

	weekday, err := translator.TranslateToWeekDayToPolish(warsawTime.Weekday())
	if err != nil {
		return ClassDataResponse{}, fmt.Errorf("error while translating week day to polish: %w", err)
	}

	return ClassDataResponse{
		ID:              class.ID,
		WeekDay:         weekday,
		StartDate:       warsawTime.Format(converter.DateLayout),
		StartHour:       warsawTime.Format(converter.HourLayout),
		ClassLevel:      class.ClassLevel,
		ClassName:       class.ClassName,
		CurrentCapacity: class.CurrentCapacity,
		MaxCapacity:     class.MaxCapacity,
		Location:        class.Location,
	}, nil
}

func FromClassDatas(classes []appModels.ClassData) ([]ClassDataResponse, error) {
	classItemsResponse := make([]ClassDataResponse, len(classes))

	for idx, item := range classes {
		classResponse, err := FromClassData(item)
		if err != nil {
			return nil, fmt.Errorf("could not convert classListItem to classListResponse: %w", err)
		}

		classItemsResponse[idx] = classResponse
	}

	return classItemsResponse, nil
}

func FromClassPresentation(class appModels.ClassPresentation) (ClassDataResponse, error) {
	warsawTime, err := converter.ConvertToWarsawTime(class.StartTime)
	if err != nil {
		return ClassDataResponse{}, fmt.Errorf("error while converting time to warsaw time: %w", err)
	}

	weekday, err := translator.TranslateToWeekDayToPolish(warsawTime.Weekday())
	if err != nil {
		return ClassDataResponse{}, fmt.Errorf("error while translating week day to polish: %w", err)
	}

	return ClassDataResponse{
		ID:              class.ID,
		WeekDay:         weekday,
		StartDate:       warsawTime.Format(converter.DateLayout),
		StartHour:       warsawTime.Format(converter.HourLayout),
		ClassLevel:      class.ClassLevel,
		ClassName:       class.ClassName,
		CurrentCapacity: class.CurrentCapacity,
		MaxCapacity:     class.MaxCapacity,
		Location:        class.Location,
	}, nil
}

func FromClassPresentations(classes []appModels.ClassPresentation) ([]ClassDataResponse, error) {
	classDatasResponse := make([]ClassDataResponse, len(classes))

	for idx, class := range classes {
		classResponse, err := FromClassPresentation(class)
		if err != nil {
			return nil, fmt.Errorf("could not convert classListItem to classListResponse: %w", err)
		}

		classDatasResponse[idx] = classResponse
	}

	return classDatasResponse, nil
}
