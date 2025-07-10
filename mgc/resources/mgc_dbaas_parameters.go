package resources

import (
	"context"
	"fmt"
	"strings"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DBaaSParameterModel struct {
	ParameterGroupID types.String  `tfsdk:"parameter_group_id"`
	ID               types.String  `tfsdk:"id"`
	Name             types.String  `tfsdk:"name"`
	Value            types.Dynamic `tfsdk:"value"`
}

type DBaaSParameterResource struct {
	ParameterService dbSDK.ParameterService
}

func NewDBaaSParameterResource() resource.Resource {
	return &DBaaSParameterResource{}
}

func (r *DBaaSParameterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_parameters"
}

func (r *DBaaSParameterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cfg, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider data", "expected tfutil.DataConfig")
		return
	}
	r.ParameterService = dbSDK.New(&cfg.CoreConfig).Parameters()
}

func (r *DBaaSParameterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DBaaS parameter",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the parameter",
				Computed:    true,
			},
			"parameter_group_id": schema.StringAttribute{
				Description: "ID of the parameter group",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the parameter",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.DynamicAttribute{
				Description: "Value of the parameter",
				Required:    true,
			},
		},
	}
}

func (r *DBaaSParameterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DBaaSParameterModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	s, err := tfutil.DynamicToGo[any](data.Value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("value"), "Invalid attribute type", "Invalid attribute type for field `value`, allowed values are [string, bool and numbers]")
		return
	}

	created, err := r.ParameterService.Create(ctx, data.ParameterGroupID.ValueString(), dbSDK.ParameterCreateRequest{
		Name:  data.Name.ValueString(),
		Value: s,
	})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
	data.ID = types.StringValue(created.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DBaaSParameterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DBaaSParameterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	all, err := r.ParameterService.List(ctx, dbSDK.ListParametersOptions{
		ParameterGroupID: data.ParameterGroupID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
	var found *dbSDK.ParameterDetailResponse
	for _, p := range all {
		if p.ID == data.ID.ValueString() {
			found = &p
			break
		}
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	data.Name = types.StringValue(found.Name)

	s, err := tfutil.GoToDynamic(found.Value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("value"), "Invalid attribute type", "Unexpected type for field Value. Please contact support.")
		return
	}

	data.Value = s
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DBaaSParameterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DBaaSParameterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var currentData DBaaSParameterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &currentData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	s, err := tfutil.DynamicToGo[any](data.Value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("value"), "Invalid attribute type", "Invalid attribute type for field `value`, allowed values are [string, bool and numbers]")
		return
	}

	updatedValue := fmt.Sprintf("%v", s)
	if updatedValue != currentData.Value.String() {
		_, err = r.ParameterService.Update(ctx, currentData.ParameterGroupID.ValueString(), currentData.ID.ValueString(), dbSDK.ParameterUpdateRequest{
			Value: s,
		})
		if err != nil {
			resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
			return
		}
	}
	currentData.Value = data.Value
	resp.Diagnostics.Append(resp.State.Set(ctx, &currentData)...)
}

func (r *DBaaSParameterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DBaaSParameterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.ParameterService.Delete(ctx, data.ParameterGroupID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
	}
}

func (r *DBaaSParameterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ",", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import format", "Use `<parameter_group_id>,<parameter_id>`")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("parameter_group_id"), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(parts[1]))...)
}
