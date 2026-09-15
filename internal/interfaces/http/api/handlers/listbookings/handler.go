package listbookings

import (
	"fmt"
	"net/http"

	"main/internal/domain/repositories"
	"main/internal/interfaces/http/api/dto"
	apiErrs "main/internal/interfaces/http/api/errs"

	"github.com/gin-gonic/gin"
)

type handler struct {
	bookingsRepo    repositories.IBookings
	apiErrorHandler apiErrs.IErrorHandler
}

func NewHandler(
	bookingsRepo repositories.IBookings,
	apiErrorHandler apiErrs.IErrorHandler,
) *handler {
	return &handler{
		bookingsRepo:    bookingsRepo,
		apiErrorHandler: apiErrorHandler,
	}
}

func (h *handler) Handle(ginCtx *gin.Context) {
	ctx := ginCtx.Request.Context()

	allBookings, err := h.bookingsRepo.List(ctx)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, err)

		return
	}

	bookingsListResponse, err := dto.ToBookingsListResponse(allBookings)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, fmt.Errorf("DTOResponse: %w", err))

		return
	}

	ginCtx.JSON(http.StatusOK, bookingsListResponse)
}
