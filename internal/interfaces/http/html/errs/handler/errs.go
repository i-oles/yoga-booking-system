package viewErrHandler

import (
	"errors"
	"net/http"

	domainErrs "main/internal/domain/errs/view"

	"github.com/gin-gonic/gin"
)

type errorHandler struct{}

func NewErrorHandler() errorHandler {
	return errorHandler{}
}

func (e errorHandler) Handle(ctx *gin.Context, tmplName string, err error) {
	var businessError *domainErrs.BusinessError
	if errors.As(err, &businessError) {
		e.handleBusinessError(ctx, tmplName, businessError)

		return
	}

	var notificationError *domainErrs.NotificationError
	if errors.As(err, &notificationError) {
		e.handleNotificationError(ctx, tmplName, notificationError)

		return
	}

	e.handleInternalError(ctx)
}

func (e errorHandler) handleBusinessError(
	ctx *gin.Context,
	tmplName string,
	businessError *domainErrs.BusinessError,
) {
	switch businessError.Code {
	case domainErrs.BookingNotFoundCode,
		domainErrs.ClassEmptyCode:
		ctx.HTML(http.StatusNotFound, tmplName, gin.H{
			"ID":    businessError.ClassID,
			"Error": businessError.Message,
		})
	case domainErrs.BookingAlreadyExistsCode,
		domainErrs.ClassExpiredCode,
		domainErrs.ClassFullyBookedCode,
		domainErrs.TooLateToBook:
		ctx.HTML(http.StatusConflict, tmplName, gin.H{
			"ID":    businessError.ClassID,
			"Error": businessError.Message,
		})
	case domainErrs.TooManyPendingBookingsCode:
		ctx.HTML(http.StatusTooManyRequests, tmplName, gin.H{
			"ID":    businessError.ClassID,
			"Error": businessError.Message,
		})
	case domainErrs.PendingBookingNotFoundCode,
		domainErrs.InvalidCancellationLinkCode:
		ctx.HTML(http.StatusNotFound, tmplName, gin.H{
			"Error": businessError.Message,
		})
	case domainErrs.SomeoneBookedClassFasterCode:
		ctx.HTML(http.StatusConflict, tmplName, gin.H{
			"Error": businessError.Message,
		})
	default:
		ctx.HTML(http.StatusInternalServerError, "err.tmpl", gin.H{
			"Error": "error_id: " + ctx.GetString("request_id"),
		})
	}
}

func (e errorHandler) handleNotificationError(
	ctx *gin.Context,
	tmplName string,
	notificationError *domainErrs.NotificationError,
) {
	ctx.HTML(http.StatusInternalServerError, tmplName, gin.H{
		"ID":    notificationError.ClassID,
		"Error": notificationError.Message,
	})
}

func (e errorHandler) handleInternalError(ctx *gin.Context) {
	if ctx.GetHeader("HX-Request") == "true" {
		ctx.Header("HX-Redirect", "/error")
		ctx.Status(http.StatusInternalServerError)

		return
	}

	ctx.HTML(http.StatusInternalServerError, "err.tmpl", gin.H{
		"Error": "error_id: " + ctx.GetString("request_id"),
	})
}
