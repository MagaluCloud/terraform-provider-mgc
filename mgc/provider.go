package mgc

import (
	"context"
	"fmt"
	"regexp"
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
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	sdk "github.com/MagaluCloud/mgc-sdk-go/client"
)

const (
	providerTypeName = "mgc"
	defaultRegion    = "br-se1"
	defaultEnv       = "prod"
)

type mgcProvider struct {
	version string
}

var rgxUUIDv4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type ProviderModel struct {
	Region    types.String `tfsdk:"region"`
	Env       types.String `tfsdk:"env"`
	ApiKey    types.String `tfsdk:"api_key"`
	AccessKey types.String `tfsdk:"access_key"`
	SecretKey types.String `tfsdk:"secret_key"`
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
				Description: "The environment to use. Options: prod / pre-prod / dev-qa. Default is " + defaultEnv,
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("prod", "pre-prod", "dev-qa"),
				},
			},
			"region": schema.StringAttribute{
				Description: "The region to use for resources. Options: br-ne1 / br-se1. Default is " + defaultRegion,
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("br-ne1", "br-se1", "br-mgl1", "br-mc1"),
				},
			},
			"api_key": schema.StringAttribute{
				Description: "The Magalu API Key for authentication.",
				Required:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(rgxUUIDv4, "must be a valid Magalu Cloud API key"),
				},
			},
			"access_key": schema.StringAttribute{
				Description: "Access Key (Access ID) for Object Storage.",
				Optional:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(rgxUUIDv4, "must be a valid Magalu Cloud access key"),
					stringvalidator.AlsoRequires(path.MatchRoot("secret_key")),
				},
			},
			"secret_key": schema.StringAttribute{
				Description: "Secret Key for Object Storage.",
				Optional:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(rgxUUIDv4, "must be a valid Magalu Cloud secret key"),
					stringvalidator.AlsoRequires(path.MatchRoot("access_key")),
				},
			},
		},
	}
}

func (p *mgcProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var plan ProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Env.ValueString() == "" {
		plan.Env = types.StringValue(defaultEnv)
	}

	if plan.Region.ValueString() == "" {
		plan.Region = types.StringValue(defaultRegion)
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
		ApiKey:    plan.ApiKey.ValueString(),
		Env:       plan.Env.ValueString(),
		Region:    plan.Region.ValueString(),
		AccessKey: plan.AccessKey.ValueString(),
		SecretKey: plan.SecretKey.ValueString(),
	}

	sdkUrl := sdk.MgcUrl(utils.RegionToUrl(output.Region, output.Env))
	tflog.Info(context.Background(), "Using MGC URL: "+sdkUrl.String())

	output.CoreConfig = *sdk.NewMgcClient(
		sdk.WithAPIKey(output.ApiKey),
		sdk.WithBaseURL(sdkUrl),
		sdk.WithUserAgent(fmt.Sprintf("MgcTF/%s (%s; %s)", tfVersion, runtime.GOOS, runtime.GOARCH)),
	)

	return output
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &mgcProvider{
			version: version,
		}
	}
}
