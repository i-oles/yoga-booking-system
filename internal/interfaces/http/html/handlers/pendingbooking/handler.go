package pendingbooking

import (
	"net/http"
	"strings"

	"main/internal/application/pendingbookings"
	"main/internal/domain/models"
	"main/internal/interfaces/http/html/dto"
	viewErrs "main/internal/interfaces/http/html/errs"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type handler struct {
	PendingBookingsService pendingbookings.IService
	ViewErrorHandler       viewErrs.IErrorHandler
}

func NewHandler(
	pendingBookingsService pendingbookings.IService,
	viewErrorHandler viewErrs.IErrorHandler,
) *handler {
	return &handler{
		PendingBookingsService: pendingBookingsService,
		ViewErrorHandler:       viewErrorHandler,
	}
}

// TODO: check if you really need to pass ClassID to template here.
func (h *handler) Handle(ginCtx *gin.Context) {
	var form dto.PendingBookingForm
	if err := ginCtx.ShouldBind(&form); err != nil {
		viewErrs.HandleError(ginCtx, err, http.StatusBadRequest)

		return
	}

	classID, err := uuid.Parse(form.ClassID)
	if err != nil {
		viewErrs.HandleError(ginCtx, err, http.StatusBadRequest)

		return
	}

	pendingBookingParams := models.PendingBookingParams{
		ClassID:   classID,
		FirstName: form.FirstName,
		LastName:  form.LastName,
		Email:     strings.ToLower(form.Email),
	}

	ctx := ginCtx.Request.Context()

	err = h.PendingBookingsService.CreatePendingBooking(ctx, pendingBookingParams)
	if err != nil {
		h.ViewErrorHandler.Handle(ginCtx, "pending_booking_form.tmpl", err)

		return
	}

	ginCtx.HTML(http.StatusOK, "pending_booking.tmpl", gin.H{"ClassID": classID})
}
