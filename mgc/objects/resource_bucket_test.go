package objects

import (
	"context"
	"errors"
	"testing"

	objSdk "github.com/MagaluCloud/mgc-sdk-go/objectstorage"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

func corsNullObject() types.Object {
	return types.ObjectNull(map[string]attr.Type{
		"allowed_headers": types.ListType{ElemType: types.StringType},
		"allowed_methods": types.ListType{ElemType: types.StringType},
		"allowed_origins": types.ListType{ElemType: types.StringType},
		"expose_headers":  types.ListType{ElemType: types.StringType},
		"max_age_seconds": types.Int64Type,
	})
}

// fakeBucketService embeds the interface so only the methods used by Read need
// to be implemented.
type fakeBucketService struct {
	objSdk.BucketService

	versioningStatus *objSdk.BucketVersioningConfiguration
	lockStatus       bool
	lockErr          error
	policy           *objSdk.Policy
	policyErr        error
	corsConfig       *objSdk.CORSConfiguration
	corsErr          error
}

func (f *fakeBucketService) Exists(context.Context, string) (bool, error) {
	return true, nil
}

func (f *fakeBucketService) GetVersioningStatus(context.Context, string) (*objSdk.BucketVersioningConfiguration, error) {
	return f.versioningStatus, nil
}

func (f *fakeBucketService) GetBucketLockStatus(context.Context, string) (bool, error) {
	return f.lockStatus, f.lockErr
}

func (f *fakeBucketService) GetPolicy(context.Context, string) (*objSdk.Policy, error) {
	return f.policy, f.policyErr
}

func (f *fakeBucketService) GetCORS(context.Context, string) (*objSdk.CORSConfiguration, error) {
	return f.corsConfig, f.corsErr
}

func readBucket(t *testing.T, svc objSdk.BucketService, prior ObjectStorageBucketV1) (ObjectStorageBucketV1, resource.ReadResponse) {
	t.Helper()
	ctx := context.Background()

	res := &objectStorageBuckets{buckets: svc, region: "br-ne1", endpoint: "https://example.com"}
	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema}}
	if diags := req.State.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("could not set prior state: %v", diags)
	}

	resp := resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Read(ctx, req, &resp)

	var got ObjectStorageBucketV1
	if !resp.Diagnostics.HasError() {
		if diags := resp.State.Get(ctx, &got); diags.HasError() {
			t.Fatalf("could not read resulting state: %v", diags)
		}
	}
	return got, resp
}

func TestRead_LockError(t *testing.T) {
	lockErr := errors.New("Versioning must be 'Enabled' on the bucket to apply a Object Lock configuration")

	t.Run("import without previous state falls back to false", func(t *testing.T) {
		svc := &fakeBucketService{
			versioningStatus: &objSdk.BucketVersioningConfiguration{Status: "Disabled"},
			lockErr:          lockErr,
		}
		prior := ObjectStorageBucketV1{
			Bucket: types.StringValue("bucket"),
			Lock:   types.BoolNull(),
			Policy: jsontypes.NewNormalizedNull(),
			CORS:   corsNullObject(),
		}

		got, resp := readBucket(t, svc, prior)

		assert.False(t, resp.Diagnostics.HasError())
		assert.Equal(t, types.BoolValue(false), got.Lock)
		assert.Equal(t, 0, resp.Diagnostics.WarningsCount(), "import must not warn about a value it never had")
	})

	t.Run("previous value is preserved with a warning", func(t *testing.T) {
		svc := &fakeBucketService{
			versioningStatus: &objSdk.BucketVersioningConfiguration{Status: "Enabled"},
			lockErr:          lockErr,
		}
		prior := ObjectStorageBucketV1{
			Bucket: types.StringValue("bucket"),
			Lock:   types.BoolValue(true),
			Policy: jsontypes.NewNormalizedNull(),
			CORS:   corsNullObject(),
		}

		got, resp := readBucket(t, svc, prior)

		assert.False(t, resp.Diagnostics.HasError())
		assert.Equal(t, types.BoolValue(true), got.Lock)
		assert.Equal(t, 1, resp.Diagnostics.WarningsCount())
	})
}

func TestRead_PolicyAndCORSErrorsPreservePreviousState(t *testing.T) {
	svc := &fakeBucketService{
		versioningStatus: &objSdk.BucketVersioningConfiguration{Status: "Enabled"},
		policyErr:        errors.New("transient error"),
		corsErr:          errors.New("transient error"),
	}
	prior := ObjectStorageBucketV1{
		Bucket: types.StringValue("bucket"),
		Lock:   types.BoolValue(false),
		Policy: jsontypes.NewNormalizedValue(`{"Version":"2012-10-17"}`),
		CORS:   corsNullObject(),
	}

	got, resp := readBucket(t, svc, prior)

	assert.False(t, resp.Diagnostics.HasError())
	assert.Equal(t, prior.Policy, got.Policy)
	assert.Equal(t, prior.CORS, got.CORS)
	assert.Equal(t, 2, resp.Diagnostics.WarningsCount())
}
