package updateclass

import (
	"fmt"
	"net/http"

	"main/internal/application/classes"
	domainErrs "main/internal/domain/errs/api"
	"main/internal/interfaces/http/api/dto"
	apiErrs "main/internal/interfaces/http/api/errs"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	var dtoUpdateClass dto.UpdateClassRequest

	err := ginCtx.ShouldBindJSON(&dtoUpdateClass)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, domainErrs.ErrValidation(err))

		return
	}

	var uri dto.UpdateClassURI

	if err := ginCtx.ShouldBindUri(&uri); err != nil {
		h.apiErrorHandler.Handle(ginCtx, domainErrs.ErrValidation(err))

		return
	}

	parsedUUID, err := uuid.Parse(uri.ClassID)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, domainErrs.ErrValidation(err))

		return
	}

	ctx := ginCtx.Request.Context()

	update := classes.UpdateClassCommand{
		StartTime:   dtoUpdateClass.StartTime,
		ClassLevel:  dtoUpdateClass.ClassLevel,
		ClassName:   dtoUpdateClass.ClassName,
		MaxCapacity: dtoUpdateClass.MaxCapacity,
		Location:    dtoUpdateClass.Location,
		Message:     dtoUpdateClass.Message,
	}

	classUpdateCommand, err := h.classesService.UpdateClass(ctx, parsedUUID, update)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, err)

		return
	}

	response, err := dto.ToClassDataResponse(classUpdateCommand)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, fmt.Errorf("DTOResponse: %w", err))

		return
	}

	ginCtx.JSON(http.StatusOK, response)
}
