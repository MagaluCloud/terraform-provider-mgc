package datasources

import (
	"context"

	sdkAzs "github.com/MagaluCloud/mgc-sdk-go/availabilityzones"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceAvailabilityZones{}

type DataSourceAvailabilityZones struct {
	sdkClient sdkAzs.Service
}

func NewDataSourceAvailabilityZones() datasource.DataSource {
	return &DataSourceAvailabilityZones{}
}

func (r *DataSourceAvailabilityZones) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_availability_zones"
}

type Regions struct {
	Regions []Region `tfsdk:"regions"`
}

type Region struct {
	Region            types.String        `tfsdk:"region"`
	AvailabilityZones []AvailabilityZones `tfsdk:"availability_zones"`
}

type AvailabilityZones struct {
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	BlockType        types.String `tfsdk:"block_type"`
}

func (r *DataSourceAvailabilityZones) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)

	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	r.sdkClient = sdkAzs.New(&dataConfig.CoreConfig).AvailabilityZones()
}

func (r *DataSourceAvailabilityZones) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List of available regions and availability zones.",
		Attributes: map[string]schema.Attribute{
			"regions": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"region": schema.StringAttribute{
							Computed: true,
						},
						"availability_zones": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"availability_zone": schema.StringAttribute{
										Computed: true,
									},
									"block_type": schema.StringAttribute{
										Computed: true,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
func (r *DataSourceAvailabilityZones) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data Regions

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	sdkOutput, err := r.sdkClient.List(ctx, sdkAzs.ListOptions{ShowBlocked: false})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	for _, region := range sdkOutput {
		var regionData Region
		regionData.Region = types.StringValue(region.ID)
		for _, az := range region.AvailabilityZones {
			regionData.AvailabilityZones = append(regionData.AvailabilityZones, AvailabilityZones{
				AvailabilityZone: types.StringValue(az.ID),
				BlockType:        types.StringPointerValue(removeNoneFromString(string(az.BlockType))),
			})
		}
		data.Regions = append(data.Regions, regionData)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func removeNoneFromString(str string) *string {
	if str == "none" {
		return nil
	}
	return &str
}
