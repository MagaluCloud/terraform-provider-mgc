package datasources

import (
	"context"

	sdkK8s "github.com/MagaluCloud/mgc-sdk-go/kubernetes"
	tfutil "github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.in/yaml.v3"
)

var _ datasource.DataSource = &DataSourceKubernetesClusterKubeConfig{}

func NewDataSourceKubernetesClusterKubeConfig() datasource.DataSource {
	return &DataSourceKubernetesClusterKubeConfig{}
}

type DataSourceKubernetesClusterKubeConfig struct {
	sdkClient sdkK8s.ClusterService
}

type DataSourceKubernetesClusterKubeConfigModel struct {
	ClusterID types.String `tfsdk:"cluster_id"`
	RawConfig types.String `tfsdk:"kubeconfig"`
}

func (d *DataSourceKubernetesClusterKubeConfig) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_cluster_kubeconfig"
}

func (d *DataSourceKubernetesClusterKubeConfig) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				Required:    true,
				Description: "The unique identifier of the Kubernetes cluster.",
			},
			"kubeconfig": schema.StringAttribute{
				Computed:    true,
				Description: "The full contents of the Kubernetes cluster's kubeconfig yaml file.",
			},
		},
	}
	resp.Schema.Description = "Get the kubeconfig of a Kubernetes cluster by cluster_id."
}

func (d *DataSourceKubernetesClusterKubeConfig) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataSourceKubernetesClusterKubeConfigModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	sdkOuput, err := d.sdkClient.GetKubeConfig(ctx, data.ClusterID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	rawConfig, err := yaml.Marshal(sdkOuput)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.RawConfig = types.StringValue(string(rawConfig))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DataSourceKubernetesClusterKubeConfig) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
