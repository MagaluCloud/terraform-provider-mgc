package virtualmachines

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	vmSDK "github.com/MagaluCloud/mgc-sdk-go/compute"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceVmInstance{}

type DataSourceVmInstance struct {
	vmInstance vmSDK.InstanceService
}

func NewDataSourceVmInstance() datasource.DataSource {
	return &DataSourceVmInstance{}
}

func (r *DataSourceVmInstance) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_machine_instance"
}

func (r *DataSourceVmInstance) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.vmInstance = vmSDK.New(&dataConfig.CoreConfig).Instances()
}

func (r *DataSourceVmInstance) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "ID of machine-type.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of instance.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp of the instance.",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp of the instance.",
			},
			"image_id": schema.StringAttribute{
				Computed:    true,
				Description: "Image ID of instance.",
			},
			"image_name": schema.StringAttribute{
				Computed:    true,
				Description: "Image name of instance.",
			},
			"image_platform": schema.StringAttribute{
				Computed:    true,
				Description: "Image platform type.",
			},
			"machine_type_id": schema.StringAttribute{
				Computed:    true,
				Description: "Machine type ID of instance.",
			},
			"machine_type_name": schema.StringAttribute{
				Computed:    true,
				Description: "Machine type name.",
			},
			"machine_type_disk": schema.Int64Attribute{
				Computed:    true,
				Description: "Machine type disk size.",
			},
			"machine_type_ram": schema.Int64Attribute{
				Computed:    true,
				Description: "Machine type RAM size.",
			},
			"machine_type_vcpus": schema.Int64Attribute{
				Computed:    true,
				Description: "Machine type vCPUs count.",
			},
			"vpc_id": schema.StringAttribute{
				Computed:    true,
				Description: "VPC ID.",
			},
			"vpc_name": schema.StringAttribute{
				Computed:    true,
				Description: "VPC name.",
			},
			"ssh_key_name": schema.StringAttribute{
				Computed:    true,
				Description: "SSH Key name.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of instance.",
			},
			"state": schema.StringAttribute{
				Computed:    true,
				Description: "State of instance.",
			},
			"user_data": schema.StringAttribute{
				Computed:    true,
				Description: "User data of instance.",
			},
			"availability_zone": schema.StringAttribute{
				Computed:    true,
				Description: "Availability zone of instance.",
			},
			"labels": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Labels associated with the instance.",
			},
			"error_message": schema.StringAttribute{
				Computed:    true,
				Description: "Error message if any.",
			},
			"error_slug": schema.StringAttribute{
				Computed:    true,
				Description: "Error slug if any.",
			},
			"interfaces": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Interface ID.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Interface name.",
						},
						"primary": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether this is the primary interface.",
						},
						"public_ipv4": schema.StringAttribute{
							Computed:    true,
							Description: "Public IPv4 address.",
						},
						"local_ipv4": schema.StringAttribute{
							Computed:    true,
							Description: "Local IPv4 address.",
						},
						"public_ipv6": schema.StringAttribute{
							Computed:    true,
							Description: "Public IPv6 address.",
						},
						"security_groups": schema.ListAttribute{
							ElementType: types.StringType,
							Computed:    true,
							Description: "Security groups associated with the interface.",
						},
					},
				},
				Description: "Network interfaces attached to the instance.",
			},
		},
		Description: "Get the available virtual-machine instance details",
	}
}

type NetworkInterfaceModel struct {
	ID             types.String   `tfsdk:"id"`
	Name           types.String   `tfsdk:"name"`
	Primary        types.Bool     `tfsdk:"primary"`
	PublicIPv4     types.String   `tfsdk:"public_ipv4"`
	LocalIPv4      types.String   `tfsdk:"local_ipv4"`
	IPv6           types.String   `tfsdk:"public_ipv6"`
	SecurityGroups []types.String `tfsdk:"security_groups"`
}

