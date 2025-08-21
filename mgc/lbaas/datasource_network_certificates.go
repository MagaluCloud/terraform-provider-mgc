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

type DataSourceLbaasNetworkCertificates struct {
	lbNetworkTLS lbSDK.NetworkCertificateService
}

func NewDataSourceLbaasNetworkCertificates() datasource.DataSource {
	return &DataSourceLbaasNetworkCertificates{}
}

func (r *DataSourceLbaasNetworkCertificates) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_network_certificates"
}

func (r *DataSourceLbaasNetworkCertificates) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	client := lbSDK.New(&dataConfig.CoreConfig)
	r.lbNetworkTLS = client.NetworkCertificates()
}

func (r *DataSourceLbaasNetworkCertificates) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List TLS Certificates for a Network Load Balancer.",
		Attributes: map[string]schema.Attribute{
			"lb_id": schema.StringAttribute{
				Required:    true,
				Description: "The Network Load Balancer ID.",
			},
			"tls_certificates": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of TLS certificates.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The TLS Certificate ID.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the TLS certificate.",
						},
						"description": schema.StringAttribute{
							Computed:    true,
							Description: "The description of the TLS certificate.",
						},
						"expiration_date": schema.StringAttribute{
							Computed:    true,
							Description: "The expiration date of the TLS certificate.",
						},
					},
				},
			},
		},
	}
}

type certificateListState struct {
	LBID            types.String           `tfsdk:"lb_id"`
	TLSCertificates []certificateItemModel `tfsdk:"tls_certificates"`
}

func (r *DataSourceLbaasNetworkCertificates) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var lbID types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("lb_id"), &lbID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lb, err := r.lbNetworkTLS.List(ctx, lbID.ValueString(), lbSDK.ListNetworkLoadBalancerRequest{})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	state := certificateListState{
		LBID:            lbID,
		TLSCertificates: make([]certificateItemModel, 0, len(lb)),
	}
	for _, c := range lb {
		item := certificateItemModel{
			ID:          types.StringValue(c.ID),
			Name:        types.StringValue(c.Name),
			Description: types.StringPointerValue(c.Description),
		}
		if c.ExpirationDate != nil {
			item.ExpirationDate = types.StringValue(c.ExpirationDate.String())
		} else {
			item.ExpirationDate = types.StringNull()
		}
		state.TLSCertificates = append(state.TLSCertificates, item)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
