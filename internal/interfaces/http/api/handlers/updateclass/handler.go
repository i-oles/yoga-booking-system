package updateclass

import (
	"net/http"

	appModels "main/internal/application/models"
	"main/internal/domain/services"
	"main/internal/interfaces/http/api/dto"
	apiErrs "main/internal/interfaces/http/api/errs"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type handler struct {
	classesService  services.IClassesService
	apiErrorHandler apiErrs.IErrorHandler
}

func NewHandler(
	classesService services.IClassesService,
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
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	var uri dto.UpdateClassURI

	if err := ginCtx.ShouldBindUri(&uri); err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	parsedUUID, err := uuid.Parse(uri.ClassID)
	if err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	ctx := ginCtx.Request.Context()

	update := appModels.UpdateClassCommand{
		StartTime:   dtoUpdateClass.StartTime,
		ClassLevel:  dtoUpdateClass.ClassLevel,
		ClassName:   dtoUpdateClass.ClassName,
		MaxCapacity: dtoUpdateClass.MaxCapacity,
		Location:    dtoUpdateClass.Location,
	}

	classUpdateCommand, err := h.classesService.UpdateClass(ctx, parsedUUID, update)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, err)

		return
	}

	response, err := dto.ToClassDataResponse(classUpdateCommand)
	if err != nil {
		ginCtx.JSON(http.StatusInternalServerError, gin.H{"error": "DTOResponse: " + err.Error()})

		return
	}

	ginCtx.JSON(http.StatusOK, response)
}
