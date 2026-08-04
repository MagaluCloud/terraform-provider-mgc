package database

import (
	"context"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DBaaSClusterSnapshotDataSource struct {
	dbaasClusters dbSDK.ClusterService
}

type dbClusterSnapshotModel struct {
	ID          types.String `tfsdk:"id"`
	ClusterID   types.String `tfsdk:"cluster_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	CreatedAt   types.String `tfsdk:"created_at"`
	Status      types.String `tfsdk:"status"`
	Size        types.Int64  `tfsdk:"size"`
}

func NewDBaaSClusterSnapshotDataSource() datasource.DataSource {
	return &DBaaSClusterSnapshotDataSource{}
}

func (r *DBaaSClusterSnapshotDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_clusters_snapshot"
}

func (r *DBaaSClusterSnapshotDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *DBaaSClusterSnapshotDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get a database cluster snapshot by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "ID of the snapshot",
				Required:    true,
			},
			"cluster_id": schema.StringAttribute{
				Description: "ID of the cluster",
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
	}
}

func (r *DBaaSClusterSnapshotDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dbClusterSnapshotModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	snapshot, err := r.dbaasClusters.GetSnapshot(ctx, data.ClusterID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.Name = types.StringValue(snapshot.Name)
	data.Description = types.StringValue(snapshot.Description)
	data.CreatedAt = types.StringValue(*utils.ConvertTimeToRFC3339(&snapshot.CreatedAt))
	data.Status = types.StringValue(string(snapshot.Status))
	data.Size = types.Int64PointerValue(utils.ConvertIntPointerToInt64Pointer(&snapshot.AllocatedSize))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
