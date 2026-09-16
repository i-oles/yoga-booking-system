package viewerrs

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

type IErrorHandler interface {
	Handle(ctx *gin.Context, tmplName string, err error)
}

func HandleError(ctx *gin.Context, err error, statusCode int) {
	slog.Error("HandlerError",
		slog.String("error", err.Error()),
		slog.Any("params", ctx.Request.URL.Query()),
		slog.String("endpoint", ctx.FullPath()),
	)

	if ctx.GetHeader("HX-Request") == "true" {
		ctx.Header("HX-Redirect", "/error")
		ctx.Status(statusCode)

		return
	}

	ctx.HTML(statusCode, "err.tmpl", gin.H{
		"Error": "error_id: " + ctx.GetString("request_id"),
	})
}
