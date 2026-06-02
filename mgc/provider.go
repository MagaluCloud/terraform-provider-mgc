package mgc

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"runtime"
	"strings"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/blockstorage"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/database"
	internalhttp "github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/http"
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
	Region        types.String    `tfsdk:"region"`
	Env           types.String    `tfsdk:"env"`
	ApiKey        types.String    `tfsdk:"api_key"`
	KeyPairID     types.String    `tfsdk:"key_pair_id"`
	KeyPairSecret types.String    `tfsdk:"key_pair_secret"`
	Endpoints     *EndpointsModel `tfsdk:"endpoints"`
}

type EndpointsModel struct {
	BlockStorage      types.String `tfsdk:"block_storage"`
	ContainerRegistry types.String `tfsdk:"container_registry"`
	Database          types.String `tfsdk:"database"`
	Kubernetes        types.String `tfsdk:"kubernetes"`
	Lbaas             types.String `tfsdk:"lbaas"`
	Network           types.String `tfsdk:"network"`
	ObjectStorage     types.String `tfsdk:"object_storage"`
	SSH               types.String `tfsdk:"ssh"`
	VirtualMachine    types.String `tfsdk:"virtual_machine"`
}

func (p *mgcProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = providerTypeName
	resp.Version = p.version
}

func (p *mgcProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform Provider for Magalu Cloud",
		Blocks: map[string]schema.Block{
			"endpoints": schema.SingleNestedBlock{
				Description: "Custom endpoint URLs for individual Magalu Cloud services. Useful for local development, testing, or private deployments.",
				Attributes: map[string]schema.Attribute{
					"block_storage":      endpointAttribute("Custom endpoint for the Block Storage service."),
					"container_registry": endpointAttribute("Custom endpoint for the Container Registry service."),
					"database":           endpointAttribute("Custom endpoint for the Database (DBaaS) service."),
					"kubernetes":         endpointAttribute("Custom endpoint for the Kubernetes service."),
					"lbaas":              endpointAttribute("Custom endpoint for the Load Balancer as a Service (LBaaS)."),
					"network":            endpointAttribute("Custom endpoint for the Network service."),
					"object_storage":     endpointAttribute("Custom endpoint for the Object Storage (S3-compatible) service."),
					"ssh":                endpointAttribute("Custom endpoint for the SSH Keys service."),
					"virtual_machine":    endpointAttribute("Custom endpoint for the Virtual Machines service."),
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			// "env" is intentionally omitted from the published docs (docs-extra/index.md)
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
			"key_pair_id": schema.StringAttribute{
				Description: "Key Pair ID for Object Storage.",
				Optional:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(rgxUUIDv4, "must be a valid Magalu Cloud key pair id"),
					stringvalidator.AlsoRequires(path.MatchRoot("key_pair_secret")),
				},
			},
			"key_pair_secret": schema.StringAttribute{
				Description: "Key Pair Secret for Object Storage.",
				Optional:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(rgxUUIDv4, "must be a valid Magalu Cloud key pair secret"),
					stringvalidator.AlsoRequires(path.MatchRoot("key_pair_id")),
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
		ApiKey:        plan.ApiKey.ValueString(),
		Env:           plan.Env.ValueString(),
		Region:        plan.Region.ValueString(),
		KeyPairID:     plan.KeyPairID.ValueString(),
		KeyPairSecret: plan.KeyPairSecret.ValueString(),
	}

	sdkUrl := sdk.MgcUrl(utils.RegionToUrl(output.Region, output.Env))
	tflog.Info(context.Background(), "Using MGC URL: "+sdkUrl.String())

	output.CoreConfig = *sdk.NewMgcClient(
		sdk.WithAPIKey(output.ApiKey),
		sdk.WithBaseURL(sdkUrl),
		sdk.WithUserAgent(fmt.Sprintf("MgcTF/%s (%s; %s)", tfVersion, runtime.GOOS, runtime.GOARCH)),
	)

	httpClient := output.CoreConfig.GetConfig().HTTPClient
	httpClient.Transport = internalhttp.NewRequestIDRoundTripper(httpClient.Transport)

	if plan.Endpoints != nil {
		endpoints := make(map[string]string)
		setEndpoint(endpoints, utils.ServiceBlockStorage, plan.Endpoints.BlockStorage)
		setEndpoint(endpoints, utils.ServiceContainerRegistry, plan.Endpoints.ContainerRegistry)
		setEndpoint(endpoints, utils.ServiceDatabase, plan.Endpoints.Database)
		setEndpoint(endpoints, utils.ServiceKubernetes, plan.Endpoints.Kubernetes)
		setEndpoint(endpoints, utils.ServiceLbaas, plan.Endpoints.Lbaas)
		setEndpoint(endpoints, utils.ServiceNetwork, plan.Endpoints.Network)
		setEndpoint(endpoints, utils.ServiceObjectStorage, plan.Endpoints.ObjectStorage)
		setEndpoint(endpoints, utils.ServiceSSH, plan.Endpoints.SSH)
		setEndpoint(endpoints, utils.ServiceVirtualMachine, plan.Endpoints.VirtualMachine)
		output.SetServiceEndpoints(endpoints)
	}

	return output
}

func setEndpoint(m map[string]string, service string, val types.String) {
	if val.IsNull() || val.IsUnknown() {
		return
	}
	v := strings.TrimRight(strings.TrimSpace(val.ValueString()), "/")
	if v != "" {
		m[service] = v
	}
}

// endpointAttribute builds the schema definition for a custom service endpoint
func endpointAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:    true,
		Description: description,
		Validators:  []validator.String{endpointURLValidator{}},
	}
}

type endpointURLValidator struct{}

func (endpointURLValidator) Description(_ context.Context) string {
	return "value must be an absolute URL with an http or https scheme and a host"
}

func (v endpointURLValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (endpointURLValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	raw := strings.TrimSpace(req.ConfigValue.ValueString())
	if raw == "" {
		return
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid endpoint URL",
			fmt.Sprintf("%q must be an absolute URL including an http/https scheme and a host, e.g. https://localhost:8080. "+
				"Provide only the base URL; the service path is appended automatically.", raw),
		)
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &mgcProvider{
			version: version,
		}
	}
}
