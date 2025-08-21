package lbaas

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	lbSDK "github.com/MagaluCloud/mgc-sdk-go/lbaas"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

type DataSourceLbaasNetworks struct {
	lbNetworkLB lbSDK.NetworkLoadBalancerService
}

func NewDataSourceLbaasNetworks() datasource.DataSource {
	return &DataSourceLbaasNetworks{}
}

func (r *DataSourceLbaasNetworks) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_networks"
}

func (r *DataSourceLbaasNetworks) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	lbaasClient := lbSDK.New(&dataConfig.CoreConfig)
	r.lbNetworkLB = lbaasClient.NetworkLoadBalancers()
}

func (r *DataSourceLbaasNetworks) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List network load balancers. Each item exposes root attributes and child IDs.",
		Attributes: map[string]schema.Attribute{
			"load_balancers": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of network load balancers.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The unique identifier of the load balancer.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the load balancer.",
						},
						"description": schema.StringAttribute{
							Computed:    true,
							Description: "The description of the load balancer.",
						},
						"public_ip_id": schema.StringAttribute{
							Computed:    true,
							Description: "The ID of the public IP associated with the load balancer, if any.",
						},
						"subnetpool_id": schema.StringAttribute{
							Computed:    true,
							Description: "The ID of the subnet pool for the load balancer.",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "The type of the load balancer.",
						},
						"visibility": schema.StringAttribute{
							Computed:    true,
							Description: "The visibility of the load balancer. Possible values: 'internal', 'external'.",
						},
						"vpc_id": schema.StringAttribute{
							Computed:    true,
							Description: "The ID of the VPC where the load balancer is deployed.",
						},
						"acls": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "List of ACL IDs.",
						},
						"backends": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "List of backend IDs.",
						},
						"health_checks": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "List of health check IDs.",
						},
						"listeners": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "List of listener IDs.",
						},
						"tls_certificates": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "List of TLS certificate IDs.",
						},
					},
				},
			},
		},
	}
}

func (r *DataSourceLbaasNetworks) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data LbaasNetworksListModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sdkList, err := r.lbNetworkLB.List(ctx, lbSDK.ListNetworkLoadBalancerRequest{})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.LoadBalancers = make([]lbNetworkItemModel, 0, len(sdkList))
	for _, sdkLB := range sdkList {
		var publicIPID types.String
		if len(sdkLB.PublicIPs) == 1 {
			publicIPID = types.StringValue(sdkLB.PublicIPs[0].ID)
		} else {
			publicIPID = types.StringNull()
		}

		aclsIDs := make([]string, 0, len(sdkLB.ACLs))
		for _, a := range sdkLB.ACLs {
			aclsIDs = append(aclsIDs, a.ID)
		}
		aclsList, diags := types.ListValueFrom(ctx, types.StringType, aclsIDs)
		resp.Diagnostics.Append(diags...)

		backendsIDs := make([]string, 0, len(sdkLB.Backends))
		for _, b := range sdkLB.Backends {
			backendsIDs = append(backendsIDs, b.ID)
		}
		backendsList, diags := types.ListValueFrom(ctx, types.StringType, backendsIDs)
		resp.Diagnostics.Append(diags...)

		healthChecksIDs := make([]string, 0, len(sdkLB.HealthChecks))
		for _, h := range sdkLB.HealthChecks {
			healthChecksIDs = append(healthChecksIDs, h.ID)
		}
		healthChecksList, diags := types.ListValueFrom(ctx, types.StringType, healthChecksIDs)
		resp.Diagnostics.Append(diags...)

		listenersIDs := make([]string, 0, len(sdkLB.Listeners))
		for _, l := range sdkLB.Listeners {
			listenersIDs = append(listenersIDs, l.ID)
		}
		listenersList, diags := types.ListValueFrom(ctx, types.StringType, listenersIDs)
		resp.Diagnostics.Append(diags...)

		certsIDs := make([]string, 0, len(sdkLB.TLSCertificates))
		for _, c := range sdkLB.TLSCertificates {
			certsIDs = append(certsIDs, c.ID)
		}
		certsList, diags := types.ListValueFrom(ctx, types.StringType, certsIDs)
		resp.Diagnostics.Append(diags...)

		data.LoadBalancers = append(data.LoadBalancers, lbNetworkItemModel{
			ID:           types.StringValue(sdkLB.ID),
			Name:         types.StringValue(sdkLB.Name),
			Description:  types.StringPointerValue(sdkLB.Description),
			PublicIPID:   publicIPID,
			SubnetpoolID: types.StringPointerValue(sdkLB.SubnetPoolID),
			Type:         types.StringValue(sdkLB.Type),
			Visibility:   types.StringValue(string(sdkLB.Visibility)),
			VPCID:        types.StringValue(sdkLB.VPCID),

			ACLs:            aclsList,
			Backends:        backendsList,
			HealthChecks:    healthChecksList,
			Listeners:       listenersList,
			TLSCertificates: certsList,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
