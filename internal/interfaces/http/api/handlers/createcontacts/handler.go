package createcontacts

import (
	"errors"
	"net/http"

	"main/internal/domain/repositories"
	"main/internal/infrastructure/errs"
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
	var createContactsRequest []dto.CreateContactRequest

	err := ginCtx.ShouldBindJSON(&createContactsRequest)
	if err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	ctx := ginCtx.Request.Context()

	contactsResponse := make([]dto.ContactDTO, 0, len(createContactsRequest))

	for _, contact := range createContactsRequest {
		contact, err := h.contactsRepo.Insert(ctx, contact.Email, contact.FirstName, contact.LastName)
		if err != nil {
			if errors.Is(err, errs.ErrAlreadyExist) {
				continue
			}

			h.apiErrorHandler.Handle(ginCtx, err)

			return
		}

		contactsResponse = append(contactsResponse, dto.ToContactDTO(contact))
	}

	ginCtx.JSON(http.StatusOK, contactsResponse)
}
