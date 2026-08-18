package objects

import (
	"context"
	"testing"

	objSdk "github.com/MagaluCloud/mgc-sdk-go/objectstorage"
	"github.com/stretchr/testify/assert"
)

func TestDropServerInjectedPolicyID(t *testing.T) {
	testCases := []struct {
		name              string
		fetchedObj        *objSdk.Policy
		stateObj          map[string]any
		expectedFetchedID string
	}{
		{
			name:              "State without Id field drops fetched ID",
			fetchedObj:        &objSdk.Policy{Id: "server-generated-id"},
			stateObj:          map[string]any{"Version": "2012-10-17"},
			expectedFetchedID: "",
		},
		{
			name:              "State with Id field keeps fetched ID",
			fetchedObj:        &objSdk.Policy{Id: "server-generated-id"},
			stateObj:          map[string]any{"Id": "user-defined-id"},
			expectedFetchedID: "server-generated-id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			DropServerInjectedPolicyID(tc.fetchedObj, tc.stateObj)

			assert.Equal(t, tc.expectedFetchedID, tc.fetchedObj.Id)
		})
	}
}

func TestDropServerInjectedStatementSids(t *testing.T) {
	testCases := []struct {
		name         string
		fetchedObj   *objSdk.Policy
		stateObj     map[string]any
		expectedSids []string
	}{
		{
			name: "State statement without Sid drops fetched Sid",
			fetchedObj: &objSdk.Policy{Statement: []objSdk.Statement{
				{Sid: "Stmt-server-generated"},
			}},
			stateObj: map[string]any{
				"Statement": []any{
					map[string]any{"Effect": "Allow"},
				},
			},
			expectedSids: []string{""},
		},
		{
			name: "State statement with explicit Sid keeps fetched Sid",
			fetchedObj: &objSdk.Policy{Statement: []objSdk.Statement{
				{Sid: "user-defined-sid"},
			}},
			stateObj: map[string]any{
				"Statement": []any{
					map[string]any{"Sid": "user-defined-sid", "Effect": "Allow"},
				},
			},
			expectedSids: []string{"user-defined-sid"},
		},
		{
			name: "Mixed statements drop only the ones missing Sid in state",
			fetchedObj: &objSdk.Policy{Statement: []objSdk.Statement{
				{Sid: "user-defined-sid"},
				{Sid: "Stmt-server-generated-2"},
			}},
			stateObj: map[string]any{
				"Statement": []any{
					map[string]any{"Sid": "user-defined-sid", "Effect": "Allow"},
					map[string]any{"Effect": "Deny"},
				},
			},
			expectedSids: []string{"user-defined-sid", ""},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dropServerInjectedStatementSids(tc.fetchedObj, tc.stateObj)

			gotSids := make([]string, len(tc.fetchedObj.Statement))
			for i, stmt := range tc.fetchedObj.Statement {
				gotSids[i] = stmt.Sid
			}

			assert.Equal(t, tc.expectedSids, gotSids)
		})
	}
}

func TestDropServerInejction(t *testing.T) {
	newFetchedObj := func() *objSdk.Policy {
		return &objSdk.Policy{
			Id:        "server-generated-id",
			Statement: []objSdk.Statement{{Sid: "Stmt-server-generated"}},
		}
	}

	t.Run("Valid state policy calls DropServerInjectedPolicyID and dropServerInjectedStatementSids", func(t *testing.T) {
		fetchedObj := newFetchedObj()
		statePolicy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow"}]}`

		dropServerInejction(fetchedObj, statePolicy)

		assert.Equal(t, "", fetchedObj.Id)
		assert.Equal(t, "", fetchedObj.Statement[0].Sid)
	})
}

func TestUpgradeState_V0PolicyEmptyStringBecomesNull(t *testing.T) {
	ctx := context.Background()
	res := &objectStorageBuckets{}

	upgraders := res.UpgradeState(ctx)
	_, ok := upgraders[0]
	assert.True(t, ok, "a V0 state upgrader must be registered")
}
