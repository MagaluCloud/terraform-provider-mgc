package containerregistry

import (
	"context"

	crSDK "github.com/MagaluCloud/mgc-sdk-go/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceCRScans{}

type DataSourceCRScans struct {
	scansService crSDK.ScansService
}

type crScan struct {
	ID              types.String `tfsdk:"id"`
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

type crScansList struct {
	RegistryID   types.String `tfsdk:"registry_id"`
	RepositoryID types.String `tfsdk:"repository_id"`
	DigestOrTag  types.String `tfsdk:"digest_or_tag"`
	Scans        []crScan     `tfsdk:"scans"`
}

func NewDataSourceCRScans() datasource.DataSource {
	return &DataSourceCRScans{}
}

func (r *DataSourceCRScans) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_registry_scans"
}

func (r *DataSourceCRScans) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	r.scansService = crSDK.New(&dataConfig.CoreConfig).Scans()
}

func (r *DataSourceCRScans) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the scan history for a specific image in a Container Registry.",
		Attributes: map[string]schema.Attribute{
			"registry_id": schema.StringAttribute{
				Description: "ID of the registry containing the image",
				Required:    true,
			},
			"repository_id": schema.StringAttribute{
				Description: "ID of the repository containing the image",
				Required:    true,
			},
			"digest_or_tag": schema.StringAttribute{
				Description: "Digest or tag identifying the image",
				Required:    true,
			},
			"scans": schema.ListNestedAttribute{
				Description: "Scan history for the image",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Unique identifier of the scan",
							Computed:    true,
						},
						"digest": schema.StringAttribute{
							Description: "Digest of the scanned image",
							Computed:    true,
						},
						"input_tag": schema.StringAttribute{
							Description: "Tag used when the scan was scheduled",
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
						},
						"updated_at": schema.StringAttribute{
							Description: "Timestamp when the scan was last updated",
							Computed:    true,
						},
						"finished_at": schema.StringAttribute{
							Description: "Timestamp when the scan finished",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *DataSourceCRScans) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data crScansList
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scans, err := r.scansService.ListByImageAll(ctx, crSDK.ListByImageInput{
		RegistryID:   data.RegistryID.ValueString(),
		RepositoryID: data.RepositoryID.ValueString(),
		DigestOrTag:  data.DigestOrTag.ValueString(),
	}, crSDK.ImageScansFilterOptions{})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	for i := range scans {
		s := scans[i]
		severityObj, diags := severitySummaryToObject(ctx, s.SeveritySummary)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		data.Scans = append(data.Scans, crScan{
			ID:              types.StringValue(s.ID),
			Digest:          types.StringValue(s.Digest),
			InputTag:        types.StringPointerValue(s.InputTag),
			Status:          types.StringValue(s.Status),
			OS:              types.StringPointerValue(s.OS),
			Architecture:    types.StringPointerValue(s.Architecture),
			SeveritySummary: severityObj,
			CreatedAt:       types.StringValue(s.CreatedAt),
			UpdatedAt:       types.StringValue(s.UpdatedAt),
			FinishedAt:      types.StringPointerValue(s.FinishedAt),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
