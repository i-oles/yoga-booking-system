package createbooking

import (
	"net/http"

	"main/internal/application/bookings"
	"main/internal/interfaces/http/html/dto"
	viewErrs "main/internal/interfaces/http/html/errs"

	"github.com/gin-gonic/gin"
)

type handler struct {
	bookingsService  bookings.IService
	viewErrorHandler viewErrs.IErrorHandler
}

func NewHandler(
	bookingService bookings.IService,
	viewErrorHandler viewErrs.IErrorHandler,
) *handler {
	return &handler{
		bookingsService:  bookingService,
		viewErrorHandler: viewErrorHandler,
	}
}

func (h *handler) Handle(ginCtx *gin.Context) {
	var form dto.BookingCreateForm
	if err := ginCtx.ShouldBindQuery(&form); err != nil {
		viewErrs.HandleError(ginCtx, err, http.StatusBadRequest)

		return
	}

	ctx := ginCtx.Request.Context()

	bookingCreation, err := h.bookingsService.CreateBooking(ctx, form.Token)
	if err != nil {
		h.viewErrorHandler.Handle(ginCtx, "err.tmpl", err)

		return
	}

	view, err := dto.ToClassView(bookingCreation.Class)
	if err != nil {
		viewErrs.HandleError(ginCtx, err, http.StatusInternalServerError)

		return
	}

	ginCtx.HTML(http.StatusOK, "confirmation_create_booking.tmpl", view)
}
