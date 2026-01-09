package containerregistry

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	crSDK "github.com/MagaluCloud/mgc-sdk-go/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

type proxyCache struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	ProviderName types.String `tfsdk:"provider_name"`
	URL          types.String `tfsdk:"url"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

type proxyCacheListDataSourceModel struct {
	ProxyCaches []proxyCache `tfsdk:"proxy_caches"`
}

type ProxyCacheListDataSource struct {
	proxyCacheService crSDK.ProxyCachesService
}

func NewProxyCacheListDataSource() datasource.DataSource {
	return &ProxyCacheListDataSource{}
}

func (pc *ProxyCacheListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_registry_proxy_caches"
}

func (pc *ProxyCacheListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (pc *ProxyCacheListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all proxy caches",
		Attributes: map[string]schema.Attribute{
			"proxy_caches": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Unique identifier for the proxy cache",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Unique name for each tenant, used for the proxy cache",
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
				},
			},
		},
	}
}

func (pc *ProxyCacheListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	proxyCaches, err := pc.proxyCacheService.ListAll(ctx, crSDK.ProxyCacheListAllOptions{})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	var result proxyCacheListDataSourceModel

	for _, pc := range proxyCaches {
		result.ProxyCaches = append(result.ProxyCaches, proxyCache{
			ID:           types.StringValue(pc.ID),
			Name:         types.StringValue(pc.Name),
			ProviderName: types.StringValue(pc.Provider),
			URL:          types.StringValue(pc.URL),
			CreatedAt:    types.StringValue(pc.CreatedAt),
			UpdatedAt:    types.StringValue(pc.UpdatedAt),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &result)...)
}
