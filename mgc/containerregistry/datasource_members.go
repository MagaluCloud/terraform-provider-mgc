package containerregistry

import (
	"context"

	crSDK "github.com/MagaluCloud/mgc-sdk-go/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceCRMembers{}

type DataSourceCRMembers struct {
	membersService crSDK.MembersService
}

type crMember struct {
	ID         types.String `tfsdk:"id"`
	RegistryID types.String `tfsdk:"registry_id"`
	UserID     types.String `tfsdk:"user_id"`
	Role       types.String `tfsdk:"role"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

type crMembersList struct {
	RegistryID types.String `tfsdk:"registry_id"`
	Members    []crMember   `tfsdk:"members"`
}

func NewDataSourceCRMembers() datasource.DataSource {
	return &DataSourceCRMembers{}
}

func (r *DataSourceCRMembers) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_registry_members"
}

func (r *DataSourceCRMembers) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *DataSourceCRMembers) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all members of a Container Registry.",
		Attributes: map[string]schema.Attribute{
			"registry_id": schema.StringAttribute{
				Description: "ID of the registry whose members will be listed",
				Required:    true,
			},
			"members": schema.ListNestedAttribute{
				Description: "List of members of the registry",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Unique identifier of the membership entry",
							Computed:    true,
						},
						"registry_id": schema.StringAttribute{
							Description: "ID of the registry",
							Computed:    true,
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
				},
			},
		},
	}
}

func (r *DataSourceCRMembers) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data crMembersList
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	members, err := r.membersService.ListAll(ctx, data.RegistryID.ValueString(), crSDK.MemberFilterOptions{})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	for _, m := range members {
		data.Members = append(data.Members, crMember{
			ID:         types.StringValue(m.ID),
			RegistryID: types.StringValue(m.RegistryID),
			UserID:     types.StringValue(m.UserID),
			Role:       types.StringValue(m.Role),
			CreatedAt:  types.StringValue(m.CreatedAt),
			UpdatedAt:  types.StringValue(m.UpdatedAt),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
