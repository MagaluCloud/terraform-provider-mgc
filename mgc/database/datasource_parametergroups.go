package database

import (
	"context"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ParameterGroupListModel struct {
	Parameters []ParameterGroupModel `tfsdk:"parameter_groups"`
}

type ParameterGroupModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	EngineID    types.String `tfsdk:"engine_id"`
}

type DataSourceDbParameterList struct {
	dbaasParameterGroups dbSDK.ParameterGroupService
}

func NewDataSourceDdbaasParameterGroups() datasource.DataSource {
	return &DataSourceDbParameterList{}
}

func (r *DataSourceDbParameterList) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_parameter_groups"
}

func (r *DataSourceDbParameterList) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.dbaasParameterGroups = dbSDK.New(&dataConfig.CoreConfig).ParametersGroup()
}

func (r *DataSourceDbParameterList) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all parameter groups",
		Attributes: map[string]schema.Attribute{
			"parameter_groups": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Parameter Group ID",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Parameter Group Name",
						},
						"description": schema.StringAttribute{
							Computed:    true,
							Description: "Parameter Group Description",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "Parameter Group Type",
						},
						"engine_id": schema.StringAttribute{
							Computed:    true,
							Description: "Parameter Group Engine ID",
						},
					},
				},
			},
		},
	}
}

func (r *DataSourceDbParameterList) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	data := ParameterGroupListModel{}
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parameters, err := r.dbaasParameterGroups.List(ctx, dbSDK.ListParameterGroupsOptions{})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	for _, parameter := range parameters {
		data.Parameters = append(data.Parameters, ParameterGroupModel{
			ID:          types.StringValue(parameter.ID),
			Name:        types.StringValue(parameter.Name),
			Description: types.StringPointerValue(parameter.Description),
			Type:        types.StringValue(string(parameter.Type)),
			EngineID:    types.StringValue(parameter.EngineID),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
