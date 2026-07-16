package createclasses

import (
	"net/http"

	"main/internal/domain/models"
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
	var createClassesRequest []dto.CreateClassRequest

	err := ginCtx.ShouldBindJSON(&createClassesRequest)
	if err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	classes := make([]models.Class, 0, len(createClassesRequest))

	for _, dtoClass := range createClassesRequest {
		class := models.Class{
			ID:          uuid.New(),
			StartTime:   dtoClass.StartTime.UTC(),
			ClassLevel:  dtoClass.ClassLevel,
			ClassName:   dtoClass.ClassName,
			MaxCapacity: dtoClass.MaxCapacity,
			Location:    dtoClass.Location,
		}

		classes = append(classes, class)
	}

	ctx := ginCtx.Request.Context()

	createdClasses, err := h.classesService.CreateClasses(ctx, classes)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, err)

		return
	}

	response, err := dto.ToClassesResponse(createdClasses)
	if err != nil {
		ginCtx.JSON(http.StatusInternalServerError, gin.H{"error": "DTOResponse: " + err.Error()})

		return
	}

	ginCtx.JSON(http.StatusCreated, response)
}
