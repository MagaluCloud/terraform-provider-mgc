package blockstorage

import (
	"context"
	"errors"
	"net/http"
	"testing"

	storageSDK "github.com/MagaluCloud/mgc-sdk-go/blockstorage"
	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/testutils"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockVolumeService struct {
	mock.Mock
	storageSDK.VolumeService
}

func (m *mockVolumeService) Get(ctx context.Context, id string, expand []storageSDK.SnapshotExpand) (*storageSDK.Volume, error) {
	args := m.Called(ctx, id, expand)
	res, _ := args.Get(0).(*storageSDK.Volume)
	return res, args.Error(1)
}

func volumeReadState(t *testing.T, r *bsVolumes, id string) tfsdk.State {
	t.Helper()
	schemaResp := testutils.GetResourceTestSchema(t, r)
	state := tfsdk.State{Schema: schemaResp.Schema}
	diags := state.Set(context.Background(), bsVolumesResourceModel{
		ID:   types.StringValue(id),
		Name: types.StringValue("data"),
		Size: types.Int64Value(10),
		Type: types.StringValue("cloud_nvme1k"),
	})
	require.False(t, diags.HasError(), "%v", diags)
	return state
}

func TestBsVolumesResource_Read_NotFoundRemovesState(t *testing.T) {
	t.Parallel()

	mockSvc := &mockVolumeService{}
	mockSvc.On("Get", mock.Anything, "vol-gone", mock.Anything).Return(nil, &clientSDK.HTTPError{
		StatusCode: http.StatusNotFound,
		Status:     "404 Not Found",
		Body:       []byte(`{"error":"not found"}`),
		Response:   &http.Response{Header: http.Header{}},
	})

	r := &bsVolumes{bsVolumes: mockSvc}
	state := volumeReadState(t, r, "vol-gone")
	resp := &resource.ReadResponse{State: state}

	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
	assert.True(t, resp.State.Raw.IsNull())
	mockSvc.AssertExpectations(t)
}

func TestBsVolumesResource_Read_OtherErrorsKeepState(t *testing.T) {
	t.Parallel()

	mockSvc := &mockVolumeService{}
	mockSvc.On("Get", mock.Anything, "vol-err", mock.Anything).Return(nil, errors.New("api error"))

	r := &bsVolumes{bsVolumes: mockSvc}
	state := volumeReadState(t, r, "vol-err")
	resp := &resource.ReadResponse{State: state}

	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.False(t, resp.State.Raw.IsNull())
}
