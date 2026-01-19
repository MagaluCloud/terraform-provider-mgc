package containerregistry

import (
	"context"

	crSDK "github.com/MagaluCloud/mgc-sdk-go/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ProxyCacheModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	ProviderName types.String `tfsdk:"provider_name"`
	URL          types.String `tfsdk:"url"`
	AccessKey    types.String `tfsdk:"access_key"`
	AccessSecret types.String `tfsdk:"access_secret"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

type ProxyCacheResource struct {
	proxyCacheService crSDK.ProxyCachesService
}

func NewContainerRegistryProxyCacheResource() resource.Resource {
	return &ProxyCacheResource{}
}

func (pc *ProxyCacheResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_registry_proxy_cache"
}

func (pc *ProxyCacheResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (pc *ProxyCacheResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the proxy cache",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the proxy cache",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Unique name for each tenant, used for the proxy cache",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(4),
					stringvalidator.LengthAtMost(63),
				},
			},
			"description": schema.StringAttribute{
				Description: "Description of the proxy cache",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"provider_name": schema.StringAttribute{
				Description: "Provider identifier",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"url": schema.StringAttribute{
				Description: "Endpoint URL for the proxied registry",
				Required:    true,
			},
			"access_key": schema.StringAttribute{
				Description: "Provider access_id",
				Optional:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"access_secret": schema.StringAttribute{
				Description: "Provider access_secret",
				Optional:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the proxy cache was created",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the proxy cache was last updated",
				Computed:    true,
			},
		},
	}
}

func (pc *ProxyCacheResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProxyCacheModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := pc.proxyCacheService.Create(ctx, crSDK.CreateProxyCacheRequest{
		Name:         data.Name.ValueString(),
		Provider:     data.ProviderName.ValueString(),
		URL:          data.URL.ValueString(),
		Description:  data.Description.ValueStringPointer(),
		AccessKey:    data.AccessKey.ValueStringPointer(),
		AccessSecret: data.AccessSecret.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.ID = types.StringValue(created.ID)
	data.Name = types.StringValue(created.Name)

	proxyCache, err := pc.proxyCacheService.Get(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.CreatedAt = types.StringPointerValue(&proxyCache.CreatedAt)
	data.UpdatedAt = types.StringPointerValue(&proxyCache.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (pc *ProxyCacheResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProxyCacheModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
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

func (pc *ProxyCacheResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProxyCacheModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ProxyCacheModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := crSDK.UpdateProxyCacheRequest{}
	changed := false

	if !plan.Name.Equal(state.Name) {
		updateReq.Name = plan.Name.ValueStringPointer()
		state.Name = plan.Name
		changed = true
	}
	if !plan.Description.Equal(state.Description) {
		updateReq.Description = plan.Description.ValueStringPointer()
		state.Description = plan.Description
		changed = true
	}
	if !plan.URL.Equal(state.URL) {
		updateReq.URL = plan.URL.ValueStringPointer()
		state.URL = plan.URL
		changed = true
	}
	if !plan.AccessKey.Equal(state.AccessKey) {
		updateReq.AccessKey = plan.AccessKey.ValueStringPointer()
		state.AccessKey = plan.AccessKey
		changed = true
	}
	if !plan.AccessSecret.Equal(state.AccessSecret) {
		updateReq.AccessSecret = plan.AccessSecret.ValueStringPointer()
		state.AccessSecret = plan.AccessSecret
		changed = true
	}

	proxyID := state.ID.ValueString()

	if changed {
		updatedProxy, err := pc.proxyCacheService.Update(ctx, proxyID, updateReq)
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}

		state.UpdatedAt = types.StringPointerValue(&updatedProxy.UpdatedAt)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (pc *ProxyCacheResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProxyCacheModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := pc.proxyCacheService.Delete(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
}

func (pc *ProxyCacheResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(
		resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...,
	)
}
