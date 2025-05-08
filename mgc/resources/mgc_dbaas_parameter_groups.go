package resources

import (
	"context"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DBaaSParametersModel struct {
	EngineID    types.String `tfsdk:"engine_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	ID          types.String `tfsdk:"id"`
}

type DBaaSParametersResource struct {
	ParametersService dbSDK.ParameterGroupService
}

func NewDBaaSParameterGroupsResource() resource.Resource {
	return &DBaaSParametersResource{}
}

func (r *DBaaSParametersResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_parameter_groups"
}

func (r *DBaaSParametersResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.ParametersService = dbSDK.New(&dataConfig.CoreConfig).ParametersGroup()
}

func (r *DBaaSParametersResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DBaaS parameters groups",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the parameters group",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"engine_id": schema.StringAttribute{
				Description: "Unique identifier for the engine",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the parameters group",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the parameters group",
				Optional:    true,
			},
		},
	}
}

func (r *DBaaSParametersResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DBaaSParametersModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.ParametersService.Create(ctx, dbSDK.ParameterGroupCreateRequest{
		EngineID:    data.EngineID.ValueString(),
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.ID = types.StringValue(created.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DBaaSParametersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DBaaSParametersModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	p, err := r.ParametersService.Get(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.Name = types.StringValue(p.Name)
	data.EngineID = types.StringValue(p.EngineID)
	data.Description = types.StringPointerValue(p.Description)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DBaaSParametersResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DBaaSParametersModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.ParametersService.Update(ctx, data.ID.ValueString(), dbSDK.ParameterGroupUpdateRequest{
		Name:        data.Name.ValueStringPointer(),
		Description: data.Description.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DBaaSParametersResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DBaaSParametersModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.ParametersService.Delete(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
}

func (r *DBaaSParametersResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, &DBaaSParametersModel{ID: types.StringValue(req.ID)})...)
}
