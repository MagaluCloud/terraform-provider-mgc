package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	computeSdk "github.com/MagaluCloud/mgc-sdk-go/compute"
	tfutil "github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
)

func NewVirtualMachineInterfaceAttachResource() resource.Resource {
	return &vmInterfaceAttach{}
}

type vmInterfaceAttach struct {
	vmInstance computeSdk.InstanceService
}

type vmInterfaceAttachResourceModel struct {
	InterfaceID types.String `tfsdk:"interface_id"`
	InstanceID  types.String `tfsdk:"instance_id"`
}

func (r *vmInterfaceAttach) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_machine_interface_attach"
}

func (r *vmInterfaceAttach) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.vmInstance = computeSdk.New(&dataConfig.CoreConfig).Instances()
}

func (r *vmInterfaceAttach) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Attaches a network interface to a VM instance.",
		Attributes: map[string]schema.Attribute{
			"interface_id": schema.StringAttribute{
				Description: "ID of the network interface to attach.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: "ID of the VM instance to attach the interface to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *vmInterfaceAttach) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data vmInterfaceAttachResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.vmInstance.AttachNetworkInterface(ctx, computeSdk.NICRequest{
		Instance: computeSdk.IDOrName{
			ID: data.InstanceID.ValueStringPointer(),
		},
		Network: computeSdk.NICRequestInterface{
			Interface: computeSdk.IDOrName{
				ID: data.InterfaceID.ValueStringPointer(),
			},
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmInterfaceAttach) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data vmInterfaceAttachResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getInstance, err := r.vmInstance.Get(ctx, data.InstanceID.ValueString(), []string{computeSdk.InstanceNetworkExpand})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	if getInstance.Network == nil || getInstance.Network.Interfaces == nil || len(*getInstance.Network.Interfaces) == 0 {
		resp.Diagnostics.AddError("No network interfaces found for VM instance", "")
		return
	}

	hasInterfaceId := false

	for _, iface := range *getInstance.Network.Interfaces {
		if iface.ID == data.InterfaceID.ValueString() {
			hasInterfaceId = true
			break
		}
	}

	if !hasInterfaceId {
		resp.Diagnostics.AddError("Network interface not found on VM instance", "")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmInterfaceAttach) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Update is not supported for this resource")
}

func (r *vmInterfaceAttach) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data vmInterfaceAttachResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.vmInstance.DetachNetworkInterface(ctx, computeSdk.NICRequest{
		Instance: computeSdk.IDOrName{
			ID: data.InstanceID.ValueStringPointer(),
		},
		Network: computeSdk.NICRequestInterface{
			Interface: computeSdk.IDOrName{
				ID: data.InterfaceID.ValueStringPointer(),
			},
		},
	})

	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
	}
}
