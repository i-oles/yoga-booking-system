package notifier

import "main/internal/domain/models"

type BaseTmplData struct {
	RecipientFirstName string
	ClassName          string
	ClassLevel         string
	WeekDay            string
	Hour               string
	Date               string
	Location           string
	LocationLink       string
	Signature          string
}

type ClassUpdateTmplData struct {
	BaseTmplData     BaseTmplData
	CancellationLink string
	Message          string
}

type ClassCancellationTmplData struct {
	BaseTmplData  BaseTmplData
	Message       string
	PassID        *int
	PassSlotsView []PassSlotView
}

type BookingConfirmationTmplData struct {
	BaseTmplData     BaseTmplData
	CancellationLink string
	PassID           *int
	PassSlotsView    []PassSlotView
}

type BookingCancellationTmplData struct {
	BaseTmplData  BaseTmplData
	PassID        *int
	PassSlotsView []PassSlotView
}

type BookingReminderTmplData struct {
	BaseTmplData     BaseTmplData
	CancellationLink string
	PassID           *int
	PassSlotsView    []PassSlotView
}

type BookingConfirmationRequestTmplData struct {
	RecipientFirstName string
	ConfirmationLink   string
	Signature          string
}

type PassActivationTmplData struct {
	PassID        int
	PassSlotsView []PassSlotView
	Signature     string
}

type PassSlotView struct {
	Status         models.PassSlotStatus
	ClassStartDate string
}
