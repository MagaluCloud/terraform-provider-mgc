package lbaas

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	lbSDK "github.com/MagaluCloud/mgc-sdk-go/lbaas"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

type DataSourceLbaasNetworkListener struct {
	lbNetworkListener lbSDK.NetworkListenerService
}

func NewDataSourceLbaasNetworkListener() datasource.DataSource {
	return &DataSourceLbaasNetworkListener{}
}

func (r *DataSourceLbaasNetworkListener) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_network_listener"
}

func (r *DataSourceLbaasNetworkListener) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}
	client := lbSDK.New(&dataConfig.CoreConfig)
	r.lbNetworkListener = client.NetworkListeners()
}

func (r *DataSourceLbaasNetworkListener) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: experimentalWarning + "Get a specific Network Load Balancer Listener by Load Balancer ID and Listener ID.",
		Attributes: map[string]schema.Attribute{
			"lb_id": schema.StringAttribute{
				Required:    true,
				Description: "The Network Load Balancer ID.",
			},
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The Listener ID.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the listener.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "The description of the listener.",
			},
			"port": schema.Int64Attribute{
				Computed:    true,
				Description: "The port for the listener.",
			},
			"protocol": schema.StringAttribute{
				Computed:    true,
				Description: "The protocol for the listener. Example: 'tcp', 'tls'.",
			},
			"backend_id": schema.StringAttribute{
				Computed:    true,
				Description: "The associated backend ID.",
			},
			"tls_certificate_id": schema.StringAttribute{
				Computed:    true,
				Description: "The associated TLS certificate ID, if any.",
			},
		},
	}
}

type listenerItemModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Port             types.Int64  `tfsdk:"port"`
	Protocol         types.String `tfsdk:"protocol"`
	BackendID        types.String `tfsdk:"backend_id"`
	TLSCertificateID types.String `tfsdk:"tls_certificate_id"`
}

type networkListenerItemModel struct {
	LBID types.String `tfsdk:"lb_id"`
	listenerItemModel
}

func (r *DataSourceLbaasNetworkListener) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var lbID, listenerID types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("lb_id"), &lbID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("id"), &listenerID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lb, err := r.lbNetworkListener.Get(ctx, lbID.ValueString(), listenerID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	item := networkListenerItemModel{
		LBID: lbID,
		listenerItemModel: listenerItemModel{
			ID:               types.StringValue(lb.ID),
			Name:             types.StringValue(lb.Name),
			Description:      types.StringPointerValue(lb.Description),
			Port:             types.Int64Value(int64(lb.Port)),
			Protocol:         types.StringValue(string(lb.Protocol)),
			BackendID:        types.StringValue(lb.BackendID),
			TLSCertificateID: types.StringPointerValue(lb.TLSCertificateID),
		},
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &item)...)
}
