package tags

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	"github.com/stretchr/testify/assert"
)

func httpError(status int, body string) error {
	return &clientSDK.HTTPError{StatusCode: status, Status: http.StatusText(status), Body: []byte(body)}
}

func TestIsNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil",
			err:      nil,
			expected: false,
		},
		{
			name:     "404 on a missing tag",
			err:      httpError(http.StatusNotFound, `{"message":"Not Found","detail":"Tag not found"}`),
			expected: true,
		},
		{
			name:     "404 on a missing tag value",
			err:      httpError(http.StatusNotFound, `{"message":"Not Found","detail":"tag value not found"}`),
			expected: true,
		},
		{
			name:     "400 on detaching a tag that is not attached",
			err:      httpError(http.StatusBadRequest, `{"message":"Bad Request","detail":"Tag was not found"}`),
			expected: true,
		},
		{
			name:     "409 on an already existing tag",
			err:      httpError(http.StatusConflict, `{"message":"Conflict","detail":"Entity already exists"}`),
			expected: false,
		},
		{
			name:     "422 on a validation error",
			err:      httpError(http.StatusUnprocessableEntity, `{"detail":[{"msg":"String should match pattern"}]}`),
			expected: false,
		},
		{
			name:     "500",
			err:      httpError(http.StatusInternalServerError, ""),
			expected: false,
		},
		{
			name:     "wrapped 404",
			err:      fmt.Errorf("reading tag: %w", httpError(http.StatusNotFound, "")),
			expected: true,
		},
		{
			name:     "plain error",
			err:      errors.New("connection refused"),
			expected: false,
		},
		{
			name:     "validation error raised by the SDK before the request",
			err:      &clientSDK.ValidationError{Field: "name", Message: "cannot be empty"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, isNotFound(tt.err))
		})
	}
}

func TestIsConflict(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil",
			err:      nil,
			expected: false,
		},
		{
			name:     "409 on creating an existing tag",
			err:      httpError(http.StatusConflict, `{"message":"Conflict","detail":"Entity already exists"}`),
			expected: true,
		},
		{
			name:     "409 on attaching an already attached tag",
			err:      httpError(http.StatusConflict, `{"message":"Conflict","detail":"Tag at index 1 is already attached to the resource"}`),
			expected: true,
		},
		{
			name:     "wrapped 409",
			err:      fmt.Errorf("creating tag: %w", httpError(http.StatusConflict, "")),
			expected: true,
		},
		{
			name:     "404",
			err:      httpError(http.StatusNotFound, ""),
			expected: false,
		},
		{
			name:     "plain error",
			err:      errors.New("connection refused"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, isConflict(tt.err))
		})
	}
}
