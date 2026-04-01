// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

// ErrorCode represents transport-level error classifications
type ErrorCode string

const (
	ErrorCodeNone             ErrorCode = "NONE"
	ErrorCodeInvalidInput     ErrorCode = "INVALID_INPUT"
	ErrorCodeUnauthorized     ErrorCode = "UNAUTHORIZED"
	ErrorCodeResourceNotFound ErrorCode = "RESOURCE_NOT_FOUND"
	ErrorCodeAlreadyExists    ErrorCode = "ALREADY_EXISTS"
	ErrorCodeThrottling       ErrorCode = "THROTTLING"
	ErrorCodeInternalError    ErrorCode = "INTERNAL_ERROR"
	ErrorCodeUnknown          ErrorCode = "UNKNOWN"
)

// Error represents a transport layer error with classification
type Error struct {
	Code       ErrorCode
	Message    string
	HTTPCode   int
	Underlying error
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Underlying
}

// ClassifyHTTPStatus maps HTTP status codes to error codes
func ClassifyHTTPStatus(statusCode int) ErrorCode {
	switch statusCode {
	case 200, 201, 204:
		return ErrorCodeNone
	case 400:
		return ErrorCodeInvalidInput
	case 401, 403:
		return ErrorCodeUnauthorized
	case 404:
		return ErrorCodeResourceNotFound
	case 409:
		return ErrorCodeAlreadyExists
	case 429:
		return ErrorCodeThrottling
	case 500, 502, 503:
		return ErrorCodeInternalError
	default:
		if statusCode >= 200 && statusCode < 300 {
			return ErrorCodeNone
		}
		return ErrorCodeUnknown
	}
}

// ToResourceErrorCode converts transport error code to formae resource error code
func ToResourceErrorCode(code ErrorCode) resource.OperationErrorCode {
	switch code {
	case ErrorCodeInvalidInput:
		return resource.OperationErrorCodeInvalidRequest
	case ErrorCodeUnauthorized:
		return resource.OperationErrorCodeAccessDenied
	case ErrorCodeResourceNotFound:
		return resource.OperationErrorCodeNotFound
	case ErrorCodeAlreadyExists:
		return resource.OperationErrorCodeAlreadyExists
	case ErrorCodeThrottling:
		return resource.OperationErrorCodeThrottling
	case ErrorCodeInternalError:
		return resource.OperationErrorCodeServiceInternalError
	default:
		return resource.OperationErrorCodeServiceInternalError
	}
}

// ClassifyCloudflareError converts a Cloudflare SDK error to our error code
func ClassifyCloudflareError(err error) ErrorCode {
	if err == nil {
		return ErrorCodeNone
	}

	var apiErr *cloudflare.Error
	if errors.As(err, &apiErr) {
		return ClassifyHTTPStatus(apiErr.StatusCode)
	}

	// Check error message for common patterns
	errMsg := err.Error()
	if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "404") {
		return ErrorCodeResourceNotFound
	}
	if strings.Contains(errMsg, "unauthorized") || strings.Contains(errMsg, "401") || strings.Contains(errMsg, "403") {
		return ErrorCodeUnauthorized
	}
	if strings.Contains(errMsg, "already exists") || strings.Contains(errMsg, "409") {
		return ErrorCodeAlreadyExists
	}
	if strings.Contains(errMsg, "rate limit") || strings.Contains(errMsg, "429") {
		return ErrorCodeThrottling
	}

	return ErrorCodeUnknown
}

// IsNotFound returns true if the error indicates a resource was not found
func IsNotFound(err error) bool {
	return ClassifyCloudflareError(err) == ErrorCodeResourceNotFound
}

// IsAlreadyExists returns true if the error indicates a resource already exists
func IsAlreadyExists(err error) bool {
	return ClassifyCloudflareError(err) == ErrorCodeAlreadyExists
}
