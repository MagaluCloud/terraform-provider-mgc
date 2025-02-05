package tfutil

import (
	"fmt"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
)

const (
	simpleHttpError       = "API request failed with HTTP error"
	simpleValidationError = "Request validation failed"
	simpleGenericError    = "An unexpected error occurred"
	simpleMaxRetriesError = "Max HTTP retries exceeded"
)

type HttpErrorResponse struct {
	Status    string
	Body      string
	URL       string
	RequestID string
}

func (e HttpErrorResponse) String() string {
	return fmt.Sprintf("HTTP Error:\n  Status: %s\n  Body: %s\n  URL: %s\n  Request ID: %s",
		e.Status,
		e.Body,
		e.URL,
		e.RequestID,
	)
}

func (e HttpErrorResponse) Error() string {
	return e.String()
}

func buildFromSDKError(err *clientSDK.HTTPError) (HttpErrorResponse, error) {
	if err == nil {
		return HttpErrorResponse{}, fmt.Errorf("cannot build error response from nil error")
	}

	e := HttpErrorResponse{
		Status: err.Status,
		Body:   string(err.Body),
	}

	if err.Response != nil && err.Response.Request != nil {
		e.URL = err.Response.Request.URL.String()
		if len(err.Response.Header[string(clientSDK.RequestIDKey)]) >= 1 {
			e.RequestID = err.Response.Header[string(clientSDK.RequestIDKey)][0]
		}
	}

	return e, nil
}

func ParseSDKError(err error) (msg, detail string) {
	if err == nil {
		return simpleGenericError, "nil error provided"
	}

	switch e := err.(type) {
	case *clientSDK.HTTPError:
		errorResponse, buildErr := buildFromSDKError(e)
		if buildErr != nil {
			return simpleGenericError, buildErr.Error()
		}
		return simpleHttpError, errorResponse.String()

	case *clientSDK.ValidationError:
		if e == nil {
			return simpleValidationError, "nil validation error"
		}
		return simpleValidationError, fmt.Sprintf("Field: %s - Message: %s", e.Field, e.Message)

	case *clientSDK.RetryError:
		if e == nil {
			return simpleMaxRetriesError, ""
		}
		if e.LastError == nil {
			return simpleMaxRetriesError, "unexpected last retry error"
		}
		if he, ok := e.LastError.(*clientSDK.HTTPError); ok {
			errorResponse, buildErr := buildFromSDKError(he)
			if buildErr != nil {
				return simpleMaxRetriesError, buildErr.Error()
			}
			return simpleMaxRetriesError, fmt.Sprintf("Max HTTP retries exceeded at %d retries.\nLast error:\n %s", e.Retries, errorResponse.String())
		}
		return simpleMaxRetriesError, fmt.Sprintf("Max HTTP retries exceeded at %d retries.\nLast error: %s", e.Retries, e.LastError.Error())

	default:
		return simpleGenericError, err.Error()
	}
}
