package resources

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	mgcSdk "github.com/MagaluCloud/magalu/mgc/lib"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdkVmSnapshots "github.com/MagaluCloud/magalu/mgc/lib/products/virtual_machine/snapshots"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/client"
	tfutil "github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
)

var (
	_ resource.Resource              = &vmSnapshots{}
	_ resource.ResourceWithConfigure = &vmSnapshots{}
)

func NewVirtualMachineSnapshotsResource() resource.Resource {
	return &vmSnapshots{}
}

type vmSnapshots struct {
	sdkClient   *mgcSdk.Client
	vmSnapshots sdkVmSnapshots.Service
}

func (r *vmSnapshots) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_machine_snapshots"
}

func (r *vmSnapshots) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	var err error
	var errDetail error
	r.sdkClient, err, errDetail = client.NewSDKClient(req, resp)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			errDetail.Error(),
		)
		return
	}

	r.vmSnapshots = sdkVmSnapshots.NewService(ctx, r.sdkClient)
}

type vmSnapshotsResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	VirtualMachineID types.String `tfsdk:"virtual_machine_id"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
	CreatedAt        types.String `tfsdk:"created_at"`
}

func (r *vmSnapshots) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Operations with snapshots for instances."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the snapshot.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Computed:      true,
			},
			"name": schema.StringAttribute{
				Description:   "The name of the snapshot.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Required:      true,
			},
			"virtual_machine_id": schema.StringAttribute{
				Description:         "The ID of the virtual machine.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "The ID of the virtual machine.",
				Required:            true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the snapshot was last updated.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description:   "The timestamp when the snapshot was created.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Computed:      true,
			},
		},
	}
}

func (r *vmSnapshots) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	//do nothing
}

func (r *vmSnapshots) getVmSnapshot(ctx context.Context, id string) (sdkVmSnapshots.GetResult, error) {
	getResult, err := r.vmSnapshots.GetContext(ctx,
		sdkVmSnapshots.GetParameters{
			Id: id,
		},
		tfutil.GetConfigsFromTags(r.sdkClient.Sdk().Config().Get, sdkVmSnapshots.GetConfigs{}))
	if err != nil {
		return sdkVmSnapshots.GetResult{}, err
	}
	return getResult, nil
}

func (r *vmSnapshots) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := &vmSnapshotsResourceModel{}
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	getResult, err := r.getVmSnapshot(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VM",
			"Could not read VM ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	data.ID = types.StringValue(getResult.Id)
	data.Name = types.StringValue(*getResult.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmSnapshots) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	plan := &vmSnapshotsResourceModel{}
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createParams := sdkVmSnapshots.CreateParameters{
		Name: plan.Name.ValueString(),
		Instance: sdkVmSnapshots.CreateParametersInstance{
			Id: plan.VirtualMachineID.ValueString(),
		},
	}

	result, err := r.vmSnapshots.CreateContext(ctx, createParams, tfutil.GetConfigsFromTags(r.sdkClient.Sdk().Config().Get, sdkVmSnapshots.CreateConfigs{}))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating VM Snapshot",
			"Could not create VM Snapshot: "+err.Error(),
		)
	}

	plan.Name = types.StringValue(plan.Name.ValueString())
	plan.ID = types.StringValue(result.Id)

	plan.CreatedAt = types.StringValue(time.Now().Format(time.RFC850))
	plan.UpdatedAt = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *vmSnapshots) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *vmSnapshots) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data vmSnapshotsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	err := r.vmSnapshots.DeleteContext(ctx,
		sdkVmSnapshots.DeleteParameters{
			Id: data.ID.ValueString(),
		},
		tfutil.GetConfigsFromTags(r.sdkClient.Sdk().Config().Get, sdkVmSnapshots.DeleteConfigs{}))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting VM Snapshot",
			"Could not delete VM Snapshot "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

}
