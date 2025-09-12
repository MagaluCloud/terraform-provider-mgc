package mgc

import (
	"context"
	"fmt"
	"runtime"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/blockstorage"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/database"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/kubernetes"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/lbaas"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/network"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/objects"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/platform"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/ssh"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/virtualmachines"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	mgcSdk "github.com/MagaluCloud/magalu/mgc/sdk"
	sdk "github.com/MagaluCloud/mgc-sdk-go/client"
)

const (
	providerTypeName   = "mgc"
	defaultRegion      = "br-se1"
	defaultEnv         = "prod"
	preProdMaxAttempts = 10
)

type mgcProvider struct {
	version string
	sdk     *mgcSdk.Sdk
}

type ProviderModel struct {
	Region        types.String        `tfsdk:"region"`
	Env           types.String        `tfsdk:"env"`
	ApiKey        types.String        `tfsdk:"api_key"`
	ObjectStorage *ObjectStorageModel `tfsdk:"object_storage"`
}

type ObjectStorageModel struct {
	ObjectKeyPair *KeyPairModel `tfsdk:"key_pair"`
}

type KeyPairModel struct {
	KeyID     types.String `tfsdk:"key_id"`
	KeySecret types.String `tfsdk:"key_secret"`
}

func (p *mgcProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = providerTypeName
	resp.Version = p.version
}

func (p *mgcProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform Provider for Magalu Cloud",
		Attributes: map[string]schema.Attribute{
			"env": schema.StringAttribute{
				Description: "The environment to use. Options: prod / pre-prod / dev-qa. Default is prod.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("prod", "pre-prod", "dev-qa"),
				},
			},
			"region": schema.StringAttribute{
				Description: "The region to use for resources. Options: br-ne1 / br-se1. Default is br-se1.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("br-ne1", "br-se1", "br-mgl1", "br-mc1"),
				},
			},
			"api_key": schema.StringAttribute{
				Description: "The Magalu API Key for authentication.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"object_storage": schema.SingleNestedAttribute{
				Description: "Configuration settings for Object Storage",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"key_pair": schema.SingleNestedAttribute{
						Description: "Bucket Key Pair configuration",
						Required:    true,
						Attributes: map[string]schema.Attribute{
							"key_id": schema.StringAttribute{
								Description: "The API Key Access ID.",
								Required:    true,
							},
							"key_secret": schema.StringAttribute{
								Description: "The API Key Secret.",
								Required:    true,
							},
						},
					},
				},
			},
		},
	}
}

func (p *mgcProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var plan ProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "fail to get configs from provider")
		return
	}

	if plan.Env.ValueString() == "" {
		plan.Env = types.StringValue(defaultEnv)
	}

	if plan.Region.ValueString() == "" {
		plan.Region = types.StringValue(defaultRegion)
	}

	if plan.ObjectStorage == nil {
		plan.ObjectStorage = &ObjectStorageModel{
			&KeyPairModel{},
		}
	}

	resourceOut := NewConfigData(plan, p.version)
	resp.DataSourceData = resourceOut
	resp.ResourceData = resourceOut
}

func (p *mgcProvider) Resources(ctx context.Context) []func() resource.Resource {
	var resources []func() resource.Resource

	resources = append(resources, blockstorage.GetResources()...)
	resources = append(resources, containerregistry.GetResources()...)
	resources = append(resources, database.GetResources()...)
	resources = append(resources, kubernetes.GetResources()...)
	resources = append(resources, lbaas.GetResources()...)
	resources = append(resources, network.GetResources()...)
	resources = append(resources, objects.GetResources()...)
	resources = append(resources, platform.GetResources()...)
	resources = append(resources, ssh.GetResources()...)
	resources = append(resources, virtualmachines.GetResources()...)

	return resources
}

func (p *mgcProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	var dataSources []func() datasource.DataSource

	dataSources = append(dataSources, blockstorage.GetDataSources()...)
	dataSources = append(dataSources, containerregistry.GetDataSources()...)
	dataSources = append(dataSources, database.GetDataSources()...)
	dataSources = append(dataSources, kubernetes.GetDataSources()...)
	dataSources = append(dataSources, lbaas.GetDataSources()...)
	dataSources = append(dataSources, network.GetDataSources()...)
	dataSources = append(dataSources, objects.GetDataSources()...)
	dataSources = append(dataSources, platform.GetDataSources()...)
	dataSources = append(dataSources, ssh.GetDataSources()...)
	dataSources = append(dataSources, virtualmachines.GetDataSources()...)

	return dataSources
}

func NewConfigData(plan ProviderModel, tfVersion string) utils.DataConfig {
	output := utils.DataConfig{
		ApiKey: plan.ApiKey.ValueString(),
		Env:    plan.Env.ValueString(),
		Region: plan.Region.ValueString(),
	}

	if plan.ObjectStorage != nil ||
		plan.ObjectStorage.ObjectKeyPair != nil ||
		plan.ObjectStorage.ObjectKeyPair.KeyID.IsNull() ||
		plan.ObjectStorage.ObjectKeyPair.KeySecret.IsNull() {
		output.Keypair = utils.KeyPairData{
			KeyID:     plan.ObjectStorage.ObjectKeyPair.KeyID.ValueString(),
			KeySecret: plan.ObjectStorage.ObjectKeyPair.KeySecret.ValueString(),
		}
	}

	sdkUrl := sdk.MgcUrl(utils.RegionToUrl(output.Region, output.Env))
	tflog.Info(context.Background(), "Using MGC URL: "+sdkUrl.String())

	var maxAttempts int
	if plan.Env.ValueString() == defaultEnv {
		maxAttempts = sdk.DefaultMaxAttempts
	} else {
		maxAttempts = preProdMaxAttempts
	}

	output.CoreConfig = *sdk.NewMgcClient(output.ApiKey,
		sdk.WithBaseURL(sdkUrl),
		sdk.WithUserAgent(fmt.Sprintf("MgcTF/%s (%s; %s)", tfVersion, runtime.GOOS, runtime.GOARCH)),
		sdk.WithRetryConfig(maxAttempts, sdk.DefaultInitialInterval, sdk.DefaultMaxInterval, sdk.DefaultBackoffFactor),
	)

	return output
}

func New(version string) func() provider.Provider {
	sdk := mgcSdk.NewSdk()
	mgcSdk.SetUserAgent("MgcTF/" + version)

	return func() provider.Provider {
		return &mgcProvider{
			sdk:     sdk,
			version: version,
		}
	}
}
