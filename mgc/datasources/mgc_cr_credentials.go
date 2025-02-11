package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	crSDK "github.com/MagaluCloud/mgc-sdk-go/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceCRCredentials{}

type DataSourceCRCredentials struct {
	crCredentials crSDK.CredentialsService
}

func NewDataSourceCRCredentials() datasource.DataSource {
	return &DataSourceCRCredentials{}
}

func (r *DataSourceCRCredentials) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_credentials"
}

type crCredentials struct {
	Email    types.String `tfsdk:"email"`
	Password types.String `tfsdk:"password"`
	Username types.String `tfsdk:"username"`
}

func (r *DataSourceCRCredentials) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	r.crCredentials = crSDK.New(&dataConfig.CoreConfig).Credentials()
}

func (r *DataSourceCRCredentials) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Credentials for Container Registry authentication",
		Attributes: map[string]schema.Attribute{
			"email": schema.StringAttribute{
				Description: "Email address for the credentials",
				Computed:    true,
				Sensitive:   false,
			},
			"password": schema.StringAttribute{
				Description: "Password for authentication",
				Computed:    true,
				Sensitive:   true,
			},
			"username": schema.StringAttribute{
				Description: "Username for authentication",
				Computed:    true,
				Sensitive:   false,
			},
		},
	}
}

func (r *DataSourceCRCredentials) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data crCredentials

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sdkOutput, err := r.crCredentials.Get(ctx)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.Email = types.StringValue(sdkOutput.Email)
	data.Password = types.StringValue(sdkOutput.Password)
	data.Username = types.StringValue(sdkOutput.Username)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
