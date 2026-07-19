package dto

import (
	"main/internal/domain/models"
)

type CreateContactRequest struct {
	Email     string `binding:"required" json:"email"`
	FirstName string `binding:"required" json:"first_name"`
	LastName  string `binding:"required" json:"last_name"`
}

type ContactDTO struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func ToContactDTO(contact models.Contact) ContactDTO {
	return ContactDTO{
		ID:        contact.ID,
		Email:     contact.Email,
		FirstName: contact.FirstName,
		LastName:  contact.LastName,
	}
}

func ToContactsDTO(contacts []models.Contact) []ContactDTO {
	result := make([]ContactDTO, len(contacts))

	for idx, contact := range contacts {
		result[idx] = ToContactDTO(contact)
	}

	return result
}
