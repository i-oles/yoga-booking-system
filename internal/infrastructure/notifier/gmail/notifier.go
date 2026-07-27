package gmail

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"main/internal/domain/models"
	notifierModels "main/internal/infrastructure/models/notifier"
	"main/internal/infrastructure/sender"
	"main/pkg/converter"
	"main/pkg/translator"
)

const PassLabel = "KARNET"

type notifier struct {
	sender                             sender.IEmailSender
	ownerEmail                         string
	bookingConfirmationRequestTmplPath string
	bookingConfirmationTmplPath        string
	classCancellationTmplPath          string
	classUpdateTmplPath                string
	bookingCancellationTmplPath        string
	passActivationTmplPath             string
	classReminderTmplPath              string
	passTmplPath                       string
	classTmplPath                      string
	signature                          string
}

func NewNotifier(
	sender sender.IEmailSender,
	ownerEmail,
	signature,
	baseTmplPath string,
) *notifier {
	return &notifier{
		sender:                             sender,
		ownerEmail:                         ownerEmail,
		signature:                          signature,
		bookingConfirmationRequestTmplPath: baseTmplPath + "booking_confirmation_request.tmpl",
		bookingConfirmationTmplPath:        baseTmplPath + "booking_confirmation.tmpl",
		classCancellationTmplPath:          baseTmplPath + "class_cancellation.tmpl",
		classUpdateTmplPath:                baseTmplPath + "class_update.tmpl",
		bookingCancellationTmplPath:        baseTmplPath + "booking_cancellation.tmpl",
		passActivationTmplPath:             baseTmplPath + "pass_activation.tmpl",
		classReminderTmplPath:              baseTmplPath + "class_reminder.tmpl",
		passTmplPath:                       baseTmplPath + "pass.tmpl",
		classTmplPath:                      baseTmplPath + "class.tmpl",
	}
}

func (n *notifier) NotifyPassActivation(email string, passSlots []models.PassSlot) error {
	tmplData := notifierModels.PassActivationTmplData{
		Signature:     n.signature,
		PassSlotsView: n.getPassSlotsView(passSlots),
	}

	tmpl, err := template.ParseFiles(n.passActivationTmplPath, n.passTmplPath)
	if err != nil {
		return fmt.Errorf("could not parse template: %w", err)
	}

	subject := "Yoga - Twój karnet jest aktywny!"

	err = n.sendMessage(email, subject, tmpl, tmplData)
	if err != nil {
		return fmt.Errorf("could not send msg: %w", err)
	}

	return nil
}

func (n *notifier) NotifyConfirmationLink(
	email, firstName, confirmationLink string, classStartTime time.Time,
) error {
	tmplData := notifierModels.BookingConfirmationRequestTmplData{
		RecipientFirstName: firstName,
		ConfirmationLink:   confirmationLink,
		Signature:          n.signature,
	}

	tmpl, err := template.ParseFiles(n.bookingConfirmationRequestTmplPath)
	if err != nil {
		return fmt.Errorf("could not parse template: %w", err)
	}

	classStartTimeDetails, err := getClassStartTimeDetails(classStartTime)
	if err != nil {
		return fmt.Errorf("could not get class start time details: %w", err)
	}

	subject := fmt.Sprintf("Yoga (%s) - Potwierdź swoją rezerwację!", classStartTimeDetails.startDate)

	err = n.sendMessage(email, subject, tmpl, tmplData)
	if err != nil {
		return fmt.Errorf("could not send msg: %w", err)
	}

	return nil
}

func (n *notifier) NotifyBookingConfirmation(
	params models.NotifierParams, cancellationLink string,
) error {
	classStartTimeDetails, err := getClassStartTimeDetails(params.StartTime)
	if err != nil {
		return fmt.Errorf("could not get class start time details: %w", err)
	}

	baseTmplData := n.getBaseTmplData(params, classStartTimeDetails)

	tmplData := notifierModels.BookingConfirmationTmplData{
		BaseTmplData:     baseTmplData,
		CancellationLink: cancellationLink,
		PassSlotsView:    n.getPassSlotsView(params.PassSlots),
	}

	tmpl, err := template.ParseFiles(n.bookingConfirmationTmplPath, n.passTmplPath, n.classTmplPath)
	if err != nil {
		return fmt.Errorf("could not parse template: %w", err)
	}

	subject := fmt.Sprintf("Yoga (%s) - rezerwacja potwierdzona!", classStartTimeDetails.startDate)

	msgToRecipient, err := n.buildMsgToRecipient(params.RecipientEmail, subject, tmpl, tmplData)
	if err != nil {
		return fmt.Errorf("could not build msg to recipient %s: %w", params.RecipientEmail, err)
	}

	msgToOwner := n.buildMsgToOwner(
		models.StatusBooked,
		params.RecipientFirstName,
		params.RecipientLastName,
		n.getPassSlotsView(params.PassSlots),
		classStartTimeDetails,
	)

	if err = n.sender.Send(msgToRecipient, msgToOwner); err != nil {
		return fmt.Errorf("failed to send emails: %w", err)
	}

	return nil
}

