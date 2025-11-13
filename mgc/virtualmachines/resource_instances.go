package virtualmachines

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"

	computeSdk "github.com/MagaluCloud/mgc-sdk-go/compute"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	VmInstanceStatusTimeout = 60 * time.Minute
)

type InstanceStatus string

var imageExpands []computeSdk.InstanceExpand = []computeSdk.InstanceExpand{computeSdk.InstanceImageExpand,
	computeSdk.InstanceMachineTypeExpand, computeSdk.InstanceNetworkExpand}

var errorStatus = []InstanceStatus{
	StatusCreatingError,
	StatusCreatingNetworkError,
	StatusCreatingErrorCapacity,
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
	StatusCreatingErrorCapacity        InstanceStatus = "creating_error_capacity"
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
	vmSnapshots computeSdk.SnapshotService
}

func (r *vmInstances) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_machine_instances"
}

func (r *vmInstances) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.vmInstances = computeSdk.New(&dataConfig.CoreConfig).Instances()
	r.vmSnapshots = computeSdk.New(&dataConfig.CoreConfig).Snapshots()
}

type vmInstancesResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	CreatedAt              types.String `tfsdk:"created_at"`
	SshKeyName             types.String `tfsdk:"ssh_key_name"`
	VpcID                  types.String `tfsdk:"vpc_id"`
	MachineType            types.String `tfsdk:"machine_type"`
	Image                  types.String `tfsdk:"image"`
	UserData               types.String `tfsdk:"user_data"`
	AvailabilityZone       types.String `tfsdk:"availability_zone"`
	NetworkInterfaces      types.List   `tfsdk:"network_interfaces"`
	NetworkInterfaceId     types.String `tfsdk:"network_interface_id"`
	AllocatePublicIpv4     types.Bool   `tfsdk:"allocate_public_ipv4"`
	CreationSecurityGroups types.List   `tfsdk:"creation_security_groups"`
	LocalIPv4              types.String `tfsdk:"local_ipv4"`
	IPv6                   types.String `tfsdk:"ipv6"`
	IPv4                   types.String `tfsdk:"ipv4"`
	SnapshotID             types.String `tfsdk:"snapshot_id"`
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
				Description: "The VPC ID where the primary network interface will be created.",
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
				Description: `The image name used for the virtual machine instance.
			 This attribute is required when not creating the instance from a snapshot (i.e., when "snapshot_id" is not set).
			 If "snapshot_id" is provided, the snapshot will be used instead of an image.`,
				Optional: true,
				Computed: true,
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
			"network_interface_id": schema.StringAttribute{
				Description: `The primary network interface ID is the primary interface used for network traffic that will be associated with the instance.
If not specified, a new network interface will be created in the specified VPC or in the default VPC if no VPC is specified.
Read the documentation guides for more details.`,
				// Optional: true,
				Computed:   true,
				Validators: []validator.String{
					// stringvalidator.ConflictsWith(path.MatchRoot("vpc_id")),
					// stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					// stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"allocate_public_ipv4": schema.BoolAttribute{
				Description: `If true, the primary network interface will be created with a public IPv4 address.
A Public IPv4 address resource will be created and associated with you tenant, when deleting the instance the Public IPv4 will not be deleted and charges may apply.
If false, the primary network interface will be created without a public IPv4 address.
Default is false.
This attribute can only be used when "network_interface_id" is not set.`,
				Optional:  true,
				WriteOnly: true,
				Validators: []validator.Bool{
					boolvalidator.ConflictsWith(path.MatchRoot("network_interface_id")),
				},
			},
			"creation_security_groups": schema.ListAttribute{
				Description: `List of security group IDs to be associated with the primary network interface on creation.
If not specified, the default security group of the VPC will be used.
For manage security groups after the instance creation, use the network resources.
Find out more in the documentation guides.
This attribute can only be used when "network_interface_id" is not set.`,
				ElementType: types.StringType,
				Optional:    true,
				WriteOnly:   true,
				Validators: []validator.List{
					listvalidator.ConflictsWith(path.MatchRoot("network_interface_id")),
					listvalidator.SizeAtLeast(1),
					listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"local_ipv4": schema.StringAttribute{
				Description: "The primary network interface IPv4 address of the virtual machine instance.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ipv6": schema.StringAttribute{
				Description: "The primary network interface IPv6 address of the virtual machine instance.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ipv4": schema.StringAttribute{
				Description: "The primary network interface public IPv4 address of the virtual machine instance.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"snapshot_id": schema.StringAttribute{
				Description:   "The snapshot ID used to create the virtual machine instance. If set, the snapshot will be used instead of an image.",
				Optional:      true,
				WriteOnly:     true,
				PlanModifiers: []planmodifier.String{},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("image")),
				},
			},
			"network_interfaces": schema.ListNestedAttribute{
				Description: "The network interfaces attached to the virtual machine instance.",
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
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
	convertedData := r.toTerraformModel(ctx, getResult)
	resp.Diagnostics.Append(resp.State.Set(ctx, &convertedData)...)
}

func (r *vmInstances) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	state := vmInstancesResourceModel{}
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.AllocatePublicIpv4.ValueBoolPointer() == nil {
		state.AllocatePublicIpv4 = types.BoolValue(false)
	}

	var sg *[]computeSdk.CreateParametersNetworkInterfaceWithID
	if !state.CreationSecurityGroups.IsNull() {
		var sgIDs []string
		resp.Diagnostics.Append(state.CreationSecurityGroups.ElementsAs(ctx, &sgIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		items := make([]computeSdk.CreateParametersNetworkInterfaceWithID, 0, len(sgIDs))
		for _, id := range sgIDs {
			items = append(items, computeSdk.CreateParametersNetworkInterfaceWithID{
				ID: id,
			})
		}
		sg = &items
	}

	createNetwork := computeSdk.CreateParametersNetwork{
		AssociatePublicIp: state.AllocatePublicIpv4.ValueBoolPointer(),
		Interface:         &computeSdk.CreateParametersNetworkInterface{
			// ID: state.NetworkInterfaceId.ValueStringPointer(),
		},
	}
	if sg != nil {
		createNetwork.Interface.SecurityGroups = sg
	}
	if state.VpcID.ValueString() != "" {
		createNetwork.Vpc = &computeSdk.IDOrName{
			ID: state.VpcID.ValueStringPointer(),
		}
	}

	var createdID string
	var err error

	if state.SnapshotID.ValueString() != "" {
		createdID, err = r.vmSnapshots.Restore(ctx, state.SnapshotID.ValueString(), computeSdk.RestoreSnapshotRequest{
			Name: state.Name.ValueString(),
			MachineType: computeSdk.IDOrName{
				Name: state.MachineType.ValueStringPointer(),
			},
			SSHKeyName:       state.SshKeyName.ValueStringPointer(),
			UserData:         state.UserData.ValueStringPointer(),
			AvailabilityZone: state.AvailabilityZone.ValueStringPointer(),
			Network:          &createNetwork,
		})
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
	} else {
		if state.Image.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(path.Root("image"),
				"The image attribute must be specified when not restoring from a snapshot.",
				"Either set the 'image' attribute to the name of an image to use for creating the instance,"+
					"or set 'snapshot_id' to restore the instance from a snapshot. Leaving both empty is not supported.")
		}
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
			Network:          &createNetwork,
		}

		createdID, err = r.vmInstances.Create(ctx, createParams)
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
	}

	getResponse, err := r.waitUntilInstanceStatusMatches(ctx, createdID, StatusCompleted)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
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
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
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
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
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
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	_, err = r.waitUntilInstanceStatusMatches(ctx, data.ID.ValueString(), StatusDeleted)
	if err != nil {
		switch e := err.(type) {
		case *clientSDK.HTTPError:
			if e.StatusCode == http.StatusNotFound {
				return
			}
		default:
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
	}
}

func (r *vmInstances) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	model := vmInstancesResourceModel{
		ID:                     types.StringValue(req.ID),
		Name:                   types.StringUnknown(),
		CreatedAt:              types.StringUnknown(),
		SshKeyName:             types.StringUnknown(),
		VpcID:                  types.StringUnknown(),
		MachineType:            types.StringUnknown(),
		Image:                  types.StringUnknown(),
		UserData:               types.StringUnknown(),
		AvailabilityZone:       types.StringUnknown(),
		NetworkInterfaces:      r.toTerraformNetworkInterfacesList(ctx, []VmInstancesNetworkInterfaceModel{}),
		NetworkInterfaceId:     types.StringUnknown(),
		AllocatePublicIpv4:     types.BoolNull(),
		CreationSecurityGroups: types.ListNull(types.StringType),
		LocalIPv4:              types.StringUnknown(),
		IPv6:                   types.StringUnknown(),
		IPv4:                   types.StringUnknown(),
		SnapshotID:             types.StringUnknown(),
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
		data.VpcID = types.StringValue(*server.Network.Vpc.ID)
	}

	data.NetworkInterfaceId = types.StringNull()
	data.LocalIPv4 = types.StringNull()
	data.IPv6 = types.StringNull()
	data.IPv4 = types.StringNull()
	for _, ni := range interfaces {
		if ni.Primary.ValueBool() {
			data.NetworkInterfaceId = ni.ID
			data.LocalIPv4 = ni.LocalIpv4
			data.IPv6 = ni.Ipv6
			data.IPv4 = ni.Ipv4
			break
		}
	}

	data.AllocatePublicIpv4 = types.BoolNull()
	data.CreationSecurityGroups = types.ListNull(types.StringType)
	data.SnapshotID = types.StringNull()

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
