package resources

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	computeSdk "github.com/MagaluCloud/mgc-sdk-go/compute"
	tfutil "github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	VmInstanceStatusTimeout = 60 * time.Minute
)

type InstanceStatus string

var imageExpands []string = []string{computeSdk.InstanceImageExpand, computeSdk.InstanceMachineTypeExpand, computeSdk.InstanceNetworkExpand}

var errorStatus = []InstanceStatus{
	StatusCreatingError,
	StatusCreatingNetworkError,
	StatusCreatingErrorQuota,
	StatusCreatingErrorQuotaRam,
	StatusCreatingErrorQuotaVcpu,
	StatusCreatingErrorQuotaDisk,
	StatusCreatingErrorQuotaInstance,
	StatusCreatingErrorQuotaFloatingIP,
	StatusCreatingErrorQuotaNetwork,
	StatusRetypingError,
	StatusRetypingErrorQuotaRam,
	StatusRetypingErrorQuotaVcpu,
	StatusRetypingErrorQuota,
	StatusDeletingError,
}

const (
	StatusAttachingNic                 InstanceStatus = "attaching_nic"
	StatusDetachingNic                 InstanceStatus = "detaching_nic"
	StatusAttachNicPending             InstanceStatus = "attach_nic_pending"
	StatusDetachNicPending             InstanceStatus = "detach_nic_pending"
	StatusProvisioning                 InstanceStatus = "provisioning"
	StatusCreating                     InstanceStatus = "creating"
	StatusCreatingError                InstanceStatus = "creating_error"
	StatusCreatingNetworkError         InstanceStatus = "creating_network_error"
	StatusCreatingErrorQuota           InstanceStatus = "creating_error_quota"
	StatusCreatingErrorQuotaRam        InstanceStatus = "creating_error_quota_ram"
	StatusCreatingErrorQuotaVcpu       InstanceStatus = "creating_error_quota_vcpu"
	StatusCreatingErrorQuotaDisk       InstanceStatus = "creating_error_quota_disk"
	StatusCreatingErrorQuotaInstance   InstanceStatus = "creating_error_quota_instance"
	StatusCreatingErrorQuotaFloatingIP InstanceStatus = "creating_error_quota_floating_ip"
	StatusCreatingErrorQuotaNetwork    InstanceStatus = "creating_error_quota_network"
	StatusCompleted                    InstanceStatus = "completed"
	StatusRetypingPending              InstanceStatus = "retyping_pending"
	StatusRetyping                     InstanceStatus = "retyping"
	StatusRetypingConfirmed            InstanceStatus = "retyping_confirmed"
	StatusRetypingError                InstanceStatus = "retyping_error"
	StatusRetypingErrorQuotaRam        InstanceStatus = "retyping_error_quota_ram"
	StatusRetypingErrorQuotaVcpu       InstanceStatus = "retyping_error_quota_vcpu"
	StatusRetypingErrorQuota           InstanceStatus = "retyping_error_quota"
	StatusStoppingPending              InstanceStatus = "stopping_pending"
	StatusStopping                     InstanceStatus = "stopping"
	StatusSuspendingPending            InstanceStatus = "suspending_pending"
	StatusSuspending                   InstanceStatus = "suspending"
	StatusRebootingPending             InstanceStatus = "rebooting_pending"
	StatusRebooting                    InstanceStatus = "rebooting"
	StatusStartingPending              InstanceStatus = "starting_pending"
	StatusStarting                     InstanceStatus = "starting"
	StatusDeletingPending              InstanceStatus = "deleting_pending"
	StatusDeleting                     InstanceStatus = "deleting"
	StatusDeletingError                InstanceStatus = "deleting_error"
	StatusDeletingNetworkError         InstanceStatus = "deleting_network_error"
	StatusDeleted                      InstanceStatus = "deleted"
)

func (s InstanceStatus) String() string {
	return string(s)
}

func (s InstanceStatus) IsError() bool {
	return slices.Contains(errorStatus, s)
}

func NewVirtualMachineInstancesResource() resource.Resource {
	return &vmInstances{}
}

type vmInstances struct {
	vmInstances computeSdk.InstanceService
}

func (r *vmInstances) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_machine_instances"
}

func (r *vmInstances) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.vmInstances = computeSdk.New(&dataConfig.CoreConfig).Instances()
}

type vmInstancesResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	CreatedAt         types.String `tfsdk:"created_at"`
	SshKeyName        types.String `tfsdk:"ssh_key_name"`
	VpcId             types.String `tfsdk:"vpc_id"`
	MachineType       types.String `tfsdk:"machine_type"`
	Image             types.String `tfsdk:"image"`
	UserData          types.String `tfsdk:"user_data"`
	AvailabilityZone  types.String `tfsdk:"availability_zone"`
	NetworkInterfaces types.List   `tfsdk:"network_interfaces"`
}

type VmInstancesNetworkInterfaceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Ipv4      types.String `tfsdk:"ipv4"`
	LocalIpv4 types.String `tfsdk:"local_ipv4"`
	Ipv6      types.String `tfsdk:"ipv6"`
	Primary   types.Bool   `tfsdk:"primary"`
}

func (r *vmInstances) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Manages virtual machine instances in Magalu Cloud."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the virtual machine instance.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the virtual machine instance.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z0-9]+(?:[-_][a-z0-9]+)*$`),
						"The name must contain only lowercase letters, numbers, underlines and hyphens. Hyphens and underlines cannot be located at the edges either.",
					),
				},
				Required: true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the virtual machine instance was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ssh_key_name": schema.StringAttribute{
				Description: "The name of the SSH key associated with the virtual machine instance. Not required for Windows instances.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Description: "The ID of the VPC the instance is in.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"machine_type": schema.StringAttribute{
				Description: "The machine type name of the virtual machine instance.",
				Required:    true,
			},
			"image": schema.StringAttribute{
				Description: "The image name used for the virtual machine instance.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_data": schema.StringAttribute{
				Description: "User data for instance initialization.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"availability_zone": schema.StringAttribute{
				Description: "The availability zone of the virtual machine instance.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"network_interfaces": schema.ListNestedAttribute{
				Description: "The network interfaces of the virtual machine instance.",
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the network interface.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the network interface.",
							Computed:    true,
						},
						"ipv4": schema.StringAttribute{
							Description: "The IPv4 address of the network interface.",
							Computed:    true,
						},
						"local_ipv4": schema.StringAttribute{
							Description: "The local IPv4 address of the network interface.",
							Computed:    true,
						},
						"ipv6": schema.StringAttribute{
							Description: "The IPv6 address of the network interface.",
							Computed:    true,
						},
						"primary": schema.BoolAttribute{
							Description: "Whether the network interface is primary.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *vmInstances) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := vmInstancesResourceModel{}
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getResult, err := r.vmInstances.Get(ctx, data.ID.ValueString(), imageExpands)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
	convertedData := r.toTerraformModel(ctx, getResult)
	resp.Diagnostics.Append(resp.State.Set(ctx, &convertedData)...)
}

func (r *vmInstances) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	state := vmInstancesResourceModel{}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createdPublicIp := false

	createParams := computeSdk.CreateRequest{
		Name: state.Name.ValueString(),
		MachineType: computeSdk.IDOrName{
			Name: state.MachineType.ValueStringPointer(),
		},
		Image: computeSdk.IDOrName{
			Name: state.Image.ValueStringPointer(),
		},
		UserData:         state.UserData.ValueStringPointer(),
		AvailabilityZone: state.AvailabilityZone.ValueStringPointer(),
		SshKeyName:       state.SshKeyName.ValueStringPointer(),
		Network: &computeSdk.CreateParametersNetwork{
			AssociatePublicIp: &createdPublicIp,
		},
	}

	if state.VpcId.ValueString() != "" {
		createParams.Network.Vpc = &computeSdk.CreateParametersNetworkVpc{
			Vpc: computeSdk.IDOrName{
				ID: state.VpcId.ValueStringPointer(),
			},
		}
	}

	createdId, err := r.vmInstances.Create(ctx, createParams)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
	getResponse, err := r.waitUntilInstanceStatusMatches(ctx, createdId, StatusCompleted)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	convertedResult := r.toTerraformModel(ctx, getResponse)
	resp.Diagnostics.Append(resp.State.Set(ctx, &convertedResult)...)
}

