package database

import (
	"context"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DataSourceDbParameter struct {
	dbaasParameterGroups dbSDK.ParameterGroupService
}

func NewDataSourceDbaaSParameterGroup() datasource.DataSource {
	return &DataSourceDbParameter{}
}

func (r *DataSourceDbParameter) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_parameter_group"
}

func (r *DataSourceDbParameter) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *DataSourceDbParameter) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get parameter group details by its ID",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "ID of the parameter group",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the parameter group",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the parameter group",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "Type of the parameter group",
				Computed:    true,
			},
			"engine_id": schema.StringAttribute{
				Description: "ID of the engine",
				Computed:    true,
			},
		},
	}
}

func (r *DataSourceDbParameter) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	data := ParameterGroupModel{}
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parameter, err := r.dbaasParameterGroups.Get(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.Description = types.StringPointerValue(parameter.Description)
	data.Type = types.StringValue(string(parameter.Type))
	data.EngineID = types.StringValue(parameter.EngineID)
	data.Name = types.StringValue(parameter.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
