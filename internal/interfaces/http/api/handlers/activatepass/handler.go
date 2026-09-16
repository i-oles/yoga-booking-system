package activatepass

import (
	"fmt"
	"net/http"

	"main/internal/application/passes"
	domainErrs "main/internal/domain/errs/api"
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
	var req dto.ActivatePassRequest

	err := ginCtx.ShouldBindJSON(&req)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, domainErrs.ErrValidation(err))

		return
	}

	passActivation, err := h.passesService.ActivatePass(
		ginCtx.Request.Context(),
		req.Email,
		req.InitialAssignedSlots,
		req.TotalSlots,
	)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, err)

		return
	}

	passActivationResp, err := dto.ToPassActivationResp(passActivation)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, fmt.Errorf("DTOResponse: %w", err))

		return
	}

	ginCtx.JSON(http.StatusOK, passActivationResp)
}
