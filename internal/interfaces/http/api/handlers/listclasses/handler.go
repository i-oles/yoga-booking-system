package listclasses

import (
	"net/http"

	"main/internal/application/classes"
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

	err := ginCtx.ShouldBindJSON(&listClassRequest)
	if err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

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
		ginCtx.JSON(http.StatusInternalServerError, gin.H{"error": "ClassListResponse: " + err.Error()})

		return
	}

	ginCtx.JSON(http.StatusOK, response)
}
