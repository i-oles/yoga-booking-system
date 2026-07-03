package gmail

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"strings"
	"time"

	"main/internal/domain/models"
	notifierModels "main/internal/infrastructure/models/notifier"
	"main/pkg/converter"
	"main/pkg/translator"

	"gopkg.in/gomail.v2"
)

const PassLabel = "KARNET"

type notifier struct {
	dialer                             *gomail.Dialer
	login                              string
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
	host string,
	port int,
	login string,
	password string,
	signature string,
	baseTmplPath string,
) *notifier {
	dialer := gomail.NewDialer(host, port, login, password)
	dialer.TLSConfig = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         host,
		InsecureSkipVerify: false,
	}

	return &notifier{
		dialer:                             dialer,
		login:                              login,
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

	msgToRecipient, err := n.buildMsgToRecipient(email, subject, tmpl, tmplData)
	if err != nil {
		return fmt.Errorf("could not build msg to recipient %s: %w", email, err)
	}

	if err = n.dialer.DialAndSend(msgToRecipient); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
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

	msgToRecipient, err := n.buildMsgToRecipient(email, subject, tmpl, tmplData)
	if err != nil {
		return fmt.Errorf("could not build msg to recipient %s: %w", email, err)
	}

	if err = n.dialer.DialAndSend(msgToRecipient); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
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

	if err = n.dialer.DialAndSend(msgToRecipient, msgToOwner); err != nil {
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

	if err = n.dialer.DialAndSend(msgToRecipient, msgToOwner); err != nil {
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

	msgToRecipient, err := n.buildMsgToRecipient(params.RecipientEmail, subject, tmpl, tmplData)
	if err != nil {
		return fmt.Errorf("could not build msg to recipient %s: %w", params.RecipientEmail, err)
	}

	if err = n.dialer.DialAndSend(msgToRecipient); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
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

	msgToRecipient, err := n.buildMsgToRecipient(params.RecipientEmail, subject, tmpl, tmplData)
	if err != nil {
		return fmt.Errorf("could not build msg to recipient %s: %w", params.RecipientEmail, err)
	}

	if err = n.dialer.DialAndSend(msgToRecipient); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
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

	msgToRecipient, err := n.buildMsgToRecipient(params.RecipientEmail, subject, tmpl, tmplData)
	if err != nil {
		return fmt.Errorf("could not build msg to recipient %s: %w", params.RecipientEmail, err)
	}

	if err = n.dialer.DialAndSend(msgToRecipient); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (n *notifier) buildMsgToRecipient(
	email,
	subject string,
	tmpl *template.Template,
	tmplData any,
) (*gomail.Message, error) {
	var body strings.Builder

	err := tmpl.Execute(&body, tmplData)
	if err != nil {
		return nil, fmt.Errorf("could not execute template: %w", err)
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", n.login)
	msg.SetHeader("To", email)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", body.String())

	return msg, nil
}

func (n *notifier) buildMsgToOwner(
	status models.OperationStatus,
	recipientFirstName, recipientLastName string,
	passSlotsView []notifierModels.PassSlotView,
	classTimeDetails timeDetails,
) *gomail.Message {
	subject := fmt.Sprintf("%s %s %s",
		recipientFirstName,
		recipientLastName,
		status,
	)

	if !isAllPassSlotsBlank(passSlotsView) {
		assignedSlots := 0

		for _, slot := range passSlotsView {
			if slot.Status == models.Future || slot.Status == models.Past {
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

	msgToOwner := gomail.NewMessage()
	msgToOwner.SetHeader("From", n.login)
	msgToOwner.SetHeader("To", n.login)
	msgToOwner.SetHeader("Subject", subject)
	msgToOwner.SetBody("text/html", msg)

	return msgToOwner
}

func isAllPassSlotsBlank(passSlotsView []notifierModels.PassSlotView) bool {
	for _, passSlotView := range passSlotsView {
		if passSlotView.Status != models.Blank {
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
		LocationName:       params.LocationName,
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
