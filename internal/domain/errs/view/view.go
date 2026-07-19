package view

import (
	"fmt"

	"github.com/google/uuid"
)

const (
	BookingNotFoundCode int = iota
	BookingAlreadyExistsCode
	ClassExpiredCode
	PendingBookingNotFoundCode
	TooManyPendingBookingsCode
	ClassFullyBookedCode
	ClassEmptyCode
	SomeoneBookedClassFasterCode
	InvalidCancellationLinkCode
	TooLateToBook
)

type BusinessError struct {
	ClassID *uuid.UUID
	Code    int
	Message string
	Err     error
}

func (e *BusinessError) Error() string {
	return e.Err.Error()
}

func ErrBookingAlreadyExists(classID uuid.UUID, email string, err error) *BusinessError {
	return &BusinessError{
		ClassID: &classID,
		Code:    BookingAlreadyExistsCode,
		Message: "Wygląda na to, że rezerwacja dla: " + email + " już istnieje. " +
			"Sprawdź skrzynkę mailową, aby znaleźć wcześniejsze potwierdzenie.",
		Err: err,
	}
}

func ErrBookingNotFound(err error) *BusinessError {
	return &BusinessError{
		Code:    BookingNotFoundCode,
		Message: "Nie znaleziono rezerwacji na te zajęcia. Została już wcześniej odwołana albo usunięta.",
		Err:     err,
	}
}

func ErrClassExpired(classID uuid.UUID, err error) *BusinessError {
	return &BusinessError{
		ClassID: &classID,
		Code:    ClassExpiredCode,
		Message: "Rezerwacja niedostępna – zajęcia już się zaczęły albo odbyły.",
		Err:     err,
	}
}

func ErrPendingBookingNotFound(err error) *BusinessError {
	return &BusinessError{
		Code: PendingBookingNotFoundCode,
		Message: "Link potwierdzający rezerwację wygasł bądź został już wykorzystany, " +
			"rozpocznij nową rezerwację lub poszukaj potwierdzania w skrzynkce mailowej.",
		Err: err,
	}
}

func ErrTooManyPendingBookings(classID uuid.UUID, email string, err error) *BusinessError {
	return &BusinessError{
		Code:    TooManyPendingBookingsCode,
		ClassID: &classID,
		Message: fmt.Sprintf("Wyczerpano limit linków potwierdzających dla %s. "+
			"Sprawdź wiadości odebrane lub spam w skrzynce mailowej.", email),
		Err: err,
	}
}

func ErrClassFullyBooked(classID uuid.UUID, err error) *BusinessError {
	return &BusinessError{
		Code:    ClassFullyBookedCode,
		ClassID: &classID,
		Message: "Brak wolnych miejsc na te zajęcia",
		Err:     err,
	}
}

func ErrTooLateToBook(classID uuid.UUID, err error) *BusinessError {
	return &BusinessError{
		Code:    ClassFullyBookedCode,
		ClassID: &classID,
		Message: "Zajęcia nie mają żadnej innej rezerwacji, a do rozpoczęcia zostało zbyt mało czasu by zarezerwować te zajęcia.",
		Err:     err,
	}
}

func ErrSomeoneBookedClassFaster(err error) *BusinessError {
	return &BusinessError{
		Code:    SomeoneBookedClassFasterCode,
		Message: "Ktoś Cię uprzedził... :( Brak wolnych miejsc na te zajęcia.",
		Err:     err,
	}
}

func ErrInvalidCancellationLink(err error) *BusinessError {
	return &BusinessError{
		Code:    InvalidCancellationLinkCode,
		Message: "Link do odwołania rezerwacji wygasł albo jest nieprawidłowy, skontaktuj się ze mną.",
		Err:     err,
	}
}
