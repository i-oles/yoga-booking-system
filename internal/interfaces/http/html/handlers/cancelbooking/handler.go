package cancelbooking

import (
	"net/http"

	"main/internal/application/bookings"
	"main/internal/interfaces/http/html/dto"
	viewErrs "main/internal/interfaces/http/html/errs"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type handler struct {
	bookingService   bookings.IService
	viewErrorHandler viewErrs.IErrorHandler
}

func NewHandler(
	bookingService bookings.IService,
	viewErrorHandler viewErrs.IErrorHandler,
) *handler {
	return &handler{
		bookingService:   bookingService,
		viewErrorHandler: viewErrorHandler,
	}
}

func (h *handler) Handle(ginCtx *gin.Context) {
	var uri dto.BookingCancelURI

	if err := ginCtx.ShouldBindUri(&uri); err != nil {
		viewErrs.HandleError(ginCtx, err, http.StatusBadRequest)

		return
	}

	var form dto.BookingCancelForm

	if err := ginCtx.ShouldBindQuery(&form); err != nil {
		viewErrs.HandleError(ginCtx, err, http.StatusBadRequest)

		return
	}

	bookingID, err := uuid.Parse(uri.BookingID)
	if err != nil {
		viewErrs.HandleError(ginCtx, err, http.StatusBadRequest)

		return
	}

	ctx := ginCtx.Request.Context()

	err = h.bookingService.CancelBooking(ctx, bookingID, form.Token)
	if err != nil {
		h.viewErrorHandler.Handle(ginCtx, "err.tmpl", err)

		return
	}

	ginCtx.HTML(http.StatusOK, "confirmation_cancel_booking.tmpl", gin.H{})
}
