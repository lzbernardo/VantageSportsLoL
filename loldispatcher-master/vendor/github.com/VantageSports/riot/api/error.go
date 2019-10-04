package api

import (
	"fmt"
	"strconv"
	"time"
)

const (
	rateLimitExceeded  int = 429
	serviceUnavailable     = 503
)

type APIError interface {
	Error() string

	// IsRetryable indicates that the request is likely to succeed in the
	// future, assuming the client waits until RetryAfter() to send it.
	IsRetryable() bool

	// RetryAfter returns the time after which (iff IsRetryable() is true)
	// the request can be retried.
	RetryAfter() time.Time

	// The error code returned by riot API
	Code() int
}

type apiError struct {
	code          int
	rateErrorType string
	retryAfter    time.Time
}

func (e *apiError) Code() int {
	return e.code
}

func (e *apiError) IsRetryable() bool {
	return e.code == rateLimitExceeded || e.code == serviceUnavailable
}

func (e *apiError) RetryAfter() time.Time {
	return e.retryAfter
}

func (e *apiError) Error() string {
	var retrySuffix, timeSuffix string
	if e.IsRetryable() {
		retrySuffix = fmt.Sprintf(", rate error type: %s ", e.rateErrorType)
	}
	if !e.retryAfter.IsZero() {
		timeSuffix = fmt.Sprintf(", retry after: %v", e.retryAfter)
	}
	return fmt.Sprintf("api error - code: %v%s%s", e.code, retrySuffix, timeSuffix)
}

func NewAPIError(code int, limitTypeStr, retryAfter string) *apiError {
	apiErr := &apiError{
		code:          code,
		rateErrorType: limitTypeStr,
	}
	if retryAfter != "" {
		retrySecs, err := strconv.ParseInt(retryAfter, 10, 64)
		if err != nil {
			fmt.Println("unable to parse number from retryAfter:", retryAfter)
			retrySecs = 60
		}
		apiErr.retryAfter = time.Now().Add(time.Second * time.Duration(retrySecs))
	}
	return apiErr
}
