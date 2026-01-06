package containerregistry

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	crSDK "github.com/MagaluCloud/mgc-sdk-go/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

type proxyCacheDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	ProviderName types.String `tfsdk:"provider_name"`
	URL          types.String `tfsdk:"url"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

type ProxyCacheDataSource struct {
	proxyCacheService crSDK.ProxyCachesService
}

func NewProxyCacheDataSource() datasource.DataSource {
	return &ProxyCacheDataSource{}
}

func (pc *ProxyCacheDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_registry_proxy_cache"
}

func (pc *ProxyCacheDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	pc.proxyCacheService = crSDK.New(&dataConfig.CoreConfig).ProxyCaches()
}

func (pc *ProxyCacheDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a specific proxy cache.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the proxy cache",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Unique name for each tenant, used for the proxy cache",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the proxy cache",
				Computed:    true,
			},
			"provider_name": schema.StringAttribute{
				Description: "Provider identifier",
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "Endpoint URL for the proxied registry",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the proxy cache was created",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the proxy cache was last updated",
				Computed:    true,
			},
		},
	}
}

func (pc *ProxyCacheDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data proxyCacheDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	proxyCache, err := pc.proxyCacheService.Get(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.ID = types.StringValue(proxyCache.ID)
	data.Name = types.StringValue(proxyCache.Name)
	data.Description = types.StringValue(proxyCache.Description)
	data.ProviderName = types.StringValue(proxyCache.Provider)
	data.URL = types.StringValue(proxyCache.URL)
	data.CreatedAt = types.StringValue(proxyCache.CreatedAt)
	data.UpdatedAt = types.StringValue(proxyCache.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
