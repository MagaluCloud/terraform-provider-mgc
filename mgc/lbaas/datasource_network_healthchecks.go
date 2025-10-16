package lbaas

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	lbSDK "github.com/MagaluCloud/mgc-sdk-go/lbaas"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

type DataSourceLbaasNetworkHealthChecks struct {
	lbNetworkHeathCheck lbSDK.NetworkHealthCheckService
}

func NewDataSourceLbaasNetworkHealthChecks() datasource.DataSource {
	return &DataSourceLbaasNetworkHealthChecks{}
}

func (r *DataSourceLbaasNetworkHealthChecks) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_network_healthchecks"
}

func (r *DataSourceLbaasNetworkHealthChecks) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	client := lbSDK.New(&dataConfig.CoreConfig)
	r.lbNetworkHeathCheck = client.NetworkHealthChecks()
}

func (r *DataSourceLbaasNetworkHealthChecks) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List Health Checks for a Network Load Balancer.",
		Attributes: map[string]schema.Attribute{
			"lb_id": schema.StringAttribute{
				Required:    true,
				Description: "The Network Load Balancer ID.",
			},
			"health_checks": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of health checks.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The Health Check ID.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the health check.",
						},
						"description": schema.StringAttribute{
							Computed:    true,
							Description: "The description of the health check.",
						},
						"protocol": schema.StringAttribute{
							Computed:    true,
							Description: "The protocol for the health check. Example: 'tcp', 'http'.",
						},
						"port": schema.Int64Attribute{
							Computed:    true,
							Description: "The port for health checks.",
						},
						"path": schema.StringAttribute{
							Computed:    true,
							Description: "The path for HTTP health checks.",
						},
						"interval_seconds": schema.Int64Attribute{
							Computed:    true,
							Description: "The interval between health checks.",
						},
						"timeout_seconds": schema.Int64Attribute{
							Computed:    true,
							Description: "The timeout for health checks.",
						},
						"healthy_status_code": schema.Int64Attribute{
							Computed:    true,
							Description: "The HTTP status code considered healthy.",
						},
						"healthy_threshold_count": schema.Int64Attribute{
							Computed:    true,
							Description: "The number of consecutive successful health checks required.",
						},
						"initial_delay_seconds": schema.Int64Attribute{
							Computed:    true,
							Description: "The initial delay before starting health checks.",
						},
						"unhealthy_threshold_count": schema.Int64Attribute{
							Computed:    true,
							Description: "The number of consecutive failed health checks before marking unhealthy.",
						},
					},
				},
			},
		},
	}
}

type healthCheckListState struct {
	LBID         types.String       `tfsdk:"lb_id"`
	HealthChecks []HealthCheckModel `tfsdk:"health_checks"`
}

func (r *DataSourceLbaasNetworkHealthChecks) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var lbID types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("lb_id"), &lbID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lb, err := r.lbNetworkHeathCheck.ListAll(ctx, lbID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	state := healthCheckListState{
		LBID:         lbID,
		HealthChecks: make([]HealthCheckModel, 0, len(lb)),
	}
	for _, hc := range lb {
		state.HealthChecks = append(state.HealthChecks, HealthCheckModel{
			ID:                      types.StringValue(hc.ID),
			Name:                    types.StringValue(hc.Name),
			Description:             types.StringPointerValue(hc.Description),
			Protocol:                types.StringValue(string(hc.Protocol)),
			Port:                    types.Int64Value(int64(hc.Port)),
			Path:                    types.StringPointerValue(hc.Path),
			IntervalSeconds:         types.Int64Value(int64(hc.IntervalSeconds)),
			TimeoutSeconds:          types.Int64Value(int64(hc.TimeoutSeconds)),
			HealthyStatusCode:       types.Int64Value(int64(hc.HealthyStatusCode)),
			HealthyThresholdCount:   types.Int64Value(int64(hc.HealthyThresholdCount)),
			InitialDelaySeconds:     types.Int64Value(int64(hc.InitialDelaySeconds)),
			UnhealthyThresholdCount: types.Int64Value(int64(hc.UnhealthyThresholdCount)),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
