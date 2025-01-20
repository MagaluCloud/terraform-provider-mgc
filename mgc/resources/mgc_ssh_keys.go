package resources

import (
	"context"

	sdkSSHKeys "github.com/MagaluCloud/magalu/mgc/lib/products/profile/ssh_keys"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/client"
	tfutil "github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	mgcSdk "github.com/MagaluCloud/magalu/mgc/lib"
)

var (
	_ resource.Resource              = &sshKeys{}
	_ resource.ResourceWithConfigure = &sshKeys{}
)

func NewSshKeysResource() resource.Resource {
	return &sshKeys{}
}

type sshKeys struct {
	sdkClient *mgcSdk.Client
	sshKeys   sdkSSHKeys.Service
}

func (r *sshKeys) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_keys"
}

func (r *sshKeys) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if req.ProviderData == nil {
		resp.Diagnostics.AddError(
			"Expected provider config",
			"Please, use with `provider = mgc.example`",
		)
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

	r.sshKeys = sdkSSHKeys.NewService(ctx, r.sdkClient)
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

func setValuesFromCreate(result sdkSSHKeys.CreateResult) *sshKeyModel {
	return &sshKeyModel{
		ID:   types.StringValue(result.Id),
		Name: types.StringValue(result.Name),
		Key:  types.StringValue(result.Key),
	}
}

func setValuesFromGet(result sdkSSHKeys.GetResult) *sshKeyModel {
	return &sshKeyModel{
		ID:   types.StringValue(result.Id),
		Name: types.StringValue(result.Name),
		Key:  types.StringValue(result.Key),
	}
}

func (r *sshKeys) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	plan := &sshKeyModel{}
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)

	getResult, err := r.sshKeys.GetContext(ctx, sdkSSHKeys.GetParameters{
		KeyId: plan.ID.ValueString(),
	},
		tfutil.GetConfigsFromTags(r.sdkClient.Sdk().Config().Get, sdkSSHKeys.GetConfigs{}))

	if err != nil {
		resp.Diagnostics.AddError("Error Reading ssh key", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, setValuesFromGet(getResult))...)
}

func (r *sshKeys) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := &sshKeyModel{}
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createResult, err := r.sshKeys.CreateContext(ctx, sdkSSHKeys.CreateParameters{
		Key:  plan.Key.ValueString(),
		Name: plan.Name.ValueString(),
	},
		tfutil.GetConfigsFromTags(r.sdkClient.Sdk().Config().Get, sdkSSHKeys.CreateConfigs{}))

	if err != nil {
		resp.Diagnostics.AddError("Error creating ssh key", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, setValuesFromCreate(createResult))...)
}

func (r *sshKeys) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *sshKeys) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data sshKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	_, err := r.sshKeys.DeleteContext(ctx,
		sdkSSHKeys.DeleteParameters{
			KeyId: data.ID.ValueString(),
		},
		tfutil.GetConfigsFromTags(r.sdkClient.Sdk().Config().Get, sdkSSHKeys.DeleteConfigs{}),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting ssh key", err.Error())
	}
}
