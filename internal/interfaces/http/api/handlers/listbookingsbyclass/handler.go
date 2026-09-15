package listbookingsbyclass

import (
	"fmt"
	"net/http"

	domainErrs "main/internal/domain/errs/api"
	"main/internal/domain/repositories"
	"main/internal/interfaces/http/api/dto"
	apiErrs "main/internal/interfaces/http/api/errs"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	classIDStr := ginCtx.Param("class_id")

	classID, err := uuid.Parse(classIDStr)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, domainErrs.ErrValidation(err))

		return
	}

	ctx := ginCtx.Request.Context()

	allBookingsForClass, err := h.bookingsRepo.ListByClassID(ctx, classID)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, err)

		return
	}

	response, err := dto.ToBookingsListResponse(allBookingsForClass)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, fmt.Errorf("DTOResponse: %w", err))

		return
	}

	ginCtx.JSON(http.StatusOK, response)
}
