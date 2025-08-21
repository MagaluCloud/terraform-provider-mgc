package virtualmachines

import (
	"context"
	"time"

	vmSDK "github.com/MagaluCloud/mgc-sdk-go/compute"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceVmSnapshots{}

type DataSourceVmSnapshots struct {
	vmSnapshot vmSDK.SnapshotService
}

func (r *DataSourceVmSnapshots) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_machine_snapshots"
}

func NewDataSourceVmSnapshots() datasource.DataSource {
	return &DataSourceVmSnapshots{}
}

type vmSnapshotsModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	InstanceID       types.String `tfsdk:"instance_id"`
	VirtualMachineID types.String `tfsdk:"virtual_machine_id"`
	ImageID          types.String `tfsdk:"image_id"`
	Size             types.Int64  `tfsdk:"size"`
	State            types.String `tfsdk:"state"`
	Status           types.String `tfsdk:"status"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
	CreatedAt        types.String `tfsdk:"created_at"`
}

type vmSnapshotsListModel struct {
	Snapshots []vmSnapshotsModel `tfsdk:"snapshots"`
}

func (r *DataSourceVmSnapshots) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This data source provides a list of virtual machine snapshots.",
		Attributes: map[string]schema.Attribute{
			"snapshots": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of available VM machine-types.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the snapshot.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the snapshot.",
							Computed:    true,
						},
						"virtual_machine_id": schema.StringAttribute{
							Description: "The ID of the virtual machine.",
							Computed:    true,
						},
						"instance_id": schema.StringAttribute{
							Description: "The ID of the instance.",
							Computed:    true,
						},
						"image_id": schema.StringAttribute{
							Description: "The ID of the image.",
							Computed:    true,
						},
						"size": schema.Int64Attribute{
							Description: "The size of the snapshot.",
							Computed:    true,
						},
						"state": schema.StringAttribute{
							Description: "The state of the snapshot.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The status of the snapshot.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "The timestamp when the snapshot was last updated.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "The timestamp when the snapshot was created.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *DataSourceVmSnapshots) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.vmSnapshot = vmSDK.New(&dataConfig.CoreConfig).Snapshots()
}

func (r *DataSourceVmSnapshots) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data vmSnapshotsListModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	snapList, err := r.vmSnapshot.List(ctx, vmSDK.ListOptions{})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	for _, snap := range snapList {
		data.Snapshots = append(data.Snapshots, vmSnapshotsModel{
			ID:               types.StringValue(snap.ID),
			Name:             types.StringValue(snap.Name),
			InstanceID:       types.StringValue(snap.Instance.ID),
			VirtualMachineID: types.StringPointerValue(snap.Instance.MachineType.ID),
			ImageID:          types.StringPointerValue(snap.Instance.Image.ID),
			Size:             types.Int64Value(int64(snap.Size)),
			State:            types.StringValue(snap.State),
			Status:           types.StringValue(snap.Status),
			UpdatedAt:        types.StringValue(snap.UpdatedAt.Format(time.RFC3339)),
			CreatedAt:        types.StringValue(snap.CreatedAt.Format(time.RFC3339)),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
