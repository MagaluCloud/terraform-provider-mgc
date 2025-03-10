package resources

import (
	"context"

	sdkSSHKeys "github.com/MagaluCloud/mgc-sdk-go/sshkeys"

	tfutil "github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &sshKeys{}
	_ resource.ResourceWithConfigure = &sshKeys{}
)

func NewSshKeysResource() resource.Resource {
	return &sshKeys{}
}

type sshKeys struct {
	sshKeys sdkSSHKeys.KeyService
}

func (r *sshKeys) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_keys"
}

func (r *sshKeys) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.sshKeys = sdkSSHKeys.New(&dataConfig.CoreConfig).Keys()
}

type sshKeyModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Key  types.String `tfsdk:"key"`
}

func (r *sshKeys) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the SSH key",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				Description: "Public key",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Description: "ID of the SSH key",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
	resp.Schema.Description = "Add a new SSH key to your account"
}

func (r *sshKeys) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	plan := &sshKeyModel{}
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)

	result, err := r.sshKeys.Get(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	model := &sshKeyModel{
		ID:   types.StringValue(result.ID),
		Name: types.StringValue(result.Name),
		Key:  types.StringValue(result.Key),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *sshKeys) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := &sshKeyModel{}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createResult, err := r.sshKeys.Create(ctx, sdkSSHKeys.CreateSSHKeyRequest{
		Name: plan.Name.ValueString(),
		Key:  plan.Key.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
	plan.ID = types.StringValue(createResult.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *sshKeys) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("This resource does not support update", "The resource does not support update operations.")
}

func (r *sshKeys) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data sshKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	_, err := r.sshKeys.Delete(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
	}
}

func (r *sshKeys) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
