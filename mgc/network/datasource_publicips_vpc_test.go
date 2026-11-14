package network

import (
	"context"
	"errors"
	"testing"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePublicIPVPCReader struct {
	vpcs       []netSDK.VPC
	publicIPs  map[string][]netSDK.PublicIPDb
	listErr    error
	publicIPErr map[string]error
}

func (f *fakePublicIPVPCReader) List(context.Context) ([]netSDK.VPC, error) {
	return f.vpcs, f.listErr
}

func (f *fakePublicIPVPCReader) ListPublicIPs(_ context.Context, vpcID string) ([]netSDK.PublicIPDb, error) {
	if err := f.publicIPErr[vpcID]; err != nil {
		return nil, err
	}
	return f.publicIPs[vpcID], nil
}

func TestFindPublicIPVPCIDs(t *testing.T) {
	t.Parallel()

	t.Run("resolves IDs through VPC-scoped public IP lists", func(t *testing.T) {
		t.Parallel()

		service := &fakePublicIPVPCReader{
			vpcs: []netSDK.VPC{
				{ID: publicIPVPCStringPointer("vpc-1")},
				{ID: publicIPVPCStringPointer("vpc-2")},
			},
			publicIPs: map[string][]netSDK.PublicIPDb{
				"vpc-1": {{ID: publicIPVPCStringPointer("ip-1")}},
				"vpc-2": {{ID: publicIPVPCStringPointer("ip-2")}},
			},
		}

		resolved, err := findPublicIPVPCIDs(context.Background(), service, []string{"ip-1", "ip-2", "missing"})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"ip-1": "vpc-1", "ip-2": "vpc-2"}, resolved)
	})

	t.Run("returns empty result when no IDs are requested", func(t *testing.T) {
		t.Parallel()

		service := &fakePublicIPVPCReader{}
		resolved, err := findPublicIPVPCIDs(context.Background(), service, nil)
		require.NoError(t, err)
		assert.Empty(t, resolved)
	})

	t.Run("returns an error when VPC listing fails", func(t *testing.T) {
		t.Parallel()

		service := &fakePublicIPVPCReader{listErr: errors.New("list failed")}
		_, err := findPublicIPVPCIDs(context.Background(), service, []string{"ip-1"})
		require.ErrorContains(t, err, "list VPCs")
	})

	t.Run("returns an error when a VPC public IP listing fails", func(t *testing.T) {
		t.Parallel()

		service := &fakePublicIPVPCReader{
			vpcs: []netSDK.VPC{{ID: publicIPVPCStringPointer("vpc-1")}},
			publicIPErr: map[string]error{"vpc-1": errors.New("request failed")},
		}
		_, err := findPublicIPVPCIDs(context.Background(), service, []string{"ip-1"})
		require.ErrorContains(t, err, "vpc-1")
	})
}

func publicIPVPCStringPointer(value string) *string {
	return &value
}
