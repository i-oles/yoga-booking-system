package passes

import (
	"reflect"
	"testing"
	"time"

	"main/internal/domain/models"
)

func TestBuildPassSlots(t *testing.T) {
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		bookings   []models.Booking
		totalSlots int
		want       []models.PassSlot
	}{
		{
			name: "creates future slots",
			bookings: []models.Booking{
				{
					Class: models.Class{
						StartTime: now.Add(2 * time.Hour),
					},
				},
				{
					Class: models.Class{
						StartTime: now.Add(4 * time.Hour),
					},
				},
			},
			totalSlots: 2,
			want: []models.PassSlot{
				{
					ClassStartTime: ptrTime(now.Add(2 * time.Hour)),
					Status:         models.FutureStatus,
				},
				{
					ClassStartTime: ptrTime(now.Add(4 * time.Hour)),
					Status:         models.FutureStatus,
				},
			},
		},
		{
			name: "marks past and future bookings and sorts them",
			bookings: []models.Booking{
				{
					Class: models.Class{
						StartTime: now.Add(3 * time.Hour),
					},
				},
				{
					Class: models.Class{
						StartTime: now.Add(-2 * time.Hour),
					},
				},
				{
					Class: models.Class{
						StartTime: now.Add(1 * time.Hour),
					},
				},
			},
			totalSlots: 3,
			want: []models.PassSlot{
				{
					ClassStartTime: ptrTime(now.Add(-2 * time.Hour)),
					Status:         models.PastStatus,
				},
				{
					ClassStartTime: ptrTime(now.Add(1 * time.Hour)),
					Status:         models.FutureStatus,
				},
				{
					ClassStartTime: ptrTime(now.Add(3 * time.Hour)),
					Status:         models.FutureStatus,
				},
			},
		},
		{
			name: "adds blank slots when total slots are not filled",
			bookings: []models.Booking{
				{
					Class: models.Class{
						StartTime: now.Add(time.Hour),
					},
				},
			},
			totalSlots: 3,
			want: []models.PassSlot{
				{
					ClassStartTime: ptrTime(now.Add(time.Hour)),
					Status:         models.FutureStatus,
				},
				{
					Status: models.BlankStatus,
				},
				{
					Status: models.BlankStatus,
				},
			},
		},
		{
			name:       "returns only blank slots when there are no bookings",
			bookings:   []models.Booking{},
			totalSlots: 2,
			want: []models.PassSlot{
				{
					Status: models.BlankStatus,
				},
				{
					Status: models.BlankStatus,
				},
			},
		},
		{
			name: "keeps blank slots at the end",
			bookings: []models.Booking{
				{
					Class: models.Class{
						StartTime: now.Add(5 * time.Hour),
					},
				},
				{
					Class: models.Class{
						StartTime: now.Add(-time.Hour),
					},
				},
			},
			totalSlots: 4,
			want: []models.PassSlot{
				{
					ClassStartTime: ptrTime(now.Add(-time.Hour)),
					Status:         models.PastStatus,
				},
				{
					ClassStartTime: ptrTime(now.Add(5 * time.Hour)),
					Status:         models.FutureStatus,
				},
				{
					Status: models.BlankStatus,
				},
				{
					Status: models.BlankStatus,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildPassSlots(
				tt.bookings,
				tt.totalSlots,
				now,
			)

			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("expected: %+v, got %+v", tt.want, result)
			}
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
