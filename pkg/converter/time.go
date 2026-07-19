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
)

func ConvertToWarsawTime(t time.Time) (time.Time, error) { //nolint
	loc, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		return time.Time{}, fmt.Errorf("error while loading location: %w", err)
	}

	return t.In(loc), nil
}

func ConvertClassTime(startTime time.Time) (weekDay, startDate, startHour string, err error) {
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
