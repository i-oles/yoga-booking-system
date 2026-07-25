package emails

import (
	"net/http"
	"strings"

	"main/internal/infrastructure/sender/memory"

	"github.com/gin-gonic/gin"
)

type handler struct {
	storage *memory.Storage
}

func NewHandler(storage *memory.Storage) *handler {
	return &handler{storage: storage}
}

func (h *handler) Handle(ctx *gin.Context) {
	var builder strings.Builder

	for _, view := range h.storage.GetViews() {
		builder.WriteString(view)
	}

	ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(builder.String()))
}
