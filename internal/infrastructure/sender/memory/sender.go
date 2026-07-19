package memory

import (
	"main/internal/infrastructure/sender"
)

type Sender struct {
	storage *Storage
}

func NewSender(
	storage *Storage,
) *Sender {
	return &Sender{
		storage: storage,
	}
}

func (s *Sender) Send(messages ...sender.Message) error {
	for _, m := range messages {
		s.storage.AddView("from: " + m.From + `<hr style="margin:5px 0;">`)
		s.storage.AddView("to: " + m.To + `<hr style="margin:5px 0;">`)
		s.storage.AddView("subject: " + m.Subject + `<hr style="margin:5px 0 40px 0;">`)
		s.storage.AddView(m.Body + `<hr style="margin:40px 0;">`)
	}

	return nil
}
