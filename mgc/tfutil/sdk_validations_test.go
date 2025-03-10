package tfutil

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
)

func TestParseSDKError_NilError(t *testing.T) {
	msg, detail := ParseSDKError(nil)
	if msg != simpleGenericError {
		t.Errorf("expected message %q, got %q", simpleGenericError, msg)
	}
	if detail != "nil error provided" {
		t.Errorf("expected detail %q, got %q", "nil error provided", detail)
	}
}

func TestParseSDKError_HTTPError(t *testing.T) {
	reqURL, _ := url.Parse("http://example.com/test")
	resp := &http.Response{
		Request: &http.Request{
			URL: reqURL,
		},
		Header: http.Header{
			string(clientSDK.RequestIDKey): []string{"req-123"},
			MgcTraceIDKey:                  []string{"trace-789"},
		},
	}
	httpErr := &clientSDK.HTTPError{
		Status:   "404 Not Found",
		Body:     []byte(`{"error":"not found"}`),
		Response: resp,
	}

	msg, detail := ParseSDKError(httpErr)
	if msg != simpleHttpError {
		t.Errorf("expected message %q, got %q", simpleHttpError, msg)
	}

	expectedDetail := "HTTP Error:\n  Status: 404 Not Found\n  Body: {\"error\":\"not found\"}\n  URL: http://example.com/test\n  Request ID: req-123\n  MGC Trace ID: trace-789"
	if detail != expectedDetail {
		t.Errorf("expected detail %q, got %q", expectedDetail, detail)
	}
}

func TestParseSDKError_HTTPErrorNilResponse(t *testing.T) {
	httpErr := &clientSDK.HTTPError{
		Status: "500 Internal Server Error",
		Body:   []byte("server error"),
		Response: &http.Response{
			Header: http.Header{
				string(clientSDK.RequestIDKey): []string{"req-456"},
			},
		},
	}

	msg, detail := ParseSDKError(httpErr)
	if msg != simpleHttpError {
		t.Errorf("expected message %q, got %q", simpleHttpError, msg)
	}

	expectedDetail := "HTTP Error:\n  Status: 500 Internal Server Error\n  Body: server error\n  URL: \n  Request ID: \n  MGC Trace ID: "
	if detail != expectedDetail {
		t.Errorf("expected detail %q, got %q", expectedDetail, detail)
	}
}

func TestParseSDKError_NilHTTPError(t *testing.T) {
	var httpErr *clientSDK.HTTPError
	err := error(httpErr)

	msg, detail := ParseSDKError(err)
	if msg != simpleGenericError {
		t.Errorf("expected message %q, got %q", simpleGenericError, msg)
	}
	if detail != "cannot build error response from nil error" {
		t.Errorf("expected detail %q, got %q", "cannot build error response from nil error", detail)
	}
}

func TestParseSDKError_ValidationError(t *testing.T) {
	valErr := &clientSDK.ValidationError{
		Field:   "email",
		Message: "invalid format",
	}

	msg, detail := ParseSDKError(valErr)
	if msg != simpleValidationError {
		t.Errorf("expected message %q, got %q", simpleValidationError, msg)
	}

	expectedDetail := "Field: email - Message: invalid format"
	if detail != expectedDetail {
		t.Errorf("expected detail %q, got %q", expectedDetail, detail)
	}
}

func TestParseSDKError_NilValidationError(t *testing.T) {
	var valErr *clientSDK.ValidationError
	err := error(valErr)

	msg, detail := ParseSDKError(err)
	if msg != simpleValidationError {
		t.Errorf("expected message %q, got %q", simpleValidationError, msg)
	}
	if detail != "nil validation error" {
		t.Errorf("expected detail %q, got %q", "nil validation error", detail)
	}
}

func TestParseSDKError_RetryErrorWithHTTPError(t *testing.T) {
	reqURL, _ := url.Parse("http://example.com/retry")
	resp := &http.Response{
		Request: &http.Request{
			URL: reqURL,
		},
		Header: http.Header{
			string(clientSDK.RequestIDKey): []string{"retry-456"},
			MgcTraceIDKey:                  []string{"trace-123"},
		},
	}
	httpErr := &clientSDK.HTTPError{
		Status:   "503 Service Unavailable",
		Body:     []byte("unavailable"),
		Response: resp,
	}
	retryErr := &clientSDK.RetryError{
		Retries:   3,
		LastError: httpErr,
	}

	msg, detail := ParseSDKError(retryErr)
	if msg != simpleMaxRetriesError {
		t.Errorf("expected message %q, got %q", simpleMaxRetriesError, msg)
	}

	expectedHTTPDetail := "HTTP Error:\n  Status: 503 Service Unavailable\n  Body: unavailable\n  URL: http://example.com/retry\n  Request ID: retry-456\n  MGC Trace ID: trace-123"
	expectedDetail := fmt.Sprintf("Max HTTP retries exceeded at %d retries.\nLast error:\n %s", 3, expectedHTTPDetail)
	if detail != expectedDetail {
		t.Errorf("expected detail %q, got %q", expectedDetail, detail)
	}
}

func TestParseSDKError_RetryErrorWithGenericError(t *testing.T) {
	retryErr := &clientSDK.RetryError{
		Retries:   2,
		LastError: fmt.Errorf("connection refused"),
	}

	msg, detail := ParseSDKError(retryErr)
	if msg != simpleMaxRetriesError {
		t.Errorf("expected message %q, got %q", simpleMaxRetriesError, msg)
	}

	expectedDetail := "Max HTTP retries exceeded at 2 retries.\nLast error: connection refused"
	if detail != expectedDetail {
		t.Errorf("expected detail %q, got %q", expectedDetail, detail)
	}
}

func TestParseSDKError_RetryErrorNilLastError(t *testing.T) {
	retryErr := &clientSDK.RetryError{
		Retries:   3,
		LastError: nil,
	}

	msg, detail := ParseSDKError(retryErr)
	if msg != simpleMaxRetriesError {
		t.Errorf("expected message %q, got %q", simpleMaxRetriesError, msg)
	}
	if detail != "unexpected last retry error" {
		t.Errorf("expected detail %q, got %q", "unexpected last retry error", detail)
	}
}

func TestParseSDKError_NilRetryError(t *testing.T) {
	var retryErr *clientSDK.RetryError
	err := error(retryErr)

	msg, detail := ParseSDKError(err)
	if msg != simpleMaxRetriesError {
		t.Errorf("expected message %q, got %q", simpleMaxRetriesError, msg)
	}
	if detail != "" {
		t.Errorf("expected empty detail, got %q", detail)
	}
}

func TestParseSDKError_GenericError(t *testing.T) {
	err := fmt.Errorf("generic error")
	msg, detail := ParseSDKError(err)
	if msg != simpleGenericError {
		t.Errorf("expected message %q, got %q", simpleGenericError, msg)
	}
	if detail != "generic error" {
		t.Errorf("expected detail %q, got %q", "generic error", detail)
	}
}