func (n *notifier) NotifyBookingCancellation(params models.NotifierParams) error {
	classStartTimeDetails, err := getClassStartTimeDetails(params.StartTime)
	if err != nil {
		return fmt.Errorf("could not get class start time details: %w", err)
	}

	tmplData := notifierModels.BookingCancellationTmplData{
		BaseTmplData:  n.getBaseTmplData(params, classStartTimeDetails),
		PassSlotsView: n.getPassSlotsView(params.PassSlots),
	}

	tmpl, err := template.ParseFiles(n.bookingCancellationTmplPath, n.passTmplPath, n.classTmplPath)
	if err != nil {
		return fmt.Errorf("could not parse template: %w", err)
	}

	subject := fmt.Sprintf("Yoga (%s) - rezerwacja odwołana!", classStartTimeDetails.startDate)

	msgToRecipient, err := n.buildMsgToRecipient(params.RecipientEmail, subject, tmpl, tmplData)
	if err != nil {
		return fmt.Errorf("could not build msg to recipient %s: %w", params.RecipientEmail, err)
	}

	msgToOwner := n.buildMsgToOwner(
		models.StatusCancelled,
		params.RecipientFirstName,
		params.RecipientLastName,
		n.getPassSlotsView(params.PassSlots),
		classStartTimeDetails,
	)

	if err = n.sender.Send(msgToRecipient, msgToOwner); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (n *notifier) NotifyClassUpdate(
	params models.NotifierParams, msg, cancellationLink string,
) error {
	classStartTimeDetails, err := getClassStartTimeDetails(params.StartTime)
	if err != nil {
		return fmt.Errorf("could not get class start time details: %w", err)
	}

	tmplData := notifierModels.ClassUpdateTmplData{
		BaseTmplData:     n.getBaseTmplData(params, classStartTimeDetails),
		Message:          msg,
		CancellationLink: cancellationLink,
	}

	tmpl, err := template.ParseFiles(n.classUpdateTmplPath, n.classTmplPath)
	if err != nil {
		return fmt.Errorf("could not parse template: %w", err)
	}

	subject := fmt.Sprintf(
		"Yoga (%s) Musiałem wprowadzić zmiany w zajęciach!",
		classStartTimeDetails.startDate,
	)

	err = n.sendMessage(params.RecipientEmail, subject, tmpl, tmplData)
	if err != nil {
		return fmt.Errorf("could not send msg: %w", err)
	}

	return nil
}

func (n *notifier) NotifyClassCancellation(params models.NotifierParams, msg string) error {
	classStartTimeDetails, err := getClassStartTimeDetails(params.StartTime)
	if err != nil {
		return fmt.Errorf("could not get date details: %w", err)
	}

	tmplData := notifierModels.ClassCancellationTmplData{
		BaseTmplData:  n.getBaseTmplData(params, classStartTimeDetails),
		Message:       msg,
		PassSlotsView: n.getPassSlotsView(params.PassSlots),
	}

	tmpl, err := template.ParseFiles(n.classCancellationTmplPath, n.passTmplPath, n.classTmplPath)
	if err != nil {
		return fmt.Errorf("could not parse template: %w", err)
	}

	subject := fmt.Sprintf("Yoga (%s) - zajęcia odwołane!", classStartTimeDetails.startDate)

	err = n.sendMessage(params.RecipientEmail, subject, tmpl, tmplData)
	if err != nil {
		return fmt.Errorf("could not send msg: %w", err)
	}

	return nil
}

func (n *notifier) NotifyBookingReminder(
	params models.NotifierParams, cancellationLink string,
) error {
	classStartTimeDetails, err := getClassStartTimeDetails(params.StartTime)
	if err != nil {
		return fmt.Errorf("could not get class start time details: %w", err)
	}

	tmplData := notifierModels.BookingReminderTmplData{
		BaseTmplData:     n.getBaseTmplData(params, classStartTimeDetails),
		CancellationLink: cancellationLink,
		PassSlotsView:    n.getPassSlotsView(params.PassSlots),
	}

	tmpl, err := template.ParseFiles(n.classReminderTmplPath, n.passTmplPath, n.classTmplPath)
	if err != nil {
		return fmt.Errorf("could not parse template: %w", err)
	}

	subject := fmt.Sprintf("Yoga (%s) - przypomnienie o zajęciach!", classStartTimeDetails.startDate)

	err = n.sendMessage(params.RecipientEmail, subject, tmpl, tmplData)
	if err != nil {
		return fmt.Errorf("could not send msg: %w", err)
	}

	return nil
}

func (n *notifier) sendMessage(
	email string,
	subject string,
	tmpl *template.Template,
	tmplData any,
) error {
	msgToRecipient, err := n.buildMsgToRecipient(email, subject, tmpl, tmplData)
	if err != nil {
		return fmt.Errorf("could not build msg to recipient %s: %w", email, err)
	}

	if err = n.sender.Send(msgToRecipient); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (n *notifier) buildMsgToRecipient(
	email,
	subject string,
	tmpl *template.Template,
	tmplData any,
) (sender.Message, error) {
	var body strings.Builder

	err := tmpl.Execute(&body, tmplData)
	if err != nil {
		return sender.Message{}, fmt.Errorf("could not execute template: %w", err)
	}

	return sender.Message{
		From:    n.ownerEmail,
		To:      email,
		Subject: subject,
		Body:    body.String(),
	}, nil
}

func (n *notifier) buildMsgToOwner(
	status models.OperationStatus,
	recipientFirstName, recipientLastName string,
	passSlotsView []notifierModels.PassSlotView,
	classTimeDetails timeDetails,
) sender.Message {
	subject := fmt.Sprintf("%s %s %s",
		recipientFirstName,
		recipientLastName,
		status,
	)

	if !isAllPassSlotsBlank(passSlotsView) {
		assignedSlots := 0

		for _, slot := range passSlotsView {
			if slot.Status == models.FutureStatus || slot.Status == models.PastStatus {
				assignedSlots++
			}
		}

		subject += fmt.Sprintf(
			" %s: %d/%d", PassLabel, assignedSlots, len(passSlotsView),
		)
	}

	msg := fmt.Sprintf("%s (%s) - %s",
		classTimeDetails.weekDayInPolish,
		classTimeDetails.startDate,
		classTimeDetails.startHour,
	)

	return sender.Message{
		From:    n.ownerEmail,
		To:      n.ownerEmail,
		Subject: subject,
		Body:    msg,
	}
}

func isAllPassSlotsBlank(passSlotsView []notifierModels.PassSlotView) bool {
	for _, passSlotView := range passSlotsView {
		if passSlotView.Status != models.BlankStatus {
			return false
		}
	}

	return true
}

type timeDetails struct {
	startHour       string
	startDate       string
	weekDayInPolish string
}

func getClassStartTimeDetails(t time.Time) (timeDetails, error) {
	timeWarsawUTC, err := converter.ConvertToWarsawTime(t)
	if err != nil {
		return timeDetails{}, fmt.Errorf("could not convert to warsaw time: %w", err)
	}

	weekDayInPolish, err := translator.TranslateToWeekDayToPolish(timeWarsawUTC.Weekday())
	if err != nil {
		return timeDetails{}, fmt.Errorf("could not translate: %s weekday: %w", weekDayInPolish, err)
	}

	startDate := timeWarsawUTC.Format(converter.DateLayout)
	startHour := timeWarsawUTC.Format(converter.HourLayout)

	return timeDetails{
		startHour:       startHour,
		startDate:       startDate,
		weekDayInPolish: weekDayInPolish,
	}, nil
}

func (n *notifier) getBaseTmplData(
	params models.NotifierParams, classStartTimeDetails timeDetails,
) notifierModels.BaseTmplData {
	return notifierModels.BaseTmplData{
		RecipientFirstName: params.RecipientFirstName,
		ClassName:          params.ClassName,
		ClassLevel:         params.ClassLevel,
		Hour:               classStartTimeDetails.startHour,
		WeekDay:            classStartTimeDetails.weekDayInPolish,
		Date:               classStartTimeDetails.startDate,
		Location:           params.Location,
		LocationLink:       params.LocationLink,
		Signature:          n.signature,
	}
}

func (n *notifier) getPassSlotsView(passSlots []models.PassSlot) []notifierModels.PassSlotView {
	passSlotsView := make([]notifierModels.PassSlotView, 0, len(passSlots))

	for _, slot := range passSlots {
		passSlotView := notifierModels.PassSlotView{
			Status: slot.Status,
		}

		if slot.ClassStartTime != nil {
			passSlotView.ClassStartDate = slot.ClassStartTime.Format("02.01")
		}

		passSlotsView = append(passSlotsView, passSlotView)
	}

	return passSlotsView
}
