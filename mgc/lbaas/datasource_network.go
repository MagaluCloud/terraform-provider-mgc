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

type DataSourceLbaasNetwork struct {
	lbNetworkLB lbSDK.NetworkLoadBalancerService
}

func NewDataSourceLbaasNetwork() datasource.DataSource {
	return &DataSourceLbaasNetwork{}
}

func (r *DataSourceLbaasNetwork) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_network"
}

func (r *DataSourceLbaasNetwork) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *DataSourceLbaasNetwork) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get the details of a network load balancer.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
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
			"acls": schema.ListNestedAttribute{
				Description: "Access Control Lists for the load balancer.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the ACL rule.",
							Computed:    true,
						},
						"action": schema.StringAttribute{
							Description: "The action for the ACL rule. Valid values: 'ALLOW', 'DENY', 'DENY_UNSPECIFIED'. Note: values are case-sensitive and must be uppercase.",
							Computed:    true,
						},
						"ethertype": schema.StringAttribute{
							Description: "The ethertype for the ACL rule.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the ACL rule.",
							Computed:    true,
						},
						"protocol": schema.StringAttribute{
							Description: "The protocol for the ACL rule.",
							Computed:    true,
						},
						"remote_ip_prefix": schema.StringAttribute{
							Description: "The remote IP prefix for the ACL rule.",
							Computed:    true,
						},
					},
				},
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
	}
}

func (r *DataSourceLbaasNetwork) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var id types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lb, err := r.lbNetworkLB.Get(ctx, id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	var publicIPID types.String
	if len(lb.PublicIPs) == 1 {
		publicIPID = types.StringValue(lb.PublicIPs[0].ID)
	} else {
		publicIPID = types.StringNull()
	}

	backendsIDs := make([]string, 0, len(lb.Backends))
	for _, b := range lb.Backends {
		backendsIDs = append(backendsIDs, b.ID)
	}
	backendsList, diags := types.ListValueFrom(ctx, types.StringType, backendsIDs)
	resp.Diagnostics.Append(diags...)

	healthChecksIDs := make([]string, 0, len(lb.HealthChecks))
	for _, hc := range lb.HealthChecks {
		healthChecksIDs = append(healthChecksIDs, hc.ID)
	}
	healthChecksList, diags := types.ListValueFrom(ctx, types.StringType, healthChecksIDs)
	resp.Diagnostics.Append(diags...)

	listenersIDs := make([]string, 0, len(lb.Listeners))
	for _, l := range lb.Listeners {
		listenersIDs = append(listenersIDs, l.ID)
	}
	listenersList, diags := types.ListValueFrom(ctx, types.StringType, listenersIDs)
	resp.Diagnostics.Append(diags...)

	certsIDs := make([]string, 0, len(lb.TLSCertificates))
	for _, c := range lb.TLSCertificates {
		certsIDs = append(certsIDs, c.ID)
	}
	certsList, diags := types.ListValueFrom(ctx, types.StringType, certsIDs)
	resp.Diagnostics.Append(diags...)

	aclModels := make([]ACLModel, len(lb.ACLs))
	for i, acl := range lb.ACLs {
		aclModels[i] = ACLModel{
			ID:             types.StringValue(acl.ID),
			Action:         types.StringValue(acl.Action),
			Ethertype:      types.StringValue(string(acl.Ethertype)),
			Protocol:       types.StringValue(string(acl.Protocol)),
			Name:           types.StringPointerValue(acl.Name),
			RemoteIPPrefix: types.StringValue(acl.RemoteIPPrefix),
		}
	}

	state := lbNetworkItemModel{
		ID:              types.StringValue(lb.ID),
		Name:            types.StringValue(lb.Name),
		Description:     types.StringPointerValue(lb.Description),
		PublicIPID:      publicIPID,
		SubnetpoolID:    types.StringPointerValue(lb.SubnetPoolID),
		Type:            types.StringValue(lb.Type),
		Visibility:      types.StringValue(string(lb.Visibility)),
		VPCID:           types.StringValue(lb.VPCID),
		ACLs:            aclModels,
		Backends:        backendsList,
		HealthChecks:    healthChecksList,
		Listeners:       listenersList,
		TLSCertificates: certsList,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
