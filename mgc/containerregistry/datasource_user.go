package containerregistry

import (
	"context"

	crSDK "github.com/MagaluCloud/mgc-sdk-go/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceCRUser{}

type DataSourceCRUser struct {
	usersService crSDK.UsersService
}

type crUserModel struct {
	ID        types.String `tfsdk:"id"`
	Username  types.String `tfsdk:"username"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func NewDataSourceCRUser() datasource.DataSource {
	return &DataSourceCRUser{}
}

func (r *DataSourceCRUser) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_registry_user"
}

func (r *DataSourceCRUser) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	r.usersService = crSDK.New(&dataConfig.CoreConfig).Users()
}

func (r *DataSourceCRUser) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns the authenticated Container Registry user, useful for retrieving the user ID to use in other resources.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the container registry user",
				Computed:    true,
			},
			"username": schema.StringAttribute{
				Description: "Username assigned to the container registry user",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the user was created",
				Computed:    true,
			},
		},
	}
}

func (r *DataSourceCRUser) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data crUserModel

	user, err := r.usersService.Get(ctx)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.ID = types.StringValue(user.ID)
	data.Username = types.StringValue(user.Username)
	data.CreatedAt = types.StringValue(user.CreatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
