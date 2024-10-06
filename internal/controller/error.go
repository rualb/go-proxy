package controller

import (
	"go-proxy/internal/service"

	"github.com/labstack/echo/v4"
)

// APIError has a error code and a message.
type APIError struct {
	Code    int
	Message string
}

type ErrorController struct {
	appService service.AppService
}

// NewErrorController is constructor.
func NewErrorController(appService service.AppService) *ErrorController {
	return &ErrorController{appService: appService}
}

// JSONError is cumstomize error handler
func (x *ErrorController) JSONError(err error, c echo.Context) {

	c.Echo().DefaultHTTPErrorHandler(err, c)

	// // TODO show user-frendly html page on internal server error
	// // e.g. error on jwt timeout on signUp set password

	// logger := x.appService.Logger()
	// code := http.StatusInternalServerError
	// msg := http.StatusText(code)

	// if he, ok := err.(*echo.HTTPError); ok {
	// 	code = he.Code
	// 	msg = he.Message.(string)
	// }

	// var apierr APIError
	// apierr.Code = code
	// apierr.Message = msg

	// if !c.Response().Committed {
	// 	if reserr := c.JSON(code, apierr); reserr != nil {
	// 		logger.ZapLogger().Errorf(reserr.Error())
	// 	}
	// }
	// logger.ZapLogger().Debugln(err.Error())
}
