package virtualmachines

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	vmSDK "github.com/MagaluCloud/mgc-sdk-go/compute"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceVmMachineType{}

type DataSourceVmMachineType struct {
	vmType vmSDK.InstanceTypeService
}

type MachineTypeModel struct {
	ID                types.String   `tfsdk:"id"`
	Name              types.String   `tfsdk:"name"`
	Disk              types.Int64    `tfsdk:"disk"`
	Ram               types.Int64    `tfsdk:"ram"`
	VCPU              types.Int64    `tfsdk:"vcpu"`
	GPU               types.Int64    `tfsdk:"gpu"`
	AvailabilityZones []types.String `tfsdk:"availability_zones"`
}

type MachineTypesModel struct {
	MachineTypes []MachineTypeModel `tfsdk:"machine_types"`
}

func NewDataSourceVmMachineType() datasource.DataSource {
	return &DataSourceVmMachineType{}
}

func (r *DataSourceVmMachineType) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_machine_types"
}

func (r *DataSourceVmMachineType) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.vmType = vmSDK.New(&dataConfig.CoreConfig).InstanceTypes()
}

func (r *DataSourceVmMachineType) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"machine_types": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of available VM machine-types.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "ID of machine-type.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of type.",
						},
						"disk": schema.Int64Attribute{
							Computed:    true,
							Description: "Disk",
						},
						"ram": schema.Int64Attribute{
							Computed:    true,
							Description: "Ram",
						},
						"vcpu": schema.Int64Attribute{
							Computed:    true,
							Description: "VCpu",
						},
						"gpu": schema.Int64Attribute{
							Computed:    true,
							Description: "GPU",
						},
						"availability_zones": schema.ListAttribute{
							Computed:    true,
							Description: "The availability zones of the machine-type.",
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
	resp.Schema.Description = "Get the available virtual-machine types."
}

const typeActive string = "active"

func (r *DataSourceVmMachineType) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MachineTypesModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	sdkOutput, err := r.vmType.List(ctx, vmSDK.InstanceTypeListOptions{})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	for _, typ := range sdkOutput {
		if typ.Status != typeActive {
			continue
		}

		azs := []types.String{}
		if typ.AvailabilityZones != nil {
			for _, az := range *typ.AvailabilityZones {
				azs = append(azs, types.StringValue(az))
			}
		}

		data.MachineTypes = append(data.MachineTypes, MachineTypeModel{
			ID:                types.StringValue(typ.ID),
			Name:              types.StringValue(typ.Name),
			Disk:              types.Int64Value(int64(typ.Disk)),
			Ram:               types.Int64Value(int64(typ.RAM)),
			VCPU:              types.Int64Value(int64(typ.VCPUs)),
			GPU:               types.Int64PointerValue(utils.ConvertIntPointerToInt64Pointer(typ.GPU)),
			AvailabilityZones: azs,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
