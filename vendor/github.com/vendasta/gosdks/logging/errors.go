package logging

import (
	"github.com/vendasta/gosdks/util"
	"golang.org/x/net/context"
)

// ErrorWithTrace returns a ServiceError and logs its stack trace
func ErrorWithTrace(ctx context.Context, errorType util.ErrorType, format string, a ...interface{}) util.ServiceError {
	StackTrace(ctx, format, a...)
	return util.Error(errorType, format, a...)
}
