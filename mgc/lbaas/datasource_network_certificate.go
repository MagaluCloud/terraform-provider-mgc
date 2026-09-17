package lbaas

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	lbSDK "github.com/MagaluCloud/mgc-sdk-go/lbaas"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/project"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

type DataSourceLbaasNetworkCertificate struct {
	lbNetworkTLS lbSDK.NetworkCertificateService
}

func NewDataSourceLbaasNetworkCertificate() datasource.DataSource {
	return &DataSourceLbaasNetworkCertificate{}
}

func (r *DataSourceLbaasNetworkCertificate) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_network_certificate"
}

func (r *DataSourceLbaasNetworkCertificate) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	client := lbSDK.New(dataConfig.CoreFor(utils.ServiceLbaas))
	r.lbNetworkTLS = client.NetworkCertificates()
}

func (r *DataSourceLbaasNetworkCertificate) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: experimentalWarning + "Get a specific Network Load Balancer TLS Certificate by Load Balancer ID and Certificate ID.",
		Attributes: map[string]schema.Attribute{
			"project_id": project.DataSourceAttribute(),
			"lb_id": schema.StringAttribute{
				Required:    true,
				Description: "The Network Load Balancer ID.",
			},
			"id": schema.StringAttribute{
				Required:    true,
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
	}
}

type certificateItemModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
}

type certificateItemState struct {
	LBID types.String `tfsdk:"lb_id"`
	certificateItemModel
	ProjectID types.String `tfsdk:"project_id"`
}

func (r *DataSourceLbaasNetworkCertificate) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var lbID, certID, projectID types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("lb_id"), &lbID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("id"), &certID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("project_id"), &projectID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lb, err := r.lbNetworkTLS.Get(ctx, project.CallOptions(projectID), lbID.ValueString(), certID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(project.ParseSDKError(err, projectID))
		return
	}
	item := certificateItemState{
		ProjectID: projectID,
		LBID:      lbID,
		certificateItemModel: certificateItemModel{
			ID:          types.StringValue(lb.ID),
			Name:        types.StringValue(lb.Name),
			Description: types.StringPointerValue(lb.Description),
		},
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &item)...)
}
