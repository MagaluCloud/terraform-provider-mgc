package objects

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"

	objSdk "github.com/MagaluCloud/mgc-sdk-go/objectstorage"
)

func convertToCORSConfiguration(ctx context.Context, tfCORS CORS) (*objSdk.CORSConfiguration, error) {
	var allowedHeaders, allowedMethods, allowedOrigins, exposeHeaders []string

	if !tfCORS.AllowedHeaders.IsNull() && !tfCORS.AllowedHeaders.IsUnknown() {
		if diags := tfCORS.AllowedHeaders.ElementsAs(ctx, &allowedHeaders, false); diags.HasError() {
			return nil, fmt.Errorf("failed to parse allowed headers")
		}
	}
	if tfCORS.AllowedMethods.IsUnknown() {
		return nil, fmt.Errorf("allowed methods cannot be unknown")
	}
	if !tfCORS.AllowedMethods.IsNull() {
		if diags := tfCORS.AllowedMethods.ElementsAs(ctx, &allowedMethods, false); diags.HasError() {
			return nil, fmt.Errorf("failed to parse allowed methods")
		}
	}
	if tfCORS.AllowedOrigins.IsUnknown() {
		return nil, fmt.Errorf("allowed origins cannot be unknown")
	}
	if !tfCORS.AllowedOrigins.IsNull() {
		if diags := tfCORS.AllowedOrigins.ElementsAs(ctx, &allowedOrigins, false); diags.HasError() {
			return nil, fmt.Errorf("failed to parse allowed origins")
		}
	}
	if !tfCORS.ExposeHeaders.IsNull() && !tfCORS.ExposeHeaders.IsUnknown() {
		if diags := tfCORS.ExposeHeaders.ElementsAs(ctx, &exposeHeaders, false); diags.HasError() {
			return nil, fmt.Errorf("failed to parse expose headers")
		}
	}

	maxAge := 0
	if !tfCORS.MaxAgeSeconds.IsNull() && !tfCORS.MaxAgeSeconds.IsUnknown() {
		maxAge = int(tfCORS.MaxAgeSeconds.ValueInt64())
	}

	sdkRule := objSdk.CORSRule{
		AllowedHeaders: allowedHeaders,
		AllowedMethods: allowedMethods,
		AllowedOrigins: allowedOrigins,
		ExposeHeaders:  exposeHeaders,
		MaxAgeSeconds:  maxAge,
	}

	return &objSdk.CORSConfiguration{
		CORSRules: []objSdk.CORSRule{sdkRule},
	}, nil
}

func convertFromCORSConfiguration(ctx context.Context, sdkCORS *objSdk.CORSConfiguration) (*CORS, error) {
	if len(sdkCORS.CORSRules) == 0 {
		return nil, fmt.Errorf("no CORS rules found")
	}

	sdkRule := sdkCORS.CORSRules[0]

	allowedHeaders, diag := types.ListValueFrom(ctx, types.StringType, sdkRule.AllowedHeaders)
	if diag.HasError() {
		return nil, fmt.Errorf("failed to convert allowed headers")
	}

	allowedMethods, diag := types.ListValueFrom(ctx, types.StringType, sdkRule.AllowedMethods)
	if diag.HasError() {
		return nil, fmt.Errorf("failed to convert allowed methods")
	}

	allowedOrigins, diag := types.ListValueFrom(ctx, types.StringType, sdkRule.AllowedOrigins)
	if diag.HasError() {
		return nil, fmt.Errorf("failed to convert allowed origins")
	}

	exposeHeaders, diag := types.ListValueFrom(ctx, types.StringType, sdkRule.ExposeHeaders)
	if diag.HasError() {
		return nil, fmt.Errorf("failed to convert expose headers")
	}

	return &CORS{
		AllowedHeaders: allowedHeaders,
		AllowedMethods: allowedMethods,
		AllowedOrigins: allowedOrigins,
		ExposeHeaders:  exposeHeaders,
		MaxAgeSeconds:  types.Int64Value(int64(sdkRule.MaxAgeSeconds)),
	}, nil
}
