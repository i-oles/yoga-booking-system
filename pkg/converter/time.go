package converter

import (
	"fmt"
	"time"

	"main/pkg/translator"
)

const (
	// DateLayout should not be changed, it can cause an error.
	DateLayout = "02-01-2006"
	HourLayout = "15:04"
	// DateTimeLayout is a local wall-clock layout with no UTC offset -
	// the offset is derived from Europe/Warsaw instead of being supplied by the caller.
	DateTimeLayout = "2006-01-02T15:04:05"
)

func warsawLocation() (*time.Location, error) {
	loc, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		return nil, fmt.Errorf("error while loading location: %w", err)
	}

	return loc, nil
}

func ConvertToWarsawTime(t time.Time) (time.Time, error) {
	loc, err := warsawLocation()
	if err != nil {
		return time.Time{}, err
	}

	return t.In(loc), nil
}

func ParseWarsawTime(layout, value string) (time.Time, error) {
	loc, err := warsawLocation()
	if err != nil {
		return time.Time{}, err
	}

	t, err := time.ParseInLocation(layout, value, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("error while parsing warsaw time: %w", err)
	}

	return t.UTC(), nil
}

func ConvertClassTime(startTime time.Time) (string, string, string, error) {
	warsawTime, err := ConvertToWarsawTime(startTime)
	if err != nil {
		return "", "", "", fmt.Errorf("error while converting time to warsaw time: %w", err)
	}

	weekday, err := translator.TranslateToWeekDayToPolish(warsawTime.Weekday())
	if err != nil {
		return "", "", "", fmt.Errorf("error while translating week day to polish: %w", err)
	}

	return weekday,
		warsawTime.Format(DateLayout),
		warsawTime.Format(HourLayout),
		nil
}
