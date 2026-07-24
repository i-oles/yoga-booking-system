package deleteclass

import (
	"net/http"

	"main/internal/application/classes"
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
	var dtoDeleteClassRequest dto.DeleteClassRequest

	err := ginCtx.ShouldBindJSON(&dtoDeleteClassRequest)
	if err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	classIDStr := ginCtx.Param("class_id")

	classID, err := uuid.Parse(classIDStr)
	if err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	ctx := ginCtx.Request.Context()

	err = h.classesService.DeleteClass(ctx, classID, dtoDeleteClassRequest.Message)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, err)

		return
	}

	ginCtx.JSON(http.StatusOK, gin.H{"class_id": classID})
}
