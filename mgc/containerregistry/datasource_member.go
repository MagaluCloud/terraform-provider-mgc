package containerregistry

import (
	"context"

	crSDK "github.com/MagaluCloud/mgc-sdk-go/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceCRMember{}

type DataSourceCRMember struct {
	membersService crSDK.MembersService
}

type crMemberSingle struct {
	RegistryID types.String `tfsdk:"registry_id"`
	MemberID   types.String `tfsdk:"member_id"`
	UserID     types.String `tfsdk:"user_id"`
	Role       types.String `tfsdk:"role"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

func NewDataSourceCRMember() datasource.DataSource {
	return &DataSourceCRMember{}
}

func (r *DataSourceCRMember) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_registry_member"
}

func (r *DataSourceCRMember) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	r.membersService = crSDK.New(&dataConfig.CoreConfig).Members()
}

func (r *DataSourceCRMember) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a single member of a Container Registry.",
		Attributes: map[string]schema.Attribute{
			"registry_id": schema.StringAttribute{
				Description: "ID of the registry",
				Required:    true,
			},
			"member_id": schema.StringAttribute{
				Description: "ID of the membership entry",
				Required:    true,
			},
			"user_id": schema.StringAttribute{
				Description: "ID of the user",
				Computed:    true,
			},
			"role": schema.StringAttribute{
				Description: "Role assigned to the member",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the membership was created",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the membership was last updated",
				Computed:    true,
			},
		},
	}
}

func (r *DataSourceCRMember) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data crMemberSingle
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	member, err := r.membersService.Get(ctx, data.RegistryID.ValueString(), data.MemberID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.RegistryID = types.StringValue(member.RegistryID)
	data.MemberID = types.StringValue(member.ID)
	data.UserID = types.StringValue(member.UserID)
	data.Role = types.StringValue(member.Role)
	data.CreatedAt = types.StringValue(member.CreatedAt)
	data.UpdatedAt = types.StringValue(member.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
