package containerregistry

import (
	"context"

	crSDK "github.com/MagaluCloud/mgc-sdk-go/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceCRScanStatus{}

type DataSourceCRScanStatus struct {
	scansService crSDK.ScansService
}

type crChildScan struct {
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

type crScanStatusModel struct {
	ScanID          types.String  `tfsdk:"scan_id"`
	Digest          types.String  `tfsdk:"digest"`
	InputTag        types.String  `tfsdk:"input_tag"`
	Status          types.String  `tfsdk:"status"`
	OS              types.String  `tfsdk:"os"`
	Architecture    types.String  `tfsdk:"architecture"`
	SeveritySummary types.Object  `tfsdk:"severity_summary"`
	CreatedAt       types.String  `tfsdk:"created_at"`
	UpdatedAt       types.String  `tfsdk:"updated_at"`
	FinishedAt      types.String  `tfsdk:"finished_at"`
	ChildScans      []crChildScan `tfsdk:"child_scans"`
}

func NewDataSourceCRScanStatus() datasource.DataSource {
	return &DataSourceCRScanStatus{}
}

func (r *DataSourceCRScanStatus) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_registry_scan_status"
}

func (r *DataSourceCRScanStatus) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *DataSourceCRScanStatus) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns the current status of a Container Registry image scan.",
		Attributes: map[string]schema.Attribute{
			"scan_id": schema.StringAttribute{
				Description: "ID of the scan to inspect",
				Required:    true,
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
			"child_scans": schema.ListNestedAttribute{
				Description: "Per-architecture child scans",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":           schema.StringAttribute{Computed: true},
						"digest":       schema.StringAttribute{Computed: true},
						"input_tag":    schema.StringAttribute{Computed: true},
						"status":       schema.StringAttribute{Computed: true},
						"os":           schema.StringAttribute{Computed: true},
						"architecture": schema.StringAttribute{Computed: true},
						"severity_summary": schema.ObjectAttribute{
							Computed:       true,
							AttributeTypes: severitySummaryAttrTypes,
						},
						"created_at":  schema.StringAttribute{Computed: true},
						"updated_at":  schema.StringAttribute{Computed: true},
						"finished_at": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (r *DataSourceCRScanStatus) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data crScanStatusModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scan, err := r.scansService.Get(ctx, data.ScanID.ValueString())
	if err != nil {
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

	for i := range scan.ChildScans {
		c := scan.ChildScans[i]
		childSeverity, diags := severitySummaryToObject(ctx, c.SeveritySummary)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		data.ChildScans = append(data.ChildScans, crChildScan{
			ID:              types.StringValue(c.ID),
			Digest:          types.StringValue(c.Digest),
			InputTag:        types.StringPointerValue(c.InputTag),
			Status:          types.StringValue(c.Status),
			OS:              types.StringPointerValue(c.OS),
			Architecture:    types.StringPointerValue(c.Architecture),
			SeveritySummary: childSeverity,
			CreatedAt:       types.StringValue(c.CreatedAt),
			UpdatedAt:       types.StringValue(c.UpdatedAt),
			FinishedAt:      types.StringPointerValue(c.FinishedAt),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