type VMInstanceModel struct {
	ID               types.String            `tfsdk:"id"`
	Name             types.String            `tfsdk:"name"`
	CreatedAt        types.String            `tfsdk:"created_at"`
	UpdatedAt        types.String            `tfsdk:"updated_at"`
	ImageID          types.String            `tfsdk:"image_id"`
	ImageName        types.String            `tfsdk:"image_name"`
	ImagePlatform    types.String            `tfsdk:"image_platform"`
	MachineTypeID    types.String            `tfsdk:"machine_type_id"`
	MachineTypeName  types.String            `tfsdk:"machine_type_name"`
	MachineTypeDisk  types.Int64             `tfsdk:"machine_type_disk"`
	MachineTypeRAM   types.Int64             `tfsdk:"machine_type_ram"`
	MachineTypeVCPUs types.Int64             `tfsdk:"machine_type_vcpus"`
	VPCID            types.String            `tfsdk:"vpc_id"`
	VPCName          types.String            `tfsdk:"vpc_name"`
	SshKeyName       types.String            `tfsdk:"ssh_key_name"`
	Status           types.String            `tfsdk:"status"`
	State            types.String            `tfsdk:"state"`
	UserData         types.String            `tfsdk:"user_data"`
	AvailabilityZone types.String            `tfsdk:"availability_zone"`
	ErrorMessage     types.String            `tfsdk:"error_message"`
	ErrorSlug        types.String            `tfsdk:"error_slug"`
	Labels           []types.String          `tfsdk:"labels"`
	Interfaces       []NetworkInterfaceModel `tfsdk:"interfaces"`
}

func (r *DataSourceVmInstance) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VMInstanceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.vmInstance.Get(ctx, data.ID.ValueString(), []string{vmSDK.InstanceNetworkExpand, vmSDK.InstanceImageExpand, vmSDK.InstanceMachineTypeExpand})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	var interfaces []NetworkInterfaceModel
	if instance.Network != nil && instance.Network.Interfaces != nil {
		for _, iface := range *instance.Network.Interfaces {
			networkInterface := NetworkInterfaceModel{
				ID:         types.StringValue(iface.ID),
				Name:       types.StringValue(iface.Name),
				Primary:    types.BoolPointerValue(iface.Primary),
				PublicIPv4: types.StringPointerValue(iface.AssociatedPublicIpv4),
				LocalIPv4:  types.StringValue(iface.IpAddresses.PrivateIpv4),
				IPv6:       types.StringValue(iface.IpAddresses.PublicIpv6),
			}
			if iface.SecurityGroups != nil {
				var secGroups []types.String
				for _, sg := range *iface.SecurityGroups {
					secGroups = append(secGroups, types.StringValue(sg))
				}
				networkInterface.SecurityGroups = secGroups
			}
			interfaces = append(interfaces, networkInterface)
		}
	}

	labels := []types.String{}
	if instance.Labels != nil {
		for _, label := range *instance.Labels {
			labels = append(labels, types.StringValue(label))
		}
	}

	data = VMInstanceModel{
		ID:               types.StringValue(instance.ID),
		Name:             types.StringValue(*instance.Name),
		CreatedAt:        types.StringValue(instance.CreatedAt.Format(time.RFC3339)),
		ImageID:          types.StringValue(instance.Image.ID),
		ImageName:        types.StringPointerValue(instance.Image.Name),
		ImagePlatform:    types.StringPointerValue(instance.Image.Platform),
		MachineTypeID:    types.StringValue(instance.MachineType.ID),
		MachineTypeName:  types.StringPointerValue(instance.MachineType.Name),
		MachineTypeDisk:  types.Int64PointerValue(utils.ConvertIntPointerToInt64Pointer(instance.MachineType.Disk)),
		MachineTypeRAM:   types.Int64PointerValue(utils.ConvertIntPointerToInt64Pointer(instance.MachineType.Ram)),
		MachineTypeVCPUs: types.Int64PointerValue(utils.ConvertIntPointerToInt64Pointer(instance.MachineType.Vcpus)),
		VPCID:            types.StringPointerValue(instance.Network.Vpc.ID),
		VPCName:          types.StringPointerValue(instance.Network.Vpc.Name),
		SshKeyName:       types.StringPointerValue(instance.SSHKeyName),
		Status:           types.StringValue(instance.Status),
		State:            types.StringValue(instance.State),
		UserData:         types.StringPointerValue(instance.UserData),
		AvailabilityZone: types.StringPointerValue(instance.AvailabilityZone),
		Labels:           labels,
		Interfaces:       interfaces,
	}

	if instance.UpdatedAt != nil {
		data.UpdatedAt = types.StringValue(instance.UpdatedAt.Format(time.RFC3339))
	}

	if instance.Error != nil {
		data.ErrorMessage = types.StringValue(instance.Error.Message)
		data.ErrorSlug = types.StringValue(instance.Error.Slug)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
