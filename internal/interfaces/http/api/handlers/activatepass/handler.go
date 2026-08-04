package activatepass

import (
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
	var req dto.ActivatePassRequest

	err := ginCtx.ShouldBindJSON(&req)
	if err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

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
		ginCtx.JSON(http.StatusInternalServerError, gin.H{"error": "DTOResponse: " + err.Error()})

		return
	}

	ginCtx.JSON(http.StatusOK, passActivationResp)
}
