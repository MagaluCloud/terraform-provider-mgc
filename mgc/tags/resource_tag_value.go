package tags

//go:generate go run github.com/vektra/mockery/v2@v2.53.6 --name=TagValueService --srcpkg=github.com/MagaluCloud/mgc-sdk-go/tag --output=../internal/mocks --outpkg=mocks

import (
	"context"
	"errors"
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

// idSeparator is safe because the name pattern of the API accepts spaces and
// brackets but never a comma.
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
		Description: "A value of a tag. A resource carries a single value per tag, and both the tag and the value " +
			"have to exist before they can be attached to a resource with `mgc_tag_attachment`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the value, in the form `<tag_name>,<name>`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tag_name": schema.StringAttribute{
				Description: "Name of the tag that owns this value.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
					stringvalidator.RegexMatches(tagNameRule, "must contain only letters, digits, spaces or the characters _-[]().:"),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the value, unique within the tag. Names are case sensitive, and the API has no rename, " +
					"so changing this creates a new value.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
					stringvalidator.RegexMatches(tagNameRule, "must contain only letters, digits, spaces or the characters _-[]().:"),
				},
			},
			"description": schema.StringAttribute{
				Description: "A brief description of the value.",
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
				Description: "Last update date of the value, null while it was never updated.",
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

	// A deleted parent tag takes its values with it, and the API reports it the
	// same way as a missing value.
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

	updated, err := r.values.Update(
		ctx,
		state.TagName.ValueString(),
		state.Name.ValueString(),
		buildUpdateTagValueRequest(plan),
	)
	if err != nil {
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

// buildUpdateTagValueRequest always states the desired description: the endpoint
// requires the field, and "" is what clears it.
func buildUpdateTagValueRequest(plan tagValueResourceModel) tagSDK.UpdateTagValueRequest {
	description := ""
	if planned := utils.KnownStringPointer(plan.Description); planned != nil {
		description = *planned
	}

	return tagSDK.UpdateTagValueRequest{Description: &description}
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
