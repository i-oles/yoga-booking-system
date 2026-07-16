package sender

type Message struct {
	From    string
	To      string
	Subject string
	Body    string
}

type IEmailSender interface {
	Send(messages ...Message) error
}
