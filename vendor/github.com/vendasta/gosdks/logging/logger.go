package logging

import (
	"golang.org/x/net/context"
)

// Logger provides the methods for logging.
type Logger interface {
	request(ctx context.Context, r *requestData, stream Stream)
	Debugf(ctx context.Context, f string, a ...interface{})
	Infof(ctx context.Context, f string, a ...interface{})
	Warningf(ctx context.Context, f string, a ...interface{})
	Errorf(ctx context.Context, f string, a ...interface{})
	Criticalf(ctx context.Context, f string, a ...interface{})
	Alertf(ctx context.Context, f string, a ...interface{})
	Emergencyf(ctx context.Context, f string, a ...interface{})
	StackTrace(ctx context.Context, f string, a ...interface{})
	Tag(ctx context.Context, key, value string) Logger
	RequestID() string
}

var loggerInstance Logger

// Stream enumerates the gcloud logging streams
type Stream string

const (
	// Request is the gcloud request log stream
	Request Stream = "request"
	// Pubsub is the gcloud pubsub log stream
	Pubsub Stream = "pubsub"
	// Taskqueue is the gcloud task log stream
	Taskqueue Stream = "taskqueue"
)

// GetLogger returns the current Logger instance.
func GetLogger() Logger {
	if loggerInstance == nil {
		loggerInstance = &stderrLogger{config: &config{}}
	}
	return loggerInstance
}

func logRequest(ctx context.Context, r *requestData, stream Stream) {
	GetLogger().request(ctx, r, stream)
}

// Debugf emits a debug log
func Debugf(ctx context.Context, f string, a ...interface{}) {
	GetLogger().Debugf(ctx, f, a...)
}

// Infof emits a info log
func Infof(ctx context.Context, f string, a ...interface{}) {
	GetLogger().Infof(ctx, f, a...)
}

// Warningf emits a warning log
func Warningf(ctx context.Context, f string, a ...interface{}) {
	GetLogger().Warningf(ctx, f, a...)
}

// Errorf emits an error log
func Errorf(ctx context.Context, f string, a ...interface{}) {
	GetLogger().Errorf(ctx, f, a...)
}

// Criticalf emits a critical log
func Criticalf(ctx context.Context, f string, a ...interface{}) {
	GetLogger().Criticalf(ctx, f, a...)
}

// Alertf emits an alert log
func Alertf(ctx context.Context, f string, a ...interface{}) {
	GetLogger().Alertf(ctx, f, a...)
}

// Emergencyf emits an emergency log
func Emergencyf(ctx context.Context, f string, a ...interface{}) {
	GetLogger().Emergencyf(ctx, f, a...)
}

// StackTrace logs a message with a stack trace to the current location
func StackTrace(ctx context.Context, f string, a ...interface{}) {
	GetLogger().StackTrace(ctx, f, a...)
}

// Tag sets a tag on the request
func Tag(ctx context.Context, key, value string) Logger {
	return GetLogger().Tag(ctx, key, value)
}
