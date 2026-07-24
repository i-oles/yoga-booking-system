package passes

import (
	"sort"
	"time"

	"main/internal/domain/models"
)

func BuildPassSlots(
	bookings []models.Booking,
	totalSlots int,
	now time.Time,
) []models.PassSlot {
	passSlots := make([]models.PassSlot, 0, totalSlots)

	for _, booking := range bookings {
		classStartTime := booking.Class.StartTime

		status := models.FutureStatus

		if classStartTime.Before(now) {
			status = models.PastStatus
		}

		passSlots = append(passSlots, models.PassSlot{
			ClassStartTime: &classStartTime,
			Status:         status,
		})
	}

	for len(passSlots) < totalSlots {
		passSlots = append(passSlots, models.PassSlot{
			Status: models.BlankStatus,
		})
	}

	sort.Slice(passSlots, func(i, j int) bool {
		slot := passSlots[i]
		nextSlot := passSlots[j]

		if slot.Status == models.BlankStatus {
			return false
		}

		if nextSlot.Status == models.BlankStatus {
			return true
		}

		return slot.ClassStartTime.Before(*nextSlot.ClassStartTime)
	})

	return passSlots
}
