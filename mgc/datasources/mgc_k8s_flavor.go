package datasources

import (
	"context"

	mgcSdk "github.com/MagaluCloud/magalu/mgc/lib"
	sdkNodepool "github.com/MagaluCloud/magalu/mgc/lib/products/kubernetes/flavor"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/client"
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
	sdkClient *mgcSdk.Client
	nodepool  sdkNodepool.Service
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

	var err error
	var errDetail error
	r.sdkClient, err, errDetail = client.NewSDKClient(req, resp)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			errDetail.Error(),
		)
		return
	}

	r.nodepool = sdkNodepool.NewService(ctx, r.sdkClient)
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
	result, err := r.nodepool.ListContext(ctx, tfutil.GetConfigsFromTags(r.sdkClient.Sdk().Config().Get, sdkNodepool.ListConfigs{}))

	if err != nil || result.Results == nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	f := result.Results[0]
	output := &ListResultResultsItem{
		Controlplane: resourceListResultResultsItemBastionItem(f.Controlplane),
		Nodepool:     resourceListResultResultsItemBastionItem(f.Nodepool),
	}

	resp.Diagnostics = resp.State.Set(ctx, &output)
}

func resourceListResultResultsItemBastionItem(items []sdkNodepool.ListResultResultsItemControlplaneItem) []ListResultResultsItemBastionItem {
	var result []ListResultResultsItemBastionItem
	for _, item := range items {
		result = append(result, ListResultResultsItemBastionItem{
			Id:   types.StringValue(item.Id),
			Name: types.StringValue(item.Name),
			Ram:  types.Int64Value(int64(item.Ram)),
			Size: types.Int64Value(int64(item.Size)),
			Vcpu: types.Int64Value(int64(item.Vcpu)),
		})
	}
	return result
}
