package listclasses

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"main/internal/application/classes"
	domainErrs "main/internal/domain/errs/api"
	"main/internal/interfaces/http/api/dto"
	apiErrs "main/internal/interfaces/http/api/errs"

	"github.com/gin-gonic/gin"
)

type handler struct {
	classesService  classes.IService
	apiErrorHandler apiErrs.IErrorHandler
}

func NewHandler(
	classesService classes.IService,
	apiErrorHandler apiErrs.IErrorHandler,
) *handler {
	return &handler{
		classesService:  classesService,
		apiErrorHandler: apiErrorHandler,
	}
}

func (h *handler) Handle(ginCtx *gin.Context) {
	var listClassRequest dto.ListClassesRequest

	// browsers cannot send a body on GET, so a missing body falls back to defaults
	err := ginCtx.ShouldBindJSON(&listClassRequest)
	if err != nil && !errors.Is(err, io.EOF) {
		h.apiErrorHandler.Handle(ginCtx, domainErrs.ErrValidation(err))

		return
	}

	ctx := ginCtx.Request.Context()

	classPresentations, err := h.classesService.ListClasses(
		ctx,
		listClassRequest.OnlyUpcomingClasses,
		listClassRequest.ClassesLimit,
	)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, err)

		return
	}

	response, err := dto.ToClassDataResponsesFromPresentations(classPresentations)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, fmt.Errorf("ClassListResponse: %w", err))

		return
	}

	ginCtx.JSON(http.StatusOK, response)
}
