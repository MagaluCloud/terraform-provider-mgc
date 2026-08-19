package database

import (
	"context"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DBaaSClusterSnapshotsDataSource struct {
	dbaasClusters dbSDK.ClusterService
}

type dbClusterSnapshotsModel struct {
	ClusterID types.String             `tfsdk:"cluster_id"`
	Snapshots []dbClusterSnapshotModel `tfsdk:"snapshots"`
}

func NewDBaaSClusterSnapshotsDataSource() datasource.DataSource {
	return &DBaaSClusterSnapshotsDataSource{}
}

func (r *DBaaSClusterSnapshotsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_clusters_snapshots"
}

func (r *DBaaSClusterSnapshotsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.dbaasClusters = dbSDK.New(dataConfig.CoreFor(utils.ServiceDatabase)).Clusters()
}

func (r *DBaaSClusterSnapshotsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the list of snapshots for a DBaaS (Database-as-a-Service) cluster.",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				Description: "ID of the DBaaS cluster whose snapshots will be listed.",
				Required:    true,
			},
			"snapshots": schema.ListNestedAttribute{
				Description: "List of snapshots found for the cluster.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "ID of the snapshot.",
							Computed:    true,
						},
						"cluster_id": schema.StringAttribute{
							Description: "ID of the DBaaS cluster the snapshot belongs to.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the snapshot.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Description of the snapshot.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "Timestamp of when the snapshot was created.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "Current status of the snapshot ([PENDING, CREATING, AVAILABLE, RESTORING, ERROR, DELETING, DELETED]).",
							Computed:    true,
						},
						"size": schema.Int64Attribute{
							Description: "Size of the snapshot in GB.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *DBaaSClusterSnapshotsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dbClusterSnapshotsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	snapshots, err := r.dbaasClusters.ListAllSnapshots(ctx, data.ClusterID.ValueString(), dbSDK.ClusterSnapshotFilterOptions{})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	var snapshotModels []dbClusterSnapshotModel
	for _, snapshot := range snapshots {
		snapshotModels = append(snapshotModels, dbClusterSnapshotModel{
			ID:          types.StringValue(snapshot.ID),
			Name:        types.StringValue(snapshot.Name),
			ClusterID:   data.ClusterID,
			Description: types.StringValue(snapshot.Description),
			CreatedAt:   types.StringValue(*utils.ConvertTimeToRFC3339(&snapshot.CreatedAt)),
			Status:      types.StringValue(string(snapshot.Status)),
			Size:        types.Int64PointerValue(utils.ConvertIntPointerToInt64Pointer(&snapshot.AllocatedSize)),
		})
	}

	data.Snapshots = snapshotModels
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
