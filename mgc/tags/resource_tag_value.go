package tags

//go:generate go run github.com/vektra/mockery/v2@v2.53.6 --name=TagValueService --srcpkg=github.com/MagaluCloud/mgc-sdk-go/tag --output=../internal/mocks --outpkg=mocks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tagSDK "github.com/MagaluCloud/mgc-sdk-go/tag"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const idSeparator = ","

type tagValueResourceModel struct {
	ID          types.String `tfsdk:"id"`
	TagName     types.String `tfsdk:"tag_name"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

type tagValueResource struct {
	values tagSDK.TagValueService
}

func NewTagValueResource() resource.Resource {
	return &tagValueResource{}
}

func (r *tagValueResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_value"
}

func (r *tagValueResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.values = newTagClient(dataConfig).Values()
}

func (r *tagValueResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Value of a tag, such as `production` for the tag `environment`. " +
			"A resource carries one value per tag. " +
			"A value attached to a resource cannot be deleted: detach it first, in a separate apply.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier in the form `<tag_name>,<name>`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					tagValueIDFromNames(),
				},
			},
			"tag_name": schema.StringAttribute{
				Description: "Name of the tag that owns the value. Reference `mgc_tag.<name>.name` to follow tag renames. " +
					"Moving the value to another tag is not supported.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
					stringvalidator.RegexMatches(tagNameRule, "must contain only letters, digits, spaces or the characters _-[]().:"),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the value. Unique in the tag. " +
					"Case sensitive. 1 to 255 characters: letters, digits, spaces and `_-[]().:`. " +
					"Changing it renames the value in place and keeps its attachments.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
					stringvalidator.RegexMatches(tagNameRule, "must contain only letters, digits, spaces or the characters _-[]().:"),
				},
			},
			"description": schema.StringAttribute{
				Description: "Description of the value. Up to 500 characters.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(500),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Creation date of the value.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Last update date of the value. Null if never updated.",
				Computed:    true,
			},
		},
	}
}

func (r *tagValueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tagValueResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.values.Create(ctx, plan.TagName.ValueString(), tagSDK.CreateTagValueRequest{
		Name:        plan.Name.ValueString(),
		Description: utils.KnownStringPointer(plan.Description),
	})
	if err != nil {
		if isConflict(err) {
			resp.Diagnostics.AddError(
				"Tag value already exists",
				"The tag already has a value with this name. Import it instead: terraform import <resource address> "+
					tagValueID(plan.TagName.ValueString(), plan.Name.ValueString()),
			)
			return
		}
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	plan = flattenTagValue(plan, *created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *tagValueResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data tagValueResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	value, err := r.values.Get(ctx, data.TagName.ValueString(), data.Name.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddWarning(
				"MGC Resource not found the tag value during refresh",
				"The tag value has been automatically removed from the state and will be recreated",
			)
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data = flattenTagValue(data, *value)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *tagValueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state tagValueResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.TagName.Equal(state.TagName) {
		// A renamed tag takes its values along, so a value still under the old
		// name means the config moves it to another tag.
		_, err := r.values.Get(ctx, state.TagName.ValueString(), state.Name.ValueString())
		if err == nil {
			resp.Diagnostics.AddError(
				"Moving a tag value is not supported",
				fmt.Sprintf("The value %q still exists in the tag %q, so it cannot follow tag_name to %q: move are not supported, "+
					"and deleting a value attached to a resource fails. Declare a new mgc_tag_value in %q, point the attachments to it, "+
					"and remove this one in a later apply.",
					state.Name.ValueString(), state.TagName.ValueString(), plan.TagName.ValueString(), plan.TagName.ValueString()),
			)
			return
		}
		if !isNotFound(err) {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
	}

	updated, err := r.values.Update(
		ctx,
		plan.TagName.ValueString(),
		state.Name.ValueString(),
		buildUpdateTagValueRequest(state, plan),
	)
	if err != nil {
		if isConflict(err) {
			resp.Diagnostics.AddError(
				"Tag value already exists",
				"Cannot rename the value: the tag already has a value named "+plan.Name.ValueString()+".",
			)
			return
		}
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	plan = flattenTagValue(plan, *updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *tagValueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data tagValueResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.values.Delete(ctx, data.TagName.ValueString(), data.Name.ValueString())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
	}
}

func (r *tagValueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tagName, valueName, err := parseTagValueID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import format", "Format should be: tag_name,value_name")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tag_name"), tagName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), valueName)...)
}

func flattenTagValue(tfData tagValueResourceModel, value tagSDK.TagValue) tagValueResourceModel {
	tfData.ID = types.StringValue(tagValueID(tfData.TagName.ValueString(), value.Name))
	tfData.Name = utils.FlattenStringValue(tfData.Name, &value.Name)
	tfData.Description = utils.FlattenStringValue(tfData.Description, value.Description)
	tfData.CreatedAt = types.StringPointerValue(utils.ConvertTimeToRFC3339((*time.Time)(&value.CreatedAt)))
	tfData.UpdatedAt = types.StringPointerValue(utils.ConvertTimeToRFC3339((*time.Time)(value.UpdatedAt)))

	return tfData
}

func buildUpdateTagValueRequest(state, plan tagValueResourceModel) tagSDK.UpdateTagValueRequest {
	description := ""
	if planned := utils.KnownStringPointer(plan.Description); planned != nil {
		description = *planned
	}

	request := tagSDK.UpdateTagValueRequest{Description: &description}
	if !plan.Name.Equal(state.Name) {
		request.Name = utils.KnownStringPointer(plan.Name)
	}

	return request
}

func tagValueID(tagName, valueName string) string {
	return tagName + idSeparator + valueName
}

func parseTagValueID(id string) (tagName, valueName string, err error) {
	parts := strings.Split(id, idSeparator)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", errors.New("id must be in the form <tag_name>,<value_name>")
	}

	return parts[0], parts[1], nil
}
