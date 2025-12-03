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

type DataSourceLbaasNetworkListeners struct {
	lbNetworkListener lbSDK.NetworkListenerService
}

func NewDataSourceLbaasNetworkListeners() datasource.DataSource {
	return &DataSourceLbaasNetworkListeners{}
}

func (r *DataSourceLbaasNetworkListeners) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_network_listeners"
}

func (r *DataSourceLbaasNetworkListeners) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *DataSourceLbaasNetworkListeners) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: experimentalWarning + "List Listeners for a Network Load Balancer.",
		Attributes: map[string]schema.Attribute{
			"lb_id": schema.StringAttribute{
				Required:    true,
				Description: "The Network Load Balancer ID.",
			},
			"listeners": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of listeners.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
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
				},
			},
		},
	}
}

type listenerListState struct {
	LBID      types.String        `tfsdk:"lb_id"`
	Listeners []listenerItemModel `tfsdk:"listeners"`
}

func (r *DataSourceLbaasNetworkListeners) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var lbID types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("lb_id"), &lbID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lb, err := r.lbNetworkListener.ListAll(ctx, lbID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	state := listenerListState{
		LBID:      lbID,
		Listeners: make([]listenerItemModel, 0, len(lb)),
	}
	for _, li := range lb {
		state.Listeners = append(state.Listeners, listenerItemModel{
			ID:               types.StringValue(li.ID),
			Name:             types.StringValue(li.Name),
			Description:      types.StringPointerValue(li.Description),
			Port:             types.Int64Value(int64(li.Port)),
			Protocol:         types.StringValue(string(li.Protocol)),
			BackendID:        types.StringValue(li.BackendID),
			TLSCertificateID: types.StringPointerValue(li.TLSCertificateID),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
