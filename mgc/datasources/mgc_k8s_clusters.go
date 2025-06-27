package datasources

import (
	"context"

	sdkK8s "github.com/MagaluCloud/mgc-sdk-go/kubernetes"
	tfutil "github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type KubernetesClusterReducedModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Version     types.String `tfsdk:"version"`
	Zone        types.String `tfsdk:"zone"`
	Region      types.String `tfsdk:"region"`
	State       types.String `tfsdk:"state"`
}

type KubernetesClusterReducedModelList struct {
	Clusters []KubernetesClusterReducedModel `tfsdk:"clusters"`
}

func NewDataSourceKubernetesClusterList() datasource.DataSource {
	return &DataSourceKubernetesClusters{}
}

type DataSourceKubernetesClusters struct {
	sdkClient sdkK8s.ClusterService
}

func (r *DataSourceKubernetesClusters) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)

	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	r.sdkClient = sdkK8s.New(&dataConfig.CoreConfig).Clusters()

}

func (d *DataSourceKubernetesClusters) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_clusters"
}

func (d *DataSourceKubernetesClusters) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This data source provides a list of cluster.",
		Attributes: map[string]schema.Attribute{
			"clusters": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of available VM machine-types.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Cluster's UUID.",
							Required:    true,
						},
						"name": schema.StringAttribute{
							Description: "Kubernetes cluster name.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "A brief description of the Kubernetes cluster.",
							Computed:    true,
						},
						"version": schema.StringAttribute{
							Description: "The native Kubernetes version of the cluster.",
							Computed:    true,
						},
						"zone": schema.StringAttribute{
							Description:        "Identifier of the zone where the Kubernetes cluster is located.",
							Computed:           true,
							DeprecationMessage: "Deprecated. Field 'zone' is deprecated.",
						},
						"region": schema.StringAttribute{
							Description: "Identifier of the region where the Kubernetes cluster is located.",
							Computed:    true,
						},
						"state": schema.StringAttribute{
							Description: "Current state of the cluster or node.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *DataSourceKubernetesClusters) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data KubernetesClusterReducedModelList
	diags := resp.State.Get(ctx, &data)
	if diags.HasError() {
		resp.Diagnostics = diags
		return
	}

	cluster, err := d.sdkClient.List(ctx, sdkK8s.ListOptions{})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	for _, c := range cluster {
		data.Clusters = append(data.Clusters, KubernetesClusterReducedModel{
			ID:          types.StringValue(c.ID),
			Name:        types.StringValue(c.Name),
			Version:     types.StringPointerValue(c.Version),
			Zone:        types.StringPointerValue(c.Region),
			Description: types.StringPointerValue(c.Description),
			Region:      types.StringPointerValue(c.Region),
			State:       types.StringValue(c.Status.State),
		},
		)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
