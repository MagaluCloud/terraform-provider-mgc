package resources

import (
	"context"
	"errors"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DBaaSParametersModel struct {
	EngineName    types.String `tfsdk:"engine_name"`
	EngineVersion types.String `tfsdk:"engine_version"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	ID            types.String `tfsdk:"id"`
}

type DBaaSParameterGroupsResource struct {
	ParametersService dbSDK.ParameterGroupService
	dbaasEngines      dbSDK.EngineService
}

func NewDBaaSParameterGroupsResource() resource.Resource {
	return &DBaaSParameterGroupsResource{}
}

func (r *DBaaSParameterGroupsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_parameter_groups"
}

func (r *DBaaSParameterGroupsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.ParametersService = dbSDK.New(&dataConfig.CoreConfig).ParametersGroup()
	r.dbaasEngines = dbSDK.New(&dataConfig.CoreConfig).Engines()
}

func (r *DBaaSParameterGroupsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"engine_name": schema.StringAttribute{
				Description: "Type of database engine to use (e.g., 'mysql', 'postgresql'). Cannot be changed after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"engine_version": schema.StringAttribute{
				Description: "Version of the database engine (e.g., '8.0', '13.3'). Must be compatible with the selected engine_name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
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

func (r *DBaaSParameterGroupsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DBaaSParametersModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	engineID, err := r.validateAndGetEngineID(ctx, data.EngineName.ValueString(), data.EngineVersion.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid engine name", err.Error())
		return
	}

	created, err := r.ParametersService.Create(ctx, dbSDK.ParameterGroupCreateRequest{
		EngineID:    engineID,
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

func (r *DBaaSParameterGroupsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
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

	engineName, engineVersion, err := r.getEngineNameAndVersionByID(ctx, p.EngineID)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.Name = types.StringValue(p.Name)
	data.EngineName = types.StringValue(engineName)
	data.EngineVersion = types.StringValue(engineVersion)
	data.Description = types.StringPointerValue(p.Description)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DBaaSParameterGroupsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DBaaSParametersModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var currentData DBaaSParametersModel
	resp.Diagnostics.Append(req.State.Get(ctx, &currentData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	currentData.Description = data.Description
	currentData.Name = data.Name
	_, err := r.ParametersService.Update(ctx, currentData.ID.ValueString(), dbSDK.ParameterGroupUpdateRequest{
		Name:        data.Name.ValueStringPointer(),
		Description: data.Description.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &currentData)...)
}

func (r *DBaaSParameterGroupsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
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

func (r *DBaaSParameterGroupsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, &DBaaSParametersModel{ID: types.StringValue(req.ID)})...)
}

func (r *DBaaSParameterGroupsResource) validateAndGetEngineID(ctx context.Context, engineName string, engineVersion string) (string, error) {
	active := "ACTIVE"
	engines, err := r.dbaasEngines.List(ctx, dbSDK.ListEngineOptions{
		Status: &active,
	})
	if err != nil {
		return "", err
	}
	for _, engine := range engines {
		if engine.Name == engineName && engine.Version == engineVersion {
			return engine.ID, nil
		}
	}
	return "", errors.New("engine not found")
}

func (r *DBaaSParameterGroupsResource) getEngineNameAndVersionByID(ctx context.Context, engineID string) (name string, version string, err error) {
	engine, err := r.dbaasEngines.Get(ctx, engineID)
	if err != nil {
		return "", "", err
	}
	return engine.Name, engine.Version, nil
}
