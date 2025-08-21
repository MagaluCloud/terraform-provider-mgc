package database

import (
	"context"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DataSourceDbSnapshots struct {
	dbaasInstances dbSDK.InstanceService
}

type dbSnapshotsModel struct {
	InstanceID types.String      `tfsdk:"instance_id"`
	Snapshots  []dbSnapshotModel `tfsdk:"snapshots"`
}

func NewDataSourceDbaasInstancesSnapshots() datasource.DataSource {
	return &DataSourceDbSnapshots{}
}

func (r *DataSourceDbSnapshots) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_instances_snapshots"
}

func (r *DataSourceDbSnapshots) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.dbaasInstances = dbSDK.New(&dataConfig.CoreConfig).Instances()
}

func (r *DataSourceDbSnapshots) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all snapshots for a database instance.",
		Attributes: map[string]schema.Attribute{
			"instance_id": schema.StringAttribute{
				Description: "ID of the instance",
				Required:    true,
			},
			"snapshots": schema.ListNestedAttribute{
				Description: "List of snapshots",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "ID of the snapshot",
							Computed:    true,
						},
						"instance_id": schema.StringAttribute{
							Description: "ID of the instance",
							Required:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the snapshot",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Description of the snapshot",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "Creation timestamp",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "Status of the snapshot",
							Computed:    true,
						},
						"size": schema.Int64Attribute{
							Description: "Size of the snapshot in bytes",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *DataSourceDbSnapshots) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dbSnapshotsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	snapshots, err := r.dbaasInstances.ListSnapshots(ctx, data.InstanceID.ValueString(), dbSDK.ListSnapshotOptions{})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	var snapshotModels []dbSnapshotModel
	for _, snapshot := range snapshots {
		snapshotModels = append(snapshotModels, dbSnapshotModel{
			ID:          types.StringValue(snapshot.ID),
			Name:        types.StringValue(snapshot.Name),
			InstanceID:  data.InstanceID,
			Description: types.StringValue(snapshot.Description),
			CreatedAt:   types.StringValue(*utils.ConvertTimeToRFC3339(&snapshot.CreatedAt)),
			Status:      types.StringValue(string(snapshot.Status)),
			Size:        types.Int64PointerValue(utils.ConvertIntPointerToInt64Pointer(&snapshot.AllocatedSize)),
		})
	}

	data.Snapshots = snapshotModels
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
