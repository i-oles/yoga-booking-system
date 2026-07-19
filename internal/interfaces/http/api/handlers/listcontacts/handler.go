package listcontacts

import (
	"net/http"

	"main/internal/domain/repositories"
	"main/internal/interfaces/http/api/dto"
	apiErrs "main/internal/interfaces/http/api/errs"

	"github.com/gin-gonic/gin"
)

type handler struct {
	contactsRepo    repositories.IContacts
	apiErrorHandler apiErrs.IErrorHandler
}

func NewHandler(
	contactsRepo repositories.IContacts,
	apiErrorHandler apiErrs.IErrorHandler,
) *handler {
	return &handler{
		contactsRepo:    contactsRepo,
		apiErrorHandler: apiErrorHandler,
	}
}

func (h *handler) Handle(ginCtx *gin.Context) {
	ctx := ginCtx.Request.Context()

	allContacts, err := h.contactsRepo.List(ctx)
	if err != nil {
		h.apiErrorHandler.Handle(ginCtx, err)

		return
	}

	contactsResp := dto.ToContactsDTO(allContacts)

	ginCtx.JSON(http.StatusOK, contactsResp)
}
