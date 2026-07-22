package objects

import (
	"testing"

	objSdk "github.com/MagaluCloud/mgc-sdk-go/objectstorage"
	"github.com/stretchr/testify/assert"
)

func TestDropServerInjectedPolicyID(t *testing.T) {
	testCases := []struct {
		name              string
		fetchedObj        *objSdk.Policy
		statePolicy       string
		expectedFetchedID string
	}{
		{
			name:              "Empty state policy keeps fetched ID",
			fetchedObj:        &objSdk.Policy{Id: "server-generated-id"},
			statePolicy:       "",
			expectedFetchedID: "server-generated-id",
		},
		{
			name:              "Invalid JSON state policy keeps fetched ID",
			fetchedObj:        &objSdk.Policy{Id: "server-generated-id"},
			statePolicy:       "not a json",
			expectedFetchedID: "server-generated-id",
		},
		{
			name:              "State policy with Id field keeps fetched ID",
			fetchedObj:        &objSdk.Policy{Id: "server-generated-id"},
			statePolicy:       `{"Version":"2012-10-17","Id":"user-defined-id","Statement":[]}`,
			expectedFetchedID: "server-generated-id",
		},
		{
			name:              "State policy without Id field drops fetched ID",
			fetchedObj:        &objSdk.Policy{Id: "server-generated-id"},
			statePolicy:       `{"Version":"2012-10-17","Statement":[]}`,
			expectedFetchedID: "",
		},
		{
			name:              "State policy with non-string Id field drops fetched ID",
			fetchedObj:        &objSdk.Policy{Id: "server-generated-id"},
			statePolicy:       `{"Version":"2012-10-17","Id":123,"Statement":[]}`,
			expectedFetchedID: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			DropServerInjectedPolicyID(tc.fetchedObj, tc.statePolicy)

			assert.Equal(t, tc.expectedFetchedID, tc.fetchedObj.Id)
		})
	}
}
