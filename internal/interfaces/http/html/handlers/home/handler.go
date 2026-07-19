package home

import (
	"net/http"

	"main/internal/domain/services"
	"main/internal/interfaces/http/html/dto"
	viewErrs "main/internal/interfaces/http/html/errs"

	"github.com/gin-gonic/gin"
)

const classViewLimit = 4

type handler struct {
	classesService   services.IClassesService
	viewErrorHandler viewErrs.IErrorHandler
	isVacation       bool
}

func NewHandler(
	classesService services.IClassesService,
	viewErrorHandler viewErrs.IErrorHandler,
	isVacation bool,
) *handler {
	return &handler{
		classesService:   classesService,
		viewErrorHandler: viewErrorHandler,
		isVacation:       isVacation,
	}
}

func (h *handler) Handle(ginCtx *gin.Context) {
	ctx := ginCtx.Request.Context()

	limit := classViewLimit

	classItems, err := h.classesService.ListClasses(ctx, true, &limit)
	if err != nil {
		h.viewErrorHandler.Handle(ginCtx, "err.tmpl", err)

		return
	}

	views, err := dto.ToClassViews(classItems)
	if err != nil {
		viewErrs.HandleError(ginCtx, err, http.StatusInternalServerError)

		return
	}

	ginCtx.HTML(http.StatusOK, "index.html", gin.H{
		"Classes":    views,
		"IsVacation": h.isVacation,
	})
}
