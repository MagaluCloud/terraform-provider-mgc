package datasources

import (
	"context"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DataSourceDbSnapshot struct {
	snapshots dbSDK.InstanceService
}

type dbSnapshotModel struct {
	ID          types.String `tfsdk:"id"`
	InstanceId  types.String `tfsdk:"instance_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	CreatedAt   types.String `tfsdk:"created_at"`
	Status      types.String `tfsdk:"status"`
	Size        types.Int64  `tfsdk:"size"`
}

func NewDataSourceDbaasInstancesSnapshot() datasource.DataSource {
	return &DataSourceDbSnapshot{}
}

func (r *DataSourceDbSnapshot) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_instances_snapshot"
}

func (r *DataSourceDbSnapshot) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.snapshots = dbSDK.New(&dataConfig.CoreConfig).Instances()
}

func (r *DataSourceDbSnapshot) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get a database snapshot by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "ID of the snapshot",
				Required:    true,
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
	}
}

func (r *DataSourceDbSnapshot) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dbSnapshotModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	snapshot, err := r.snapshots.GetSnapshot(ctx, data.InstanceId.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.Name = types.StringValue(snapshot.Name)
	data.Description = types.StringValue(snapshot.Description)
	data.CreatedAt = types.StringValue(snapshot.CreatedAt)
	data.Status = types.StringValue(string(snapshot.Status))
	data.Size = types.Int64PointerValue(tfutil.ConvertIntPointerToInt64Pointer(&snapshot.AllocatedSize))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
