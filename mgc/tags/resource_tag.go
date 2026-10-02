package tags

//go:generate go run github.com/vektra/mockery/v2@v2.53.6 --name=TagService --srcpkg=github.com/MagaluCloud/mgc-sdk-go/tag --output=../internal/mocks --outpkg=mocks

import (
	"context"
	"fmt"
	"regexp"
	"time"

	tagSDK "github.com/MagaluCloud/mgc-sdk-go/tag"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	tagNameRule = regexp.MustCompile(`^[\p{L}\p{N}_ \-\[\]\(\)\.\:]+$`)
	colorRule   = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)
)

type tagResourceModel struct {
	ID          types.String               `tfsdk:"id"`
	Name        types.String               `tfsdk:"name"`
	Description types.String               `tfsdk:"description"`
	Color       caseInsensitiveStringValue `tfsdk:"color"`
	Kinds       types.Set                  `tfsdk:"kinds"`
	CreatedAt   types.String               `tfsdk:"created_at"`
	UpdatedAt   types.String               `tfsdk:"updated_at"`
}

type tagResource struct {
	tags tagSDK.TagService
}

func NewTagResource() resource.Resource {
	return &tagResource{}
}

func (r *tagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (r *tagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.tags = newTagClient(dataConfig).Tags()
}

func (r *tagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A tag that can be attached to cloud resources, to organize and report on them. " +
			"Tags are global: they are not bound to a region. " +
			"Attaching a tag to a resource also requires a value (`mgc_tag_value`) and an attachment (`mgc_tag_attachment`).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The tag name, which is also its identifier.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					tagIDFromName(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the tag, unique within the tenant. Names are case sensitive: `finops` and `FinOps` are different tags. " +
					"Changing this renames the tag in place: its values and the resources that carry it follow the new name.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
					stringvalidator.RegexMatches(tagNameRule, "must contain only letters, digits, spaces or the characters _-[]().:"),
				},
			},
			"description": schema.StringAttribute{
				Description: "A brief description of the tag.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(500),
				},
			},
			"color": schema.StringAttribute{
				Description: "Color of the tag, as a 6-digit hex RGB code without the `#` prefix. Case insensitive: the API stores it lowercased. " +
					"When omitted, the API assigns a color, which is why this value is kept in state. " +
					"The API has no way to clear a color, so removing this from the configuration keeps the current one.",
				Optional:   true,
				Computed:   true,
				CustomType: caseInsensitiveStringType{},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(colorRule, "must be a 6-character hexadecimal RGB code without the '#' prefix, e.g. f54927"),
				},
			},
			"kinds": schema.SetAttribute{
				Description: "Kinds that describe what the tag is for, such as `finops`.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(kindValidator{}),
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Creation date of the tag.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Last update date of the tag, null while it was never updated.",
				Computed:    true,
			},
		},
	}
}

func (r *tagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.tags.Create(ctx, buildCreateTagRequest(plan))
	if err != nil {
		if isConflict(err) {
			name := plan.Name.ValueString()
			resp.Diagnostics.AddError(
				"Tag already exists",
				fmt.Sprintf("A tag named %q already exists in this tenant. Import it instead: terraform import <resource address> %s", name, name),
			)
			return
		}
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	plan = flattenTag(plan, *created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *tagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data tagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tag, err := r.tags.Get(ctx, data.Name.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddWarning(
				"MGC Resource not found the tag during refresh",
				"The tag has been automatically removed from the state and will be recreated",
			)
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data = flattenTag(data, *tag)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *tagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state tagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.tags.Update(ctx, state.Name.ValueString(), buildUpdateTagRequest(state, plan))
	if err != nil {
		if isConflict(err) {
			resp.Diagnostics.AddError(
				"Tag already exists",
				fmt.Sprintf("Cannot rename the tag: a tag named %q already exists in this tenant.", plan.Name.ValueString()),
			)
			return
		}
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	plan = flattenTag(plan, *updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *tagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data tagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.tags.Delete(ctx, data.Name.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
	}
}

func (r *tagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import ID", "The tag name must be provided")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}

func flattenTag(tfData tagResourceModel, tag tagSDK.Tag) tagResourceModel {
	tfData.ID = types.StringValue(tag.Name)
	tfData.Name = utils.FlattenStringValue(tfData.Name, &tag.Name)
	tfData.Description = utils.FlattenStringValue(tfData.Description, tag.Description)
	tfData.Color = flattenColor(tag.Color)
	tfData.Kinds = utils.FlattenTypeSetStringArray(tfData.Kinds, kindsToStrings(tag.Kinds))
	tfData.CreatedAt = types.StringPointerValue(utils.ConvertTimeToRFC3339((*time.Time)(&tag.CreatedAt)))
	tfData.UpdatedAt = types.StringPointerValue(utils.ConvertTimeToRFC3339((*time.Time)(tag.UpdatedAt)))

	return tfData
}

func buildCreateTagRequest(plan tagResourceModel) tagSDK.CreateTagRequest {
	return tagSDK.CreateTagRequest{
		Name:        plan.Name.ValueString(),
		Description: utils.KnownStringPointer(plan.Description),
		Color:       utils.KnownStringPointer(plan.Color.StringValue),
		Kinds:       toKinds(utils.ConvertTypeSetToStringArray(plan.Kinds)),
	}
}

func buildUpdateTagRequest(state, plan tagResourceModel) tagSDK.UpdateTagRequest {
	description := ""
	if planned := utils.KnownStringPointer(plan.Description); planned != nil {
		description = *planned
	}

	request := tagSDK.UpdateTagRequest{
		Description: &description,
	}

	if !plan.Name.Equal(state.Name) {
		request.Name = utils.KnownStringPointer(plan.Name)
	}

	if !plan.Color.Equal(state.Color) {
		request.Color = utils.KnownStringPointer(plan.Color.StringValue)
	}

	if !plan.Kinds.Equal(state.Kinds) {
		kinds := toKinds(utils.ConvertTypeSetToStringArray(plan.Kinds))
		if kinds == nil {
			kinds = []tagSDK.TagKind{}
		}
		request.Kinds = &kinds
	}

	return request
}

func flattenColor(color *string) caseInsensitiveStringValue {
	if color == nil {
		return caseInsensitiveStringValue{StringValue: types.StringNull()}
	}
	return newCaseInsensitiveString(*color)
}

func toKinds(kinds *[]string) []tagSDK.TagKind {
	if kinds == nil {
		return nil
	}

	converted := make([]tagSDK.TagKind, 0, len(*kinds))
	for _, kind := range *kinds {
		converted = append(converted, tagSDK.TagKind(kind))
	}
	return converted
}

func kindsToStrings(kinds []tagSDK.TagKind) *[]string {
	if kinds == nil {
		return nil
	}

	converted := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		converted = append(converted, string(kind))
	}
	return &converted
}