func (r *vmInstances) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	plan := vmInstancesResourceModel{}
	state := &vmInstancesResourceModel{}
	req.State.Get(ctx, state)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Name.ValueString() != plan.Name.ValueString() {
		err := r.vmInstances.Rename(ctx, plan.ID.ValueString(), plan.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
			return
		}
	}

	if state.MachineType.ValueString() != plan.MachineType.ValueString() {
		err := r.vmInstances.Retype(ctx, plan.ID.ValueString(), computeSdk.RetypeRequest{
			MachineType: computeSdk.IDOrName{
				Name: plan.MachineType.ValueStringPointer(),
			},
		})
		if err != nil {
			resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
			return
		}
	}

	getResult, err := r.waitUntilInstanceStatusMatches(ctx, plan.ID.ValueString(), StatusCompleted)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading VM", err.Error())
		return
	}

	convertedResult := r.toTerraformModel(ctx, getResult)
	resp.Diagnostics.Append(resp.State.Set(ctx, &convertedResult)...)
}

func (r *vmInstances) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data vmInstancesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	//false = not remove public ip
	err := r.vmInstances.Delete(ctx, data.ID.ValueString(), false)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
}

func (r *vmInstances) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	model := vmInstancesResourceModel{
		ID:                types.StringValue(req.ID),
		NetworkInterfaces: r.toTerraformNetworkInterfacesList(ctx, []VmInstancesNetworkInterfaceModel{}),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *vmInstances) toTerraformModel(ctx context.Context, server *computeSdk.Instance) *vmInstancesResourceModel {
	interfaces := []VmInstancesNetworkInterfaceModel{}
	if server.Network.Interfaces != nil {
		for _, port := range *server.Network.Interfaces {
			interfaces = append(interfaces, VmInstancesNetworkInterfaceModel{
				ID:        types.StringValue(port.ID),
				Name:      types.StringValue(port.Name),
				Ipv4:      types.StringPointerValue(port.AssociatedPublicIpv4),
				LocalIpv4: types.StringValue(port.IpAddresses.PrivateIpv4),
				Ipv6:      types.StringPointerValue(&port.IpAddresses.PublicIpv6),
				Primary:   types.BoolPointerValue(port.Primary),
			})
		}
	}

	data := vmInstancesResourceModel{
		ID:                types.StringValue(server.ID),
		Name:              types.StringPointerValue(server.Name),
		CreatedAt:         types.StringValue(server.CreatedAt.Format(time.RFC3339)),
		SshKeyName:        types.StringPointerValue(server.SSHKeyName),
		MachineType:       types.StringPointerValue(server.MachineType.Name),
		Image:             types.StringPointerValue(server.Image.Name),
		UserData:          types.StringPointerValue(server.UserData),
		AvailabilityZone:  types.StringPointerValue(server.AvailabilityZone),
		NetworkInterfaces: r.toTerraformNetworkInterfacesList(ctx, interfaces),
	}

	if server.Network.Vpc != nil {
		data.VpcId = types.StringValue(*server.Network.Vpc.ID)
	}

	return &data
}

func (r *vmInstances) waitUntilInstanceStatusMatches(ctx context.Context, instanceID string, status InstanceStatus) (*computeSdk.Instance, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, VmInstanceStatusTimeout)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("timeout waiting for instance %s to reach status %s", instanceID, status)
		case <-time.After(10 * time.Second):
			instance, err := r.vmInstances.Get(ctx, instanceID, imageExpands)
			if err != nil {
				return nil, err
			}
			currentStatus := InstanceStatus(instance.Status)
			if currentStatus == status {
				return instance, nil
			}
			if currentStatus.IsError() {
				if instance.Error != nil && instance.Error.Message != "" {
					return nil, fmt.Errorf("%s", instance.Error.Message)
				}
				return nil, fmt.Errorf("instance %s is in error state: %s", instanceID, currentStatus)
			}
		}
	}
}

func (r *vmInstances) toTerraformNetworkInterfacesList(ctx context.Context, interfaces []VmInstancesNetworkInterfaceModel) types.List {
	listValue, _ := types.ListValueFrom(
		ctx,
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":         types.StringType,
				"name":       types.StringType,
				"ipv4":       types.StringType,
				"local_ipv4": types.StringType,
				"ipv6":       types.StringType,
				"primary":    types.BoolType,
			},
		},
		interfaces,
	)
	return listValue
}
