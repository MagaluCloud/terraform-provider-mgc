package client

import (
	"fmt"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	mgcSdk "github.com/MagaluCloud/magalu/mgc/lib"
	"github.com/MagaluCloud/magalu/mgc/sdk"
)

type SDKFrom interface {
	resource.ConfigureRequest | datasource.ConfigureRequest
}

func NewSDKClient[T SDKFrom, G tfutil.ResponseFrom](req T, resp *G) (*mgcSdk.Client, error, error) {
	var config tfutil.DataConfig

	devErrMsg := "fail to parse provider config"
	switch tp := any(req).(type) {
	case resource.ConfigureRequest:
		if cfg, ok := tp.ProviderData.(tfutil.DataConfig); ok {
			config = cfg
			break
		}
		return nil, fmt.Errorf("%s", devErrMsg), fmt.Errorf("unexpected Resource Configure Type")
	case datasource.ConfigureRequest:
		if cfg, ok := tp.ProviderData.(tfutil.DataConfig); ok {
			config = cfg
			break
		}
		return nil, fmt.Errorf("%s", devErrMsg), fmt.Errorf("unexpected Data Source Configure Type")
	default:
		return nil, fmt.Errorf("%s", devErrMsg), fmt.Errorf("provider data is null")
	}

	local_sdk := sdk.NewSdk()
	sdkClient := mgcSdk.NewClient(local_sdk)

	if config.Region != "" {
		_ = sdkClient.Sdk().Config().SetTempConfig("region", config.Region)
	}
	if config.Env != "" {
		_ = sdkClient.Sdk().Config().SetTempConfig("env", config.Env)
	}

	_ = sdkClient.Sdk().Auth().SetAPIKey(config.ApiKey)

	if config.Keypair.KeyID != "" && config.Keypair.KeySecret != "" {
		sdkClient.Sdk().Config().AddTempKeyPair("apikey", config.Keypair.KeyID, config.Keypair.KeySecret)
	}

	return sdkClient, nil, nil
}
