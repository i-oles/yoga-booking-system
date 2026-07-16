package dto

import (
	"fmt"

	"main/internal/application/models"

	"github.com/google/uuid"
)

type BookingCancelForm struct {
	Token string `form:"token" binding:"required,len=44"`
}

type BookingCancelURI struct {
	BookingID string `uri:"id" binding:"required"`
}

type BookingCreateForm struct {
	Token string `form:"token" binding:"required,len=44"`
}

type BookingCancellationFormView struct {
	Class             BookingCancellationClassView `json:"class"`
	BookingID         uuid.UUID                    `json:"booking_id"`
	ConfirmationToken string                       `json:"confirmation_token"`
}

func ToBookingCancellationFormView(
	form models.BookingCancellationForm,
) (BookingCancellationFormView, error) {
	classEssentialView, err := ToBookingCancellationClassView(form.Class)
	if err != nil {
		return BookingCancellationFormView{},
			fmt.Errorf("could not convert classDetails to classView: %w", err)
	}

	return BookingCancellationFormView{
		Class:             classEssentialView,
		BookingID:         form.BookingID,
		ConfirmationToken: form.ConfirmationToken,
	}, nil
}
