package listpasses

import (
	"fmt"
	"net/http"

	"main/internal/application/passes"
	"main/internal/interfaces/http/api/dto"
	apiErrs "main/internal/interfaces/http/api/errs"

	"github.com/gin-gonic/gin"
)

type handler struct {
	passesService   passes.IService
	apiErrorHandler apiErrs.IErrorHandler
}

func NewHandler(
	passesService passes.IService,
	apiErrorHandler apiErrs.IErrorHandler,
) *handler {
	return &handler{
		passesService:   passesService,
		apiErrorHandler: apiErrorHandler,
	}
}

func (h *handler) Handle(ginCtx *gin.Context) {
	ctx := ginCtx.Request.Context()

	allPasses, err := h.passesService.ListPasses(ctx)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, err)

		return
	}

	response, err := dto.ToPassPresentationsDTO(allPasses)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, fmt.Errorf("DTOResponse: %w", err))

		return
	}

	ginCtx.JSON(http.StatusOK, response)
}
