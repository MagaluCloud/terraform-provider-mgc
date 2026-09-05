package blockstorage

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"testing"
	"time"

	storageSDK "github.com/MagaluCloud/mgc-sdk-go/blockstorage"
	"github.com/stretchr/testify/require"
)

type pollingVolumeService struct {
	storageSDK.VolumeService
	get func(context.Context) (*storageSDK.Volume, error)
}

func (s pollingVolumeService) Get(ctx context.Context, _ string, _ []storageSDK.SnapshotExpand) (*storageSDK.Volume, error) {
	return s.get(ctx)
}

func TestVolumeAttachmentWaitHonorsDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	calls := 0
	r := VolumeAttach{blockStorageVolumes: pollingVolumeService{get: func(context.Context) (*storageSDK.Volume, error) {
		calls++
		return &storageSDK.Volume{Status: "attaching"}, nil
	}}}
	err := r.waitForVolumeAvailability(ctx, "volume-1", AttachVolumeCompletedStatus)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Zero(t, calls, "must preserve the initial settling delay and stop when the parent deadline expires")
}

func TestVolumeAttachmentWaitRequiresCompletedStatus(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := VolumeAttach{blockStorageVolumes: pollingVolumeService{get: func(context.Context) (*storageSDK.Volume, error) {
		cancel()
		return &storageSDK.Volume{Status: "attaching"}, nil
	}}}
	require.ErrorIs(t, r.waitForVolumeAvailability(ctx, "volume-1", AttachVolumeCompletedStatus), context.Canceled)
}

type acceptedAttachment struct {
	storageSDK.VolumeService
	cancel      context.CancelFunc
	attachError error
}

func (s acceptedAttachment) Attach(context.Context, string, string) error {
	s.cancel()
	return s.attachError
}
func TestAttachmentCreatePreservesOnlyAcceptedOperations(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(fmt.Sprint(failed), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			service := acceptedAttachment{cancel: cancel}
			if failed {
				service.attachError = errors.New("attach rejected")
			}
			r := VolumeAttach{blockStorageVolumes: service}
			var schema resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schema)
			model := VolumeAttachResourceModel{BlockStorageID: types.StringValue("volume-1"), VirtualMachineID: types.StringValue("vm-1")}
			plan := tfsdk.Plan{Schema: schema.Schema}
			require.False(t, plan.Set(ctx, &model).HasError())
			resp := resource.CreateResponse{State: tfsdk.State{Schema: schema.Schema}}
			r.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)
			require.True(t, resp.Diagnostics.HasError())
			if failed {
				require.True(t, resp.State.Raw.IsNull())
				return
			}
			require.False(t, resp.State.Raw.IsNull(), "accepted attachment must remain tracked after cancellation")
			var saved VolumeAttachResourceModel
			require.False(t, resp.State.Get(context.Background(), &saved).HasError())
			require.Equal(t, model, saved)
		})
	}
}
