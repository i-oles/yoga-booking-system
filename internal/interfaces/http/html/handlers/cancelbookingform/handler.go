package cancelbookingform

import (
	"net/http"

	"main/internal/domain/services"
	"main/internal/interfaces/http/html/dto"
	viewErrs "main/internal/interfaces/http/html/errs"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type handler struct {
	bookingService   services.IBookingsService
	viewErrorHandler viewErrs.IErrorHandler
}

func NewHandler(
	bookingService services.IBookingsService,
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

	bookingCancellationForm, err := h.bookingService.GetBookingCancellationForm(
		ctx,
		bookingID,
		form.Token,
	)
	if err != nil {
		h.viewErrorHandler.Handle(ginCtx, "err.tmpl", err)

		return
	}

	view, err := dto.ToBookingCancellationFormView(bookingCancellationForm)
	if err != nil {
		viewErrs.HandleError(ginCtx, err, http.StatusInternalServerError)

		return
	}

	ginCtx.HTML(http.StatusOK, "cancel_booking_form.tmpl", view)
}
