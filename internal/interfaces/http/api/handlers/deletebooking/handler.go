package deletebooking

import (
	"net/http"

	"main/internal/application/bookings"
	apiErrs "main/internal/interfaces/http/api/errs"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type handler struct {
	bookingsService bookings.IService
	apiErrorHandler apiErrs.IErrorHandler
}

func NewHandler(
	bookingsService bookings.IService,
	apiErrorHandler apiErrs.IErrorHandler,
) *handler {
	return &handler{
		bookingsService: bookingsService,
		apiErrorHandler: apiErrorHandler,
	}
}

func (h *handler) Handle(ginCtx *gin.Context) {
	bookingIDStr := ginCtx.Param("booking_id")

	bookingID, err := uuid.Parse(bookingIDStr)
	if err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	ctx := ginCtx.Request.Context()

	err = h.bookingsService.DeleteBooking(ctx, bookingID)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, err)

		return
	}

	ginCtx.JSON(http.StatusOK, gin.H{"bookingID": bookingID})
}
