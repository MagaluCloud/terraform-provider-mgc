package containerregistry

import (
	"context"
	"fmt"
	"time"

	crSDK "github.com/MagaluCloud/mgc-sdk-go/containerregistry"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	scanStatusCompleted = "completed"
	scanStatusFailed    = "failed"
	scanStatusStopped   = "stopped"

	scanPollInterval = 10 * time.Second
	scanPollTimeout  = 30 * time.Minute
)

func waitForScanCompletion(ctx context.Context, svc crSDK.ScansService, scanID string) (*crSDK.ImageScanResponse, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, scanPollTimeout)
	defer cancel()

	for {
		scan, err := svc.Get(ctx, scanID)
		if err != nil {
			return nil, err
		}

		switch scan.Status {
		case scanStatusCompleted:
			return scan, nil
		case scanStatusFailed, scanStatusStopped:
			return nil, fmt.Errorf("the scan could not be completed (status: %s)", scan.Status)
		}

		select {
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("timeout waiting for scan %s to complete", scanID)
		case <-time.After(scanPollInterval):
		}
	}
}

var severitySummaryAttrTypes = map[string]attr.Type{
	"total":    types.Int64Type,
	"low":      types.Int64Type,
	"medium":   types.Int64Type,
	"high":     types.Int64Type,
	"critical": types.Int64Type,
	"fixable":  types.Int64Type,
}

func severitySummaryNullObject() types.Object {
	return types.ObjectNull(severitySummaryAttrTypes)
}

func severitySummaryToObject(_ context.Context, summary *crSDK.SeverityResponse) (types.Object, diag.Diagnostics) {
	if summary == nil {
		return severitySummaryNullObject(), nil
	}
	return types.ObjectValue(severitySummaryAttrTypes, map[string]attr.Value{
		"total":    types.Int64Value(int64(summary.Total)),
		"low":      types.Int64Value(int64(summary.Low)),
		"medium":   types.Int64Value(int64(summary.Medium)),
		"high":     types.Int64Value(int64(summary.High)),
		"critical": types.Int64Value(int64(summary.Critical)),
		"fixable":  types.Int64Value(int64(summary.Fixable)),
	})
}
