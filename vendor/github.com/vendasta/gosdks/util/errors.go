package util

import (
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorType is an enum encapsulating the spectrum of all possible types of errors raised by the application
type ErrorType int64

const (
	// NotFound corresponds to errors caused by missing entities
	NotFound ErrorType = 1 + iota
	// InvalidArgument corresponds to errors caused by missing or malformed arguments supplied by a client
	InvalidArgument
	// AlreadyExists corresponds to errors caused by an entity already existing
	AlreadyExists
	// PermissionDenied corresponds to a user not having permission to access a resource.
	PermissionDenied
	// Unauthenticated indicates the request does not have valid authentication credentials for the operation.
	Unauthenticated
	// Unimplemented corresponds to a function that is unimplemented
	Unimplemented
	// Unknown Error occurred
	Unknown
	// Internal Error
	Internal
	// Unavailable error occurred
	Unavailable
	// FailedPrecondition indicates operation was rejected because the
	// system is not in a state required for the operation's execution.
	// For example, directory to be deleted may be non-empty, an rmdir
	// operation is applied to a non-directory, etc.
	//
	// A litmus test that may help a service implementor in deciding
	// between FailedPrecondition, Aborted, and Unavailable:
	//  (a) Use Unavailable if the client can retry just the failing call.
	//  (b) Use Aborted if the client should retry at a higher-level
	//      (e.g., restarting a read-modify-write sequence).
	//  (c) Use FailedPrecondition if the client should not retry until
	//      the system state has been explicitly fixed.  E.g., if an "rmdir"
	//      fails because the directory is non-empty, FailedPrecondition
	//      should be returned since the client should not retry unless
	//      they have first fixed up the directory by deleting files from it.
	//  (d) Use FailedPrecondition if the client performs conditional
	//      REST Get/Update/Delete on a resource and the resource on the
	//      server does not match the condition. E.g., conflicting
	//      read-modify-write on the same resource.
	FailedPrecondition
	// DeadlineExceeded means operation expired before completion.
	// For operations that change the state of the system, this error may be
	// returned even if the operation has completed successfully. For
	// example, a successful response from a server could have been delayed
	// long enough for the deadline to expire.
	DeadlineExceeded
	// ResourceExhausted indicates some resource has been exhausted, perhaps
	// a per-user quota, or perhaps the entire file system is out of space.
	ResourceExhausted
	// Aborted indicates the operation was aborted, typically due to a
	// concurrency issue like sequencer check failures, transaction aborts,
	// etc.
	//
	// See litmus test above for deciding between FailedPrecondition,
	// Aborted, and Unavailable.
	Aborted
)

func (errType ErrorType) String() string {
	switch errType {
	case NotFound:
		return "NotFound"
	case InvalidArgument:
		return "InvalidArgument"
	case AlreadyExists:
		return "AlreadyExists"
	case PermissionDenied:
		return "PermissionDenied"
	case Unauthenticated:
		return "Unauthenticated"
	case Unimplemented:
		return "Unimplemented"
	case Unknown:
		return "Unknown"
	case Internal:
		return "Internal"
	case Unavailable:
		return "Unavailable"
	case FailedPrecondition:
		return "FailedPrecondition"
	case DeadlineExceeded:
		return "DeadlineExceeded"
	case ResourceExhausted:
		return "ResourceExhausted"
	case Aborted:
		return "Aborted"
	default:
		return "Unknown"
	}
}

// ServiceError is an error that can be translated to a GRPC-compliant error
type ServiceError struct {
	msg     string
	errType ErrorType
}

// Error returns the message associated with this error
func (v ServiceError) Error() string {
	return v.msg
}

// ErrorType returns the ErrorType associated with this error
func (v ServiceError) ErrorType() ErrorType {
	return v.errType
}

// GRPCError returns an error in a format such that it can be consumed by GRPC
func (v ServiceError) GRPCError() error {
	grpcCode := ErrorTypeToGRPCCode(v.errType)
	if grpcCode == codes.Unknown {
		return status.Errorf(codes.Unknown, "Unknown server error.")
	}
	return status.Errorf(grpcCode, v.msg)
}

// HTTPCode returns the corresponding http status code for a given error
func (v ServiceError) HTTPCode() int {
	switch v.errType {
	case NotFound:
		return http.StatusNotFound
	case InvalidArgument:
		return http.StatusBadRequest
	case AlreadyExists:
		return http.StatusConflict
	case PermissionDenied:
		return http.StatusForbidden
	case Unauthenticated:
		return http.StatusUnauthorized
	case Unimplemented:
		return http.StatusNotImplemented
	case FailedPrecondition:
		return http.StatusPreconditionFailed
	case DeadlineExceeded:
		return http.StatusRequestTimeout
	case ResourceExhausted:
		return http.StatusTooManyRequests
	case Unavailable:
		return http.StatusServiceUnavailable
	case Aborted:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// Error returns a ServiceError
func Error(errorType ErrorType, format string, a ...interface{}) ServiceError {
	return ServiceError{msg: fmt.Sprintf(format, a...), errType: errorType}
}

// FromError given an error tries to return a proper ServiceError.
func FromError(err error) ServiceError {
	statusErr, ok := status.FromError(err)
	if ok {
		return Error(GRPCCodeToErrorType(statusErr.Code()), statusErr.Message())
	}

	serviceError, ok := err.(ServiceError)
	if ok {
		return serviceError
	}
	return Error(Unknown, "Unknown server error.")
}

// IsError returns true/false if the given err matches the errorType type.
func IsError(errorType ErrorType, err error) bool {
	statusErr, ok := status.FromError(err)
	if ok {
		return GRPCCodeToErrorType(statusErr.Code()) == errorType
	}

	serviceError, ok := err.(ServiceError)
	if ok {
		return serviceError.errType == errorType
	}
	return false
}

// ToGrpcError calculates the correct GRPC error code for a ServiceError or existing GRPC error and returns it
// All errors that are not GRPC errors or ServiceErrors will be interpreted as Unknown errors
func ToGrpcError(err error) error {
	// if this is already a GRPC error, pass through
	grpcErr, ok := status.FromError(err)
	if ok {
		return grpcErr.Err()
	}
	// otherwise map to ServiceError
	return FromError(err).GRPCError()
}

//Convert a http error into a grpc error
func StatusCodeToGRPCError(statusCode int) ErrorType {
	switch statusCode {
	case 400:
		return InvalidArgument
	case 401:
		return Unauthenticated
	case 403:
		return PermissionDenied
	case 404:
		return NotFound
	case 409:
		return AlreadyExists
	case 412:
		return FailedPrecondition
	case 429:
		return ResourceExhausted
	case 501:
		return Unimplemented
	case 503:
		return Unavailable
	default:
		return Internal
	}
}

//GRPCCodeToErrorType converts a grpc status code into the matching ErrorType or Unknown
func GRPCCodeToErrorType(statusCode codes.Code) ErrorType {
	switch statusCode {
	case codes.NotFound:
		return NotFound
	case codes.InvalidArgument:
		return InvalidArgument
	case codes.AlreadyExists:
		return AlreadyExists
	case codes.PermissionDenied:
		return PermissionDenied
	case codes.Unauthenticated:
		return Unauthenticated
	case codes.Unimplemented:
		return Unimplemented
	case codes.Unknown:
		return Unknown
	case codes.Internal:
		return Internal
	case codes.Unavailable:
		return Unavailable
	case codes.FailedPrecondition:
		return FailedPrecondition
	case codes.DeadlineExceeded:
		return DeadlineExceeded
	case codes.ResourceExhausted:
		return ResourceExhausted
	case codes.Aborted:
		return Aborted
	default:
		return Unknown
	}
}

//ErrorTypeToGRPCCode converts a grpc status code into the matching ErrorType or Unknown
func ErrorTypeToGRPCCode(errorType ErrorType) codes.Code {
	switch errorType {
	case NotFound:
		return codes.NotFound
	case InvalidArgument:
		return codes.InvalidArgument
	case AlreadyExists:
		return codes.AlreadyExists
	case PermissionDenied:
		return codes.PermissionDenied
	case Unauthenticated:
		return codes.Unauthenticated
	case Unimplemented:
		return codes.Unimplemented
	case Unknown:
		return codes.Unknown
	case Internal:
		return codes.Internal
	case Unavailable:
		return codes.Unavailable
	case FailedPrecondition:
		return codes.FailedPrecondition
	case DeadlineExceeded:
		return codes.DeadlineExceeded
	case ResourceExhausted:
		return codes.ResourceExhausted
	case Aborted:
		return codes.Aborted
	default:
		return codes.Unknown
	}
}
