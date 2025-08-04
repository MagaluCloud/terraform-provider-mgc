package datasources

import (
	"context"
	"fmt"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DBaaSParameterModel struct {
	ParameterGroupID types.String       `tfsdk:"parameter_group_id"`
	Parameters       []ParameterElement `tfsdk:"parameters"`
}

type ParameterElement struct {
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type DataSourceDbParametersList struct {
	dbaasParameters dbSDK.ParameterService
}

func NewDataSourceDbParametersList() datasource.DataSource {
	return &DataSourceDbParametersList{}
}

func (r *DataSourceDbParametersList) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_parameters"
}

func (r *DataSourceDbParametersList) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cfg, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Expected tfutil.DataConfig")
		return
	}
	r.dbaasParameters = dbSDK.New(&cfg.CoreConfig).Parameters()
}

func (r *DataSourceDbParametersList) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all parameters in a parameter group",
		Attributes: map[string]schema.Attribute{
			"parameter_group_id": schema.StringAttribute{
				Description: "ID of the parameter group",
				Required:    true,
			},
			"parameters": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Parameter ID",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Parameter name",
						},
						"value": schema.StringAttribute{
							Computed:    true,
							Description: "Parameter value",
						},
					},
				},
			},
		},
	}
}

func (r *DataSourceDbParametersList) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DBaaSParameterModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, err := r.dbaasParameters.List(ctx, dbSDK.ListParametersOptions{
		ParameterGroupID: data.ParameterGroupID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	for _, p := range list {
		data.Parameters = append(data.Parameters, ParameterElement{
			ID:    types.StringValue(p.ID),
			Name:  types.StringValue(p.Name),
			Value: types.StringValue(fmt.Sprintf("%v", p.Value)),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
