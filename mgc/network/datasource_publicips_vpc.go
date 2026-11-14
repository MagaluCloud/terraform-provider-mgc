package network

import (
	"context"
	"fmt"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"
)

const nilVPCUUID = "00000000-0000-0000-0000-000000000000"

type publicIPVPCReader interface {
	List(ctx context.Context) ([]netSDK.VPC, error)
	ListPublicIPs(ctx context.Context, vpcID string) ([]netSDK.PublicIPDb, error)
}

func hasPublicIPVPCID(vpcID *string) bool {
	return vpcID != nil && *vpcID != "" && *vpcID != nilVPCUUID
}

// findPublicIPVPCIDs falls back to the VPC-scoped public IP endpoint when the
// tenant-wide public IP endpoint omits the VPC ID.
func findPublicIPVPCIDs(ctx context.Context, service publicIPVPCReader, publicIPIDs []string) (map[string]string, error) {
	wanted := make(map[string]struct{}, len(publicIPIDs))
	for _, id := range publicIPIDs {
		if id != "" {
			wanted[id] = struct{}{}
		}
	}
	if len(wanted) == 0 {
		return map[string]string{}, nil
	}

	vpcs, err := service.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list VPCs while resolving public IP VPC IDs: %w", err)
	}

	resolved := make(map[string]string, len(wanted))
	for _, vpc := range vpcs {
		if vpc.ID == nil || *vpc.ID == "" {
			continue
		}

		publicIPs, err := service.ListPublicIPs(ctx, *vpc.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list public IPs for VPC %s: %w", *vpc.ID, err)
		}

		for _, publicIP := range publicIPs {
			if publicIP.ID == nil {
				continue
			}
			if _, ok := wanted[*publicIP.ID]; ok {
				resolved[*publicIP.ID] = *vpc.ID
			}
		}
	}

	return resolved, nil
}
