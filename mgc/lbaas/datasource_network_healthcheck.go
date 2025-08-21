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

type DataSourceLbaasNetworkHealthCheck struct {
	lbNetworkHeathCheck lbSDK.NetworkHealthCheckService
}

func NewDataSourceLbaasNetworkHealthCheck() datasource.DataSource {
	return &DataSourceLbaasNetworkHealthCheck{}
}

func (r *DataSourceLbaasNetworkHealthCheck) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_network_healthcheck"
}

func (r *DataSourceLbaasNetworkHealthCheck) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *DataSourceLbaasNetworkHealthCheck) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get a specific Network Load Balancer Health Check by Load Balancer ID and Health Check ID.",
		Attributes: map[string]schema.Attribute{
			"lb_id": schema.StringAttribute{
				Required:    true,
				Description: "The Network Load Balancer ID.",
			},
			"id": schema.StringAttribute{
				Required:    true,
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
	}
}

type healthCheckItemModel struct {
	LBID types.String `tfsdk:"lb_id"`
	HealthCheckModel
}

func (r *DataSourceLbaasNetworkHealthCheck) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var lbID, hcID types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("lb_id"), &lbID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("id"), &hcID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lb, err := r.lbNetworkHeathCheck.Get(ctx, lbID.ValueString(), hcID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	item := healthCheckItemModel{
		LBID: lbID,
		HealthCheckModel: HealthCheckModel{
			ID:                      types.StringValue(lb.ID),
			Name:                    types.StringValue(lb.Name),
			Description:             types.StringPointerValue(lb.Description),
			Protocol:                types.StringValue(string(lb.Protocol)),
			Port:                    types.Int64Value(int64(lb.Port)),
			Path:                    types.StringPointerValue(lb.Path),
			IntervalSeconds:         types.Int64Value(int64(lb.IntervalSeconds)),
			TimeoutSeconds:          types.Int64Value(int64(lb.TimeoutSeconds)),
			HealthyStatusCode:       types.Int64Value(int64(lb.HealthyStatusCode)),
			HealthyThresholdCount:   types.Int64Value(int64(lb.HealthyThresholdCount)),
			InitialDelaySeconds:     types.Int64Value(int64(lb.InitialDelaySeconds)),
			UnhealthyThresholdCount: types.Int64Value(int64(lb.UnhealthyThresholdCount)),
		},
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &item)...)
}
