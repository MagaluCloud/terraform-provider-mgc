package containerregistry

import (
	"context"
	"errors"
	"net/http"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	crSDK "github.com/MagaluCloud/mgc-sdk-go/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ContainerRegistryScanModel struct {
	ID              types.String `tfsdk:"id"`
	RegistryID      types.String `tfsdk:"registry_id"`
	RepositoryID    types.String `tfsdk:"repository_id"`
	DigestOrTag     types.String `tfsdk:"digest_or_tag"`
	Digest          types.String `tfsdk:"digest"`
	InputTag        types.String `tfsdk:"input_tag"`
	Status          types.String `tfsdk:"status"`
	OS              types.String `tfsdk:"os"`
	Architecture    types.String `tfsdk:"architecture"`
	SeveritySummary types.Object `tfsdk:"severity_summary"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
	FinishedAt      types.String `tfsdk:"finished_at"`
}

type ContainerRegistryScanResource struct {
	scansService crSDK.ScansService
}

func NewContainerRegistryScanResource() resource.Resource {
	return &ContainerRegistryScanResource{}
}

func (r *ContainerRegistryScanResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_registry_scan"
}

func (r *ContainerRegistryScanResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.scansService = crSDK.New(&dataConfig.CoreConfig).Scans()
}

func (r *ContainerRegistryScanResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schedules a vulnerability scan for an image stored in a Container Registry.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the scan",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"registry_id": schema.StringAttribute{
				Description: "ID of the registry containing the image",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"repository_id": schema.StringAttribute{
				Description: "ID of the repository containing the image",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"digest_or_tag": schema.StringAttribute{
				Description: "Digest or tag of the image to scan",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"digest": schema.StringAttribute{
				Description: "Digest of the image that was scanned",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"input_tag": schema.StringAttribute{
				Description: "Tag used to identify the image when scheduling the scan",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Current status of the scan",
				Computed:    true,
			},
			"os": schema.StringAttribute{
				Description: "Operating system of the scanned image",
				Computed:    true,
			},
			"architecture": schema.StringAttribute{
				Description: "Architecture of the scanned image",
				Computed:    true,
			},
			"severity_summary": schema.ObjectAttribute{
				Description:    "Vulnerability counts grouped by severity",
				Computed:       true,
				AttributeTypes: severitySummaryAttrTypes,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the scan was scheduled",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the scan was last updated",
				Computed:    true,
			},
			"finished_at": schema.StringAttribute{
				Description: "Timestamp when the scan finished",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ContainerRegistryScanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ContainerRegistryScanModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scheduled, err := r.scansService.Schedule(ctx, crSDK.ScheduleScanInput{
		RegistryID:   data.RegistryID.ValueString(),
		RepositoryID: data.RepositoryID.ValueString(),
		DigestOrTag:  data.DigestOrTag.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	scan, err := waitForScanCompletion(ctx, r.scansService, scheduled.ID)
	if err != nil {
		resp.Diagnostics.AddError("Container Registry scan failed", err.Error())
		return
	}

	severityObj, diags := severitySummaryToObject(ctx, scan.SeveritySummary)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(scan.ID)
	data.Digest = types.StringValue(scan.Digest)
	data.InputTag = types.StringPointerValue(scan.InputTag)
	data.Status = types.StringValue(scan.Status)
	data.OS = types.StringPointerValue(scan.OS)
	data.Architecture = types.StringPointerValue(scan.Architecture)
	data.SeveritySummary = severityObj
	data.CreatedAt = types.StringValue(scan.CreatedAt)
	data.UpdatedAt = types.StringValue(scan.UpdatedAt)
	data.FinishedAt = types.StringPointerValue(scan.FinishedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerRegistryScanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ContainerRegistryScanModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scan, err := r.scansService.Get(ctx, data.ID.ValueString())
	if err != nil {
		var httpErr *clientSDK.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	severityObj, diags := severitySummaryToObject(ctx, scan.SeveritySummary)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Digest = types.StringValue(scan.Digest)
	data.InputTag = types.StringPointerValue(scan.InputTag)
	data.Status = types.StringValue(scan.Status)
	data.OS = types.StringPointerValue(scan.OS)
	data.Architecture = types.StringPointerValue(scan.Architecture)
	data.SeveritySummary = severityObj
	data.CreatedAt = types.StringValue(scan.CreatedAt)
	data.UpdatedAt = types.StringValue(scan.UpdatedAt)
	data.FinishedAt = types.StringPointerValue(scan.FinishedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerRegistryScanResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update not supported",
		"Container Registry scans are immutable; changes to inputs require replacement",
	)
}

func (r *ContainerRegistryScanResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// A completed scan is immutable history; there is nothing to delete on the
	// server. Destroying the resource only removes it from Terraform state.
}
