package tags

//go:generate go run github.com/vektra/mockery/v2@v2.53.6 --name=ResourceService --srcpkg=github.com/MagaluCloud/mgc-sdk-go/tag --output=../internal/mocks --outpkg=mocks

import (
	"context"
	"sort"

	tagSDK "github.com/MagaluCloud/mgc-sdk-go/tag"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type tagAttachmentResourceModel struct {
	ID           types.String `tfsdk:"id"`
	ResourceID   types.String `tfsdk:"resource_id"`
	Tags         types.Map    `tfsdk:"tags"`
	ResourceType types.String `tfsdk:"resource_type"`
	Region       types.String `tfsdk:"region"`
}

type tagAttachmentResource struct {
	resources tagSDK.ResourceService
}

func NewTagAttachmentResource() resource.Resource {
	return &tagAttachmentResource{}
}

func (r *tagAttachmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_attachment"
}

func (r *tagAttachmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.resources = newTagClient(dataConfig).Resources()
}

func (r *tagAttachmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The tags attached to a cloud resource. " +
			"**This resource is authoritative**: it owns every tag of `resource_id`, so a tag attached outside Terraform " +
			"is removed on the next apply, and two configurations must not manage the same resource. " +
			"The tag and the value have to exist before being attached.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the attachment, which is the id of the tagged resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_id": schema.StringAttribute{
				Description: "Id of the resource being tagged, as used by its own product, such as the id of a cluster or of a VPC.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tags": schema.MapAttribute{
				Description: "Tags attached to the resource, as a map of tag name to value. Reference the tag and the value " +
					"instead of writing them as literals, so Terraform creates them first: " +
					"`(mgc_tag.env.name) = mgc_tag_value.prod.name` (the parentheses make the key an expression).",
				Required:    true,
				ElementType: types.StringType,
			},
			"resource_type": schema.StringAttribute{
				Description: "Type of the tagged resource, as classified by the API, such as `k8s.cluster`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region": schema.StringAttribute{
				Description: "Region of the tagged resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *tagAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tagAttachmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceID := plan.ResourceID.ValueString()
	tags, diags := tagsFromMap(ctx, plan.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	attached, err := r.resources.AttachTags(ctx, resourceID, attachRequest(tags))
	if err != nil {
		if isConflict(err) {
			resp.Diagnostics.AddError(
				"Tag already attached to the resource",
				"The resource already carries one of these tags, and the API does not allow attaching it twice. "+
					"Adopt the existing tags instead: terraform import <resource address> "+resourceID,
			)
			return
		}
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	plan = flattenAttachment(plan, *attached)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *tagAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data tagAttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The API reports a resource without any tag as missing, which is exactly
	// the state left behind when every tag is detached out of band.
	current, err := r.resources.Get(ctx, data.ResourceID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddWarning(
				"MGC Resource not found the tag attachment during refresh",
				"The resource carries no tag anymore, so the attachment has been removed from the state and will be recreated",
			)
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data = flattenAttachment(data, *current)
	data.Tags = mirrorTags(*current)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *tagAttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state tagAttachmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceID := state.ResourceID.ValueString()

	currentTags, diags := tagsFromMap(ctx, state.Tags)
	resp.Diagnostics.Append(diags...)
	plannedTags, diags := tagsFromMap(ctx, plan.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	attach, detach := diffTags(currentTags, plannedTags)
	current, applyErr := r.applyTagChanges(ctx, resourceID, attach, detach)

	// A detach-only update, and any failure along the way, leave no response to
	// trust: ask the API what the resource carries, so a partial change does not
	// leave the state lying.
	if current == nil {
		read, err := r.resources.Get(ctx, resourceID)
		if err != nil {
			if applyErr != nil {
				resp.Diagnostics.AddError(utils.ParseSDKError(applyErr))
			}
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
		current = read
	}

	plan = flattenAttachment(plan, *current)
	if applyErr != nil {
		// The apply failed, so the plan is not what the resource carries: record
		// the truth instead, which Terraform accepts because an error follows.
		plan.Tags = mirrorTags(*current)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	if applyErr != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(applyErr))
	}
}

func (r *tagAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data tagAttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags, diags := tagsFromMap(ctx, data.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Deleting the resource in its own product takes the tags with it, so a tag
	// that is already gone is a successful destroy.
	for _, name := range sortedNames(tags) {
		if err := r.resources.DetachTag(ctx, data.ResourceID.ValueString(), name); err != nil && !isNotFound(err) {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
	}
}

func (r *tagAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import ID", "The id of the tagged resource must be provided")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_id"), req.ID)...)
}

// applyTagChanges detaches before attaching: the API answers 409 when the tag is
// already attached, so a changed value has to free the name first. The attach
// answers with the whole resource, which spares a read; a detach does not, hence
// the nil.
func (r *tagAttachmentResource) applyTagChanges(ctx context.Context, resourceID string, attach map[string]string, detach []string) (*tagSDK.Resource, error) {
	for _, name := range detach {
		if err := r.resources.DetachTag(ctx, resourceID, name); err != nil && !isNotFound(err) {
			return nil, err
		}
	}

	if len(attach) == 0 {
		return nil, nil
	}

	attached, err := r.resources.AttachTags(ctx, resourceID, attachRequest(attach))
	if err != nil {
		return nil, err
	}

	return attached, nil
}

// diffTags reports what has to be attached and what has to be detached to take
// the resource from the tags it carries to the ones the configuration declares.
// A tag whose value changed shows up in both.
func diffTags(state, plan map[string]string) (attach map[string]string, detach []string) {
	attach = make(map[string]string)

	for name, value := range plan {
		current, attached := state[name]
		if attached && current == value {
			continue
		}
		if attached {
			detach = append(detach, name)
		}
		attach[name] = value
	}

	for name := range state {
		if _, declared := plan[name]; !declared {
			detach = append(detach, name)
		}
	}

	sort.Strings(detach)
	return attach, detach
}

// flattenAttachment fills the attributes the API owns. It deliberately leaves
// tags alone: after a successful apply the state has to hold exactly what the
// plan declared, and the response may carry more tags than that when someone
// attached one by hand. Read is what mirrors the API, with mirrorTags, and that
// is where the extra tag shows up as drift.
func flattenAttachment(tfData tagAttachmentResourceModel, taggedResource tagSDK.Resource) tagAttachmentResourceModel {
	tfData.ID = types.StringValue(taggedResource.ExternalID)
	tfData.ResourceID = utils.FlattenStringValue(tfData.ResourceID, &taggedResource.ExternalID)
	tfData.ResourceType = types.StringValue(string(taggedResource.ResourceType.Name))
	tfData.Region = types.StringValue(taggedResource.Region)

	return tfData
}

// mirrorTags reports every tag the resource carries, which is what makes this
// resource authoritative: whatever is not in the configuration turns into a
// removal in the next plan.
func mirrorTags(taggedResource tagSDK.Resource) types.Map {
	elements := make(map[string]attr.Value, len(taggedResource.Tags))
	for _, tag := range taggedResource.Tags {
		elements[tag.Name] = types.StringValue(tag.Value)
	}

	return types.MapValueMust(types.StringType, elements)
}

func tagsFromMap(ctx context.Context, tags types.Map) (map[string]string, diag.Diagnostics) {
	converted := make(map[string]string, len(tags.Elements()))
	diags := tags.ElementsAs(ctx, &converted, false)

	return converted, diags
}

func attachRequest(tags map[string]string) tagSDK.AttachTagsRequest {
	request := tagSDK.AttachTagsRequest{Tags: make([]tagSDK.AttachTag, 0, len(tags))}
	for _, name := range sortedNames(tags) {
		request.Tags = append(request.Tags, tagSDK.AttachTag{Name: name, Value: tags[name]})
	}

	return request
}

// sortedNames keeps every request reproducible, which matters for the cassettes
// of the acceptance tests.
func sortedNames(tags map[string]string) []string {
	names := make([]string, 0, len(tags))
	for name := range tags {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}
