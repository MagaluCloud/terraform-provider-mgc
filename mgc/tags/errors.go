package tags

import (
	"errors"
	"net/http"
	"slices"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
)

func isNotFound(err error) bool {
	return hasStatus(err, http.StatusNotFound, http.StatusBadRequest)
}

func isConflict(err error) bool {
	return hasStatus(err, http.StatusConflict)
}

func hasStatus(err error, statuses ...int) bool {
	var httpErr *clientSDK.HTTPError
	if !errors.As(err, &httpErr) {
		return false
	}

	return slices.Contains(statuses, httpErr.StatusCode)
}
