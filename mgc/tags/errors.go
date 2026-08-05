package tags

import (
	"errors"
	"net/http"
	"slices"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
)

// isNotFound reports whether err means the entity is gone, so Read can drop it
// from the state and Delete can succeed.
//
// 400 counts as gone on purpose: the tags API answers it — instead of 404 — when
// detaching a tag that is not attached and when reading a resource that carries
// no tag. Revisit once the API returns 404 for those.
func isNotFound(err error) bool {
	return hasStatus(err, http.StatusNotFound, http.StatusBadRequest)
}

// isConflict reports whether err means the entity already exists: a tag with the
// same name, or a tag already attached to the resource.
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
