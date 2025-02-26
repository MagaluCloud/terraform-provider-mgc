package datasources

import (
	"context"

	sdkK8s "github.com/MagaluCloud/mgc-sdk-go/kubernetes"
	tfutil "github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ListResultResultsItem struct {
	Controlplane []ListResultResultsItemBastionItem `tfsdk:"controlplane"`
	Nodepool     []ListResultResultsItemBastionItem `tfsdk:"nodepool"`
}

type ListResultResultsItemBastionItem struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Ram  types.Int64  `tfsdk:"ram"`
	Size types.Int64  `tfsdk:"size"`
	Vcpu types.Int64  `tfsdk:"vcpu"`
}

type DataSourceKubernetesFlavor struct {
	sdkClient sdkK8s.FlavorService
}

func NewDataSourceKubernetesFlavor() datasource.DataSource {
	return &DataSourceKubernetesFlavor{}
}

func (r *DataSourceKubernetesFlavor) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_flavor"
}

func (r *DataSourceKubernetesFlavor) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)

	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	r.sdkClient = sdkK8s.New(&dataConfig.CoreConfig).Flavors()
}

func (r *DataSourceKubernetesFlavor) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Available flavors for Kubernetes clusters.",
		Attributes: map[string]schema.Attribute{
			"controlplane": schema.ListNestedAttribute{
				Description:  "Control plane configuration.",
				Computed:     true,
				NestedObject: resourceListResultResultsItemBastionItemSchema(),
			},
			"nodepool": schema.ListNestedAttribute{
				Description:  "Node pool configuration.",
				Computed:     true,
				NestedObject: resourceListResultResultsItemBastionItemSchema(),
			},
		},
	}
}

func resourceListResultResultsItemBastionItemSchema() schema.NestedAttributeObject {
	return schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "ID of the flavor.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the flavor.",
				Computed:    true,
			},
			"ram": schema.Int64Attribute{
				Description: "Amount of RAM in MB.",
				Computed:    true,
			},
			"size": schema.Int64Attribute{
				Description: "Size of the flavor.",
				Computed:    true,
			},
			"vcpu": schema.Int64Attribute{
				Description: "Number of virtual CPUs.",
				Computed:    true,
			},
		},
	}
}

func (r *DataSourceKubernetesFlavor) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := r.sdkClient.List(ctx, sdkK8s.ListOptions{ /*todo*/ })
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	controlplane := resourceListResultResultsItemBastionItem(result.ControlPlane)
	nodepool := resourceListResultResultsItemBastionItem(result.NodePool)

	output := &ListResultResultsItem{
		Controlplane: controlplane,
		Nodepool:     nodepool,
	}

	resp.Diagnostics = resp.State.Set(ctx, &output)
}

func resourceListResultResultsItemBastionItem(items []sdkK8s.Flavor) []ListResultResultsItemBastionItem {
	var result []ListResultResultsItemBastionItem
	for _, item := range items {
		result = append(result, ListResultResultsItemBastionItem{
			Id:   types.StringValue(item.ID),
			Name: types.StringValue(item.Name),
			Ram:  types.Int64Value(int64(item.RAM)),
			Size: types.Int64Value(int64(item.Size)),
			Vcpu: types.Int64Value(int64(item.VCPU)),
		})
	}
	return result
}
