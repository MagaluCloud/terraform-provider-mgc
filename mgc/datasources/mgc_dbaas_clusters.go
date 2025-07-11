package datasources

import (
	"context"
	"fmt"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DBaaSClustersDataSource struct {
	dbaasClusters dbSDK.ClusterService
}

type dbaasClustersDataSourceModel struct {
	Clusters         []dbaasClusterDataModel `tfsdk:"clusters"`
	Status           types.String            `tfsdk:"status_filter"`
	EngineID         types.String            `tfsdk:"engine_id_filter"`
	ParameterGroupID types.String            `tfsdk:"parameter_group_id_filter"`
}

func NewDBaaSClustersDataSource() datasource.DataSource {
	return &DBaaSClustersDataSource{}
}

func (ds *DBaaSClustersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_clusters"
}

func (ds *DBaaSClustersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Provider data has unexpected type")
		return
	}
	ds.dbaasClusters = dbSDK.New(&dataConfig.CoreConfig).Clusters()
}

func (ds *DBaaSClustersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a list of DBaaS clusters, optionally filtered.",
		Attributes: map[string]schema.Attribute{
			"clusters": schema.ListNestedAttribute{
				Description: "A list of DBaaS clusters.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: dbaasClusterAttributes(),
				},
			},
			"status_filter": schema.StringAttribute{
				Description: "Filter clusters by status.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(dbSDK.ClusterStatusActive), string(dbSDK.ClusterStatusError), string(dbSDK.ClusterStatusPending),
						string(dbSDK.ClusterStatusCreating), string(dbSDK.ClusterStatusDeleting), string(dbSDK.ClusterStatusDeleted),
						string(dbSDK.ClusterStatusErrorDeleting), string(dbSDK.ClusterStatusStarting), string(dbSDK.ClusterStatusStopping),
						string(dbSDK.ClusterStatusStopped), string(dbSDK.ClusterStatusBackingUp),
					),
				},
			},
			"engine_id_filter": schema.StringAttribute{
				Description: "Filter clusters by engine ID.",
				Optional:    true,
			},
			"parameter_group_id_filter": schema.StringAttribute{
				Description: "Filter clusters by parameter group ID.",
				Optional:    true,
			},
		},
	}
}

func (ds *DBaaSClustersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config dbaasClustersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	listOpts := dbSDK.ListClustersOptions{}
	if !config.Status.IsNull() && !config.Status.IsUnknown() {
		sdkStatus := dbSDK.ClusterStatus(config.Status.ValueString())
		listOpts.Status = &sdkStatus
	}
	listOpts.EngineID = config.EngineID.ValueStringPointer()
	listOpts.ParameterGroupID = config.ParameterGroupID.ValueStringPointer()

	var allSDKClusters []dbSDK.ClusterDetailResponse
	zeroOffset := 0
	limiteTop := 25
	listOpts.Offset = &zeroOffset
	listOpts.Limit = &limiteTop

	for {
		sdkClustersPage, err := ds.dbaasClusters.List(ctx, listOpts)
		if err != nil {
			resp.Diagnostics.AddError("Failed to list clusters", fmt.Sprintf("Error: %s", err.Error()))
			return
		}
		allSDKClusters = append(allSDKClusters, sdkClustersPage...)

		if len(sdkClustersPage) < *listOpts.Limit {
			break
		}
		*listOpts.Offset += *listOpts.Limit
	}

	var resultClusters []dbaasClusterDataModel

	for _, sdkCluster := range allSDKClusters {
		resultClusters = append(resultClusters, convertSDKClusterToDataModel(sdkCluster))
	}

	config.Clusters = resultClusters
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
