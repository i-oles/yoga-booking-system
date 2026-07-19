package notifier

import (
	"time"

	"main/internal/domain/models"
)

type INotifier interface {
	NotifyPassActivation(email string, passSlots []models.PassSlot) error
	NotifyConfirmationLink(email, firstName, confirmationLink string, classStartTime time.Time) error
	NotifyBookingConfirmation(params models.NotifierParams, cancellationLink string) error
	NotifyBookingCancellation(params models.NotifierParams) error
	NotifyClassUpdate(params models.NotifierParams, msg, cancellationLink string) error
	NotifyClassCancellation(params models.NotifierParams, msg string) error
	NotifyBookingReminder(params models.NotifierParams, cancellationLink string) error
}
