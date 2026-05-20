package containerregistry

import (
	"context"
	"errors"
	"net/http"
	"strings"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	crSDK "github.com/MagaluCloud/mgc-sdk-go/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ContainerRegistryMemberModel struct {
	ID         types.String `tfsdk:"id"`
	RegistryID types.String `tfsdk:"registry_id"`
	UserID     types.String `tfsdk:"user_id"`
	Role       types.String `tfsdk:"role"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

type ContainerRegistryMemberResource struct {
	membersService crSDK.MembersService
}

func NewContainerRegistryMemberResource() resource.Resource {
	return &ContainerRegistryMemberResource{}
}

func (r *ContainerRegistryMemberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_registry_member"
}

func (r *ContainerRegistryMemberResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.membersService = crSDK.New(&dataConfig.CoreConfig).Members()
}

func (r *ContainerRegistryMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the membership of a user in a Container Registry.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the membership entry",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"registry_id": schema.StringAttribute{
				Description: "ID of the registry the user is being added to",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Description: "ID of the container registry user to add as a member",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Description: "Role assigned to the member (e.g. project_admin, developer, guest)",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the membership was created",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the membership was last updated",
				Computed:    true,
			},
		},
	}
}

func (r *ContainerRegistryMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ContainerRegistryMemberModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.membersService.Add(ctx, data.RegistryID.ValueString(), crSDK.MemberRequest{
		UserID: data.UserID.ValueString(),
		Role:   data.Role.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	r.applyMember(&data, created)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerRegistryMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ContainerRegistryMemberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	member, err := r.membersService.Get(ctx, data.RegistryID.ValueString(), data.ID.ValueString())
	if err != nil {
		var httpErr *clientSDK.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	r.applyMember(&data, member)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerRegistryMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ContainerRegistryMemberModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ContainerRegistryMemberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Role.Equal(state.Role) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	updated, err := r.membersService.Update(ctx, state.RegistryID.ValueString(), state.ID.ValueString(), crSDK.MemberUpdateRequest{
		Role: plan.Role.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	r.applyMember(&state, updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ContainerRegistryMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ContainerRegistryMemberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.membersService.Delete(ctx, data.RegistryID.ValueString(), data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
}

func (r *ContainerRegistryMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ",")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import format", "Use `<registry_id>,<member_id>`")
		return
	}

	resp.Diagnostics.Append(
		resp.State.SetAttribute(ctx, path.Root("registry_id"), parts[0])...,
	)
	resp.Diagnostics.Append(
		resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...,
	)
}

func (r *ContainerRegistryMemberResource) applyMember(data *ContainerRegistryMemberModel, member *crSDK.MemberResponse) {
	data.ID = types.StringValue(member.ID)
	data.RegistryID = types.StringValue(member.RegistryID)
	data.UserID = types.StringValue(member.UserID)
	data.Role = types.StringValue(member.Role)
	data.CreatedAt = types.StringValue(member.CreatedAt)
	data.UpdatedAt = types.StringValue(member.UpdatedAt)
}
