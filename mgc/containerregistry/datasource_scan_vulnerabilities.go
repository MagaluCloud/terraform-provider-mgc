package containerregistry

import (
	"context"

	crSDK "github.com/MagaluCloud/mgc-sdk-go/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceCRScanVulnerabilities{}

type DataSourceCRScanVulnerabilities struct {
	scansService crSDK.ScansService
}

type crVulnerability struct {
	CveID          types.String   `tfsdk:"cve_id"`
	Severity       types.String   `tfsdk:"severity"`
	IsAllowlisted  types.Bool     `tfsdk:"is_allowlisted"`
	PackageName    types.String   `tfsdk:"package_name"`
	CurrentVersion types.String   `tfsdk:"current_version"`
	FixedVersion   types.String   `tfsdk:"fixed_version"`
	Fixable        types.Bool     `tfsdk:"fixable"`
	TargetPath     types.String   `tfsdk:"target_path"`
	Description    types.String   `tfsdk:"description"`
	Links          []types.String `tfsdk:"links"`
	Cvss           types.Object   `tfsdk:"cvss"`
}

type crScanVulnerabilitiesModel struct {
	ScanID          types.String      `tfsdk:"scan_id"`
	Vulnerabilities []crVulnerability `tfsdk:"vulnerabilities"`
}

var cvssSourceAttrTypes = map[string]attr.Type{
	"score":  types.Float64Type,
	"vector": types.StringType,
}

var cvssMaxAttrTypes = map[string]attr.Type{
	"source": types.StringType,
	"score":  types.Float64Type,
	"vector": types.StringType,
}

var cvssAttrTypes = map[string]attr.Type{
	"preferred": types.ObjectType{AttrTypes: cvssSourceAttrTypes},
	"max":       types.ObjectType{AttrTypes: cvssMaxAttrTypes},
	"by_source": types.MapType{ElemType: types.ObjectType{AttrTypes: cvssSourceAttrTypes}},
}

func NewDataSourceCRScanVulnerabilities() datasource.DataSource {
	return &DataSourceCRScanVulnerabilities{}
}

func (r *DataSourceCRScanVulnerabilities) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_registry_scan_vulnerabilities"
}

func (r *DataSourceCRScanVulnerabilities) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *DataSourceCRScanVulnerabilities) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns the list of vulnerabilities found in a Container Registry image scan.",
		Attributes: map[string]schema.Attribute{
			"scan_id": schema.StringAttribute{
				Description: "ID of the scan whose vulnerabilities will be listed",
				Required:    true,
			},
			"vulnerabilities": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"cve_id":          schema.StringAttribute{Computed: true},
						"severity":        schema.StringAttribute{Computed: true},
						"is_allowlisted":  schema.BoolAttribute{Computed: true},
						"package_name":    schema.StringAttribute{Computed: true},
						"current_version": schema.StringAttribute{Computed: true},
						"fixed_version":   schema.StringAttribute{Computed: true},
						"fixable":         schema.BoolAttribute{Computed: true},
						"target_path":     schema.StringAttribute{Computed: true},
						"description":     schema.StringAttribute{Computed: true},
						"links": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
						"cvss": schema.ObjectAttribute{
							Computed: true,
							AttributeTypes: map[string]attr.Type{
								"preferred": types.ObjectType{AttrTypes: cvssSourceAttrTypes},
								"max":       types.ObjectType{AttrTypes: cvssMaxAttrTypes},
								"by_source": types.MapType{ElemType: types.ObjectType{AttrTypes: cvssSourceAttrTypes}},
							},
						},
					},
				},
			},
		},
	}
}

func (r *DataSourceCRScanVulnerabilities) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data crScanVulnerabilitiesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vulns, err := r.scansService.ListVulnerabilitiesAll(ctx, data.ScanID.ValueString(), crSDK.VulnerabilitiesFilterOptions{})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	for i := range vulns {
		v := vulns[i]

		cvssObj, diags := cvssToObject(ctx, v.Cvss)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		links := make([]types.String, 0, len(v.Links))
		for _, l := range v.Links {
			links = append(links, types.StringValue(l))
		}

		data.Vulnerabilities = append(data.Vulnerabilities, crVulnerability{
			CveID:          types.StringValue(v.CveID),
			Severity:       types.StringValue(v.Severity),
			IsAllowlisted:  types.BoolValue(v.IsAllowlisted),
			PackageName:    types.StringValue(v.PackageName),
			CurrentVersion: types.StringValue(v.CurrentVersion),
			FixedVersion:   types.StringPointerValue(v.FixedVersion),
			Fixable:        types.BoolValue(v.Fixable),
			TargetPath:     types.StringValue(v.TargetPath),
			Description:    types.StringValue(v.Description),
			Links:          links,
			Cvss:           cvssObj,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func cvssSourceObject(_ context.Context, src *crSDK.VulnerabilityCvssSourceResponse) (types.Object, diag.Diagnostics) {
	if src == nil {
		return types.ObjectNull(cvssSourceAttrTypes), nil
	}
	return types.ObjectValue(cvssSourceAttrTypes, map[string]attr.Value{
		"score":  types.Float64PointerValue(src.Score),
		"vector": types.StringPointerValue(src.Vector),
	})
}

func cvssMaxObject(_ context.Context, max *crSDK.VulnerabilityCvssMaxResponse) (types.Object, diag.Diagnostics) {
	if max == nil {
		return types.ObjectNull(cvssMaxAttrTypes), nil
	}
	return types.ObjectValue(cvssMaxAttrTypes, map[string]attr.Value{
		"source": types.StringPointerValue(max.Source),
		"score":  types.Float64PointerValue(max.Score),
		"vector": types.StringPointerValue(max.Vector),
	})
}

func cvssToObject(ctx context.Context, c crSDK.VulnerabilityCvssResponse) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	preferred, d := cvssSourceObject(ctx, c.Preferred)
	diags.Append(d...)
	if diags.HasError() {
		return types.ObjectNull(cvssAttrTypes), diags
	}

	max, d := cvssMaxObject(ctx, c.Max)
	diags.Append(d...)
	if diags.HasError() {
		return types.ObjectNull(cvssAttrTypes), diags
	}

	bySource := make(map[string]attr.Value, len(c.BySource))
	for k, v := range c.BySource {
		obj, d := types.ObjectValue(cvssSourceAttrTypes, map[string]attr.Value{
			"score":  types.Float64PointerValue(v.Score),
			"vector": types.StringPointerValue(v.Vector),
		})
		diags.Append(d...)
		if diags.HasError() {
			return types.ObjectNull(cvssAttrTypes), diags
		}
		bySource[k] = obj
	}

	bySourceMap, d := types.MapValue(types.ObjectType{AttrTypes: cvssSourceAttrTypes}, bySource)
	diags.Append(d...)
	if diags.HasError() {
		return types.ObjectNull(cvssAttrTypes), diags
	}

	obj, d := types.ObjectValue(cvssAttrTypes, map[string]attr.Value{
		"preferred": preferred,
		"max":       max,
		"by_source": bySourceMap,
	})
	diags.Append(d...)
	return obj, diags
}
