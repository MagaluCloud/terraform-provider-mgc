package lbaas

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	lbSDK "github.com/MagaluCloud/mgc-sdk-go/lbaas"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

const LoadBalancerTimeout = 90 * time.Minute

type LoadBalancerResource struct {
	lbNetworkBackend        lbSDK.NetworkBackendService
	lbNetworkACL            lbSDK.NetworkACLService
	lbNetworkHealthCheck    lbSDK.NetworkHealthCheckService
	lbNetworkListener       lbSDK.NetworkListenerService
	lbNetworkTLSCertificate lbSDK.NetworkCertificateService
	lbNetworkTarget         lbSDK.NetworkBackendTargetService
	lbNetworkLB             lbSDK.NetworkLoadBalancerService
}

func NewLoadBalancerResource() resource.Resource {
	return &LoadBalancerResource{}
}

func (r *LoadBalancerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_network"
}

func (r *LoadBalancerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	lbaasClient := lbSDK.New(&dataConfig.CoreConfig)
	r.lbNetworkBackend = lbaasClient.NetworkBackends()
	r.lbNetworkACL = lbaasClient.NetworkACLs()
	r.lbNetworkHealthCheck = lbaasClient.NetworkHealthChecks()
	r.lbNetworkListener = lbaasClient.NetworkListeners()
	r.lbNetworkTLSCertificate = lbaasClient.NetworkCertificates()
	r.lbNetworkTarget = lbaasClient.NetworkBackendTargets()
	r.lbNetworkLB = lbaasClient.NetworkLoadBalancers()
}

func (r *LoadBalancerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages network load balancers in Magalu Cloud.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the load balancer.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the load balancer.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the load balancer.",
				Optional:    true,
			},
			"public_ip_id": schema.StringAttribute{
				Description: "The ID of the public IP associated with the load balancer. Required for external load balancers, must be omitted for internal load balancers.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subnetpool_id": schema.StringAttribute{
				Description: "The ID of the subnet pool for the load balancer.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The type of the load balancer. Only 'proxy' type is currently supported by the API.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("proxy"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"visibility": schema.StringAttribute{
				Description: "The visibility of the load balancer. Valid values: 'internal' (accessible only within VPC), 'external' (accessible from internet).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("internal", "external"),
					VisibilityValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Description: "The ID of the VPC where the load balancer is deployed.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"acls": schema.ListNestedAttribute{
				Description: "Access Control Lists for the load balancer.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the ACL rule.",
							Computed:    true,
						},
						"action": schema.StringAttribute{
							Description: "The action for the ACL rule. Valid values: 'ALLOW', 'DENY', 'DENY_UNSPECIFIED'. Note: values are case-sensitive and must be uppercase.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("ALLOW", "DENY", "DENY_UNSPECIFIED"),
							},
						},
						"ethertype": schema.StringAttribute{
							Description: "The ethertype for the ACL rule.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("IPv4", "IPv6"),
							},
						},
						"name": schema.StringAttribute{
							Description: "The name of the ACL rule.",
							Required:    true,
						},
						"protocol": schema.StringAttribute{
							Description: "The protocol for the ACL rule.",
							Optional:    true,
						},
						"remote_ip_prefix": schema.StringAttribute{
							Description: "The remote IP prefix for the ACL rule.",
							Optional:    true,
						},
					},
				},
			},
			"backends": schema.ListNestedAttribute{
				Description: "Backend configurations for the load balancer.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the backend.",
							Computed:    true,
						},
						"balance_algorithm": schema.StringAttribute{
							Description: "The load balancing algorithm.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("round_robin"),
							},
						},
						"description": schema.StringAttribute{
							Description: "The description of the backend.",
							Optional:    true,
						},
						"health_check_name": schema.StringAttribute{
							Description: "The name of the health check associated with this backend.",
							Optional:    true,
						},
						"close_connections_on_host_health_failure": schema.BoolAttribute{
							Description: "Whether to close connections when a host health check fails.",
							Optional:    true,
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the backend.",
							Required:    true,
						},
						"panic_threshold": schema.Float64Attribute{
							Description: "The panic threshold percentage for the backend.",
							Optional:    true,
							Validators: []validator.Float64{
								float64validator.Between(0, 100),
							},
						},
						"targets": schema.ListNestedAttribute{
							Description: "The targets for this backend.",
							Required:    true,
							Validators: []validator.List{
								TargetValidator{},
							},
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Description: "The unique identifier of the target.",
										Computed:    true,
									},
									"nic_id": schema.StringAttribute{
										Description: "The NIC ID of the target. Required when targets_type is 'instance', must be empty when targets_type is 'raw'.",
										Optional:    true,
									},
									"ip_address": schema.StringAttribute{
										Description: "The IP address of the target. Required when targets_type is 'raw', must be empty when targets_type is 'instance'.",
										Optional:    true,
									},
									"port": schema.Int64Attribute{
										Description: "The port of the target.",
										Required:    true,
										Validators: []validator.Int64{
											int64validator.Between(1, 65535),
										},
									},
								},
							},
						},
						"targets_type": schema.StringAttribute{
							Description: "The type of targets. Valid values: 'instance' (requires nic_id and port), 'raw' (requires ip_address and port).",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("instance", "raw"),
							},
						},
					},
				},
			},
			"health_checks": schema.ListNestedAttribute{
				Description: "Health check configurations for the load balancer.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the health check.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "The description of the health check.",
							Optional:    true,
						},
						"healthy_status_code": schema.Int64Attribute{
							Description: "The HTTP status code considered healthy.",
							Optional:    true,
							Computed:    true,
							Validators: []validator.Int64{
								int64validator.Between(200, 599),
							},
						},
						"healthy_threshold_count": schema.Int64Attribute{
							Description: "The number of consecutive successful health checks required.",
							Optional:    true,
							Computed:    true,
							Validators: []validator.Int64{
								int64validator.Between(1, 10),
							},
						},
						"initial_delay_seconds": schema.Int64Attribute{
							Description: "The initial delay before starting health checks.",
							Optional:    true,
							Computed:    true,
							Validators: []validator.Int64{
								int64validator.Between(0, 300),
							},
						},
						"interval_seconds": schema.Int64Attribute{
							Description: "The interval between health checks.",
							Optional:    true,
							Computed:    true,
							Validators: []validator.Int64{
								int64validator.Between(1, 300),
							},
						},
						"name": schema.StringAttribute{
							Description: "The name of the health check.",
							Required:    true,
						},
						"path": schema.StringAttribute{
							Description: "The path for HTTP health checks.",
							Optional:    true,
						},
						"port": schema.Int64Attribute{
							Description: "The port for health checks.",
							Required:    true,
							Validators: []validator.Int64{
								int64validator.Between(1, 65535),
							},
						},
						"protocol": schema.StringAttribute{
							Description: "The protocol for health checks. Valid values: 'tcp', 'http'. Note: values are case-sensitive and must be lowercase.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("tcp", "http"),
							},
						},
						"timeout_seconds": schema.Int64Attribute{
							Description: "The timeout for health checks.",
							Optional:    true,
							Computed:    true,
							Validators: []validator.Int64{
								int64validator.Between(1, 300),
							},
						},
						"unhealthy_threshold_count": schema.Int64Attribute{
							Description: "The number of consecutive failed health checks before marking unhealthy.",
							Optional:    true,
							Computed:    true,
							Validators: []validator.Int64{
								int64validator.Between(1, 10),
							},
						},
					},
				},
			},
			"listeners": schema.ListNestedAttribute{
				Description: "Listener configurations for the load balancer.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the listener.",
							Computed:    true,
						},
						"backend_name": schema.StringAttribute{
							Description: "The name of the backend associated with this listener.",
							Required:    true,
						},
						"description": schema.StringAttribute{
							Description: "The description of the listener.",
							Optional:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the listener.",
							Required:    true,
						},
						"port": schema.Int64Attribute{
							Description: "The port for the listener.",
							Required:    true,
							Validators: []validator.Int64{
								int64validator.Between(1, 65535),
							},
						},
						"protocol": schema.StringAttribute{
							Description: "The protocol for the listener. Valid values: 'tcp' (for network load balancers), 'tls' (for SSL/TLS termination). Note: values are case-sensitive and must be lowercase.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("tcp", "tls"),
							},
						},
						"tls_certificate_name": schema.StringAttribute{
							Description: "The name of the TLS certificate for HTTPS listeners.",
							Optional:    true,
						},
					},
				},
			},
			"tls_certificates": schema.ListNestedAttribute{
				Description: "TLS certificate configurations for the load balancer.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the TLS certificate.",
							Computed:    true,
						},
						"certificate": schema.StringAttribute{
							Description: "The TLS certificate content.",
							Required:    true,
							WriteOnly:   true,
							Sensitive:   true,
						},
						"description": schema.StringAttribute{
							Description: "The description of the TLS certificate.",
							Optional:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the TLS certificate.",
							Required:    true,
						},
						"private_key": schema.StringAttribute{
							Description: "The private key for the TLS certificate.",
							Required:    true,
							Sensitive:   true,
							WriteOnly:   true,
						},
						"expiration_date": schema.StringAttribute{
							Description: "The expiration date of the TLS certificate.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *LoadBalancerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LoadBalancerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createdLB, err := r.lbNetworkLB.Create(ctx, lbSDK.CreateNetworkLoadBalancerRequest{
		Description:     data.Description.ValueStringPointer(),
		Name:            data.Name.ValueString(),
		PublicIPID:      data.PublicIPID.ValueStringPointer(),
		SubnetPoolID:    data.SubnetpoolID.ValueStringPointer(),
		Type:            data.Type.ValueStringPointer(),
		Visibility:      lbSDK.LoadBalancerVisibility(data.Visibility.ValueString()),
		VPCID:           data.VPCID.ValueString(),
		ACLs:            r.convertACLsToSDK(data.ACLs),
		Backends:        r.convertBackendsToSDK(data.Backends),
		HealthChecks:    r.convertHealthChecksToSDK(data.HealthChecks),
		Listeners:       r.convertListenersToSDK(data.Listeners),
		TLSCertificates: r.convertTLSCertificatesToSDK(data.TLSCertificates),
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.ID = types.StringValue(createdLB)
	getLB, err := r.waitLoadBalancerState(ctx, createdLB, lbSDK.LoadBalancerStatusRunning)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
	if getLB == nil {
		resp.Diagnostics.AddError("Load Balancer not found", fmt.Sprintf("Load Balancer with ID %s not found after creation.", createdLB))
		return
	}

	data = r.toTerraformModel(ctx, *getLB)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *LoadBalancerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LoadBalancerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lb, err := r.lbNetworkLB.Get(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data = r.toTerraformModel(ctx, lb)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *LoadBalancerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData LoadBalancerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stateData LoadBalancerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !planData.Description.Equal(stateData.Description) || !planData.Name.Equal(stateData.Name) {
		stateData.Description = planData.Description
		stateData.Name = planData.Name
		_, err := r.lbNetworkLB.Update(ctx, planData.ID.ValueString(), lbSDK.UpdateNetworkLoadBalancerRequest{
			Description: planData.Description.ValueStringPointer(),
			Name:        planData.Name.ValueStringPointer(),
		})
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, stateData)...)
}

func (r *LoadBalancerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LoadBalancerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deletePublicIP := true
	err := r.lbNetworkLB.Delete(ctx, data.ID.ValueString(), lbSDK.DeleteNetworkLoadBalancerRequest{
		DeletePublicIP: &deletePublicIP,
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
	_, err = r.waitLoadBalancerState(ctx, data.ID.ValueString(), lbSDK.LoadBalancerStatusDeleted)
	if err != nil {
		switch e := err.(type) {
		case *clientSDK.HTTPError:
			if e.StatusCode == http.StatusNotFound {
				return
			}
		default:
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
	}

}

func (r *LoadBalancerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *LoadBalancerResource) convertACLsToSDK(acls *[]ACLModel) []lbSDK.CreateNetworkACLRequest {
	if acls == nil {
		return nil
	}

	var aclRequests []lbSDK.CreateNetworkACLRequest
	for _, acl := range *acls {
		aclRequests = append(aclRequests, lbSDK.CreateNetworkACLRequest{
			Action:         lbSDK.AclActionType(acl.Action.ValueString()),
			Name:           acl.Name.ValueStringPointer(),
			Ethertype:      lbSDK.AclEtherType(acl.Ethertype.ValueString()),
			Protocol:       lbSDK.AclProtocol(acl.Protocol.ValueString()),
			RemoteIPPrefix: acl.RemoteIPPrefix.ValueString(),
		})
	}
	return aclRequests
}

func (r *LoadBalancerResource) convertBackendsToSDK(backends []BackendModel) []lbSDK.CreateNetworkBackendRequest {
	var backendRequests []lbSDK.CreateNetworkBackendRequest
	for _, backend := range backends {

		var targets []lbSDK.NetworkBackendInstanceTargetRequest
		for _, target := range backend.Targets {
			targets = append(targets, lbSDK.NetworkBackendInstanceTargetRequest{
				NicID:     target.NICID.ValueStringPointer(),
				Port:      target.Port.ValueInt64(),
				IPAddress: target.IPAddress.ValueStringPointer(),
			})
		}

		backendRequests = append(backendRequests, lbSDK.CreateNetworkBackendRequest{
			HealthCheckName:                     backend.HealthCheckName.ValueStringPointer(),
			Name:                                backend.Name.ValueString(),
			Description:                         backend.Description.ValueStringPointer(),
			PanicThreshold:                      backend.PanicThreshold.ValueFloat64Pointer(),
			CloseConnectionsOnHostHealthFailure: backend.CloseConnectionsOnHostHealthFailure.ValueBoolPointer(),
			BalanceAlgorithm:                    lbSDK.BackendBalanceAlgorithm(backend.BalanceAlgorithm.ValueString()),
			TargetsType:                         lbSDK.BackendType(backend.TargetsType.ValueString()),
			Targets:                             &targets,
		})
	}
	return backendRequests
}

func (r *LoadBalancerResource) convertHealthChecksToSDK(healthChecks *[]HealthCheckModel) []lbSDK.CreateNetworkHealthCheckRequest {
	if healthChecks == nil {
		return nil
	}

	var healthCheckRequests []lbSDK.CreateNetworkHealthCheckRequest
	for _, healthCheck := range *healthChecks {
		healthCheckRequests = append(healthCheckRequests, lbSDK.CreateNetworkHealthCheckRequest{
			Name:                    healthCheck.Name.ValueString(),
			Description:             healthCheck.Description.ValueStringPointer(),
			Protocol:                lbSDK.HealthCheckProtocol(healthCheck.Protocol.ValueString()),
			Port:                    int(healthCheck.Port.ValueInt64()),
			Path:                    healthCheck.Path.ValueStringPointer(),
			HealthyStatusCode:       utils.ConvertInt64PointerToIntPointer(healthCheck.HealthyStatusCode.ValueInt64Pointer()),
			IntervalSeconds:         utils.ConvertInt64PointerToIntPointer(healthCheck.IntervalSeconds.ValueInt64Pointer()),
			TimeoutSeconds:          utils.ConvertInt64PointerToIntPointer(healthCheck.TimeoutSeconds.ValueInt64Pointer()),
			InitialDelaySeconds:     utils.ConvertInt64PointerToIntPointer(healthCheck.InitialDelaySeconds.ValueInt64Pointer()),
			HealthyThresholdCount:   utils.ConvertInt64PointerToIntPointer(healthCheck.HealthyThresholdCount.ValueInt64Pointer()),
			UnhealthyThresholdCount: utils.ConvertInt64PointerToIntPointer(healthCheck.UnhealthyThresholdCount.ValueInt64Pointer()),
		})
	}
	return healthCheckRequests
}

func (r *LoadBalancerResource) convertListenersToSDK(listeners []ListenerModel) []lbSDK.NetworkListenerRequest {
	var listenerRequests []lbSDK.NetworkListenerRequest
	for _, listener := range listeners {
		listenerRequests = append(listenerRequests, lbSDK.NetworkListenerRequest{
			Name:               listener.Name.ValueString(),
			Description:        listener.Description.ValueStringPointer(),
			Port:               int(listener.Port.ValueInt64()),
			Protocol:           lbSDK.ListenerProtocol(listener.Protocol.ValueString()),
			TLSCertificateName: listener.TLSCertificateName.ValueStringPointer(),
			BackendName:        listener.BackendName.ValueString(),
		})
	}
	return listenerRequests
}

func (r *LoadBalancerResource) convertTLSCertificatesToSDK(certificates *[]TLSCertificateModel) []lbSDK.CreateNetworkCertificateRequest {
	if certificates == nil {
		return nil
	}

	var certificateRequests []lbSDK.CreateNetworkCertificateRequest
	for _, certificate := range *certificates {
		certificateRequests = append(certificateRequests, lbSDK.CreateNetworkCertificateRequest{
			Name:        certificate.Name.ValueString(),
			Description: certificate.Description.ValueStringPointer(),
			Certificate: certificate.Certificate.ValueString(),
			PrivateKey:  certificate.PrivateKey.ValueString(),
		})
	}
	return certificateRequests
}

func (r *LoadBalancerResource) toTerraformModel(ctx context.Context, lb lbSDK.NetworkLoadBalancerResponse) LoadBalancerModel {
	healthCheckIDsNames := make(map[string]string)
	var healthCheckModels []HealthCheckModel
	for _, healthCheck := range lb.HealthChecks {
		healthCheckIDsNames[healthCheck.ID] = healthCheck.Name
		healthCheckModels = append(healthCheckModels, HealthCheckModel{
			ID:                      types.StringValue(healthCheck.ID),
			Name:                    types.StringValue(healthCheck.Name),
			Description:             types.StringPointerValue(healthCheck.Description),
			Protocol:                types.StringValue(string(healthCheck.Protocol)),
			Port:                    types.Int64Value(int64(healthCheck.Port)),
			Path:                    types.StringPointerValue(healthCheck.Path),
			IntervalSeconds:         types.Int64Value(int64(healthCheck.IntervalSeconds)),
			TimeoutSeconds:          types.Int64Value(int64(healthCheck.TimeoutSeconds)),
			HealthyStatusCode:       types.Int64Value(int64(healthCheck.HealthyStatusCode)),
			HealthyThresholdCount:   types.Int64Value(int64(healthCheck.HealthyThresholdCount)),
			InitialDelaySeconds:     types.Int64Value(int64(healthCheck.InitialDelaySeconds)),
			UnhealthyThresholdCount: types.Int64Value(int64(healthCheck.UnhealthyThresholdCount)),
		})
	}

	var aclModels []ACLModel
	for _, acl := range lb.ACLs {
		aclModels = append(aclModels, ACLModel{
			ID:             types.StringValue(acl.ID),
			Action:         types.StringValue(acl.Action),
			Ethertype:      types.StringValue(string(acl.Ethertype)),
			Protocol:       types.StringValue(string(acl.Protocol)),
			Name:           types.StringPointerValue(acl.Name),
			RemoteIPPrefix: types.StringValue(acl.RemoteIPPrefix),
		})
	}

	backendIDsNames := make(map[string]string)
	var backendModels []BackendModel
	for _, backend := range lb.Backends {
		backendIDsNames[backend.ID] = backend.Name
		var targets []TargetModel
		for _, target := range backend.Targets {
			targets = append(targets, TargetModel{
				ID:        types.StringValue(target.ID),
				Port:      types.Int64PointerValue(target.Port),
				NICID:     types.StringPointerValue(target.NicID),
				IPAddress: types.StringPointerValue(target.IPAddress),
			})
		}

		var healthCheckName *string
		if backend.HealthCheckID != nil {
			name := healthCheckIDsNames[*backend.HealthCheckID]
			healthCheckName = &name
		}

		backendModels = append(backendModels, BackendModel{
			ID:                                  types.StringValue(backend.ID),
			Name:                                types.StringValue(backend.Name),
			Description:                         types.StringPointerValue(backend.Description),
			BalanceAlgorithm:                    types.StringValue(string(backend.BalanceAlgorithm)),
			HealthCheckName:                     types.StringPointerValue(healthCheckName),
			PanicThreshold:                      types.Float64PointerValue(backend.PanicThreshold),
			CloseConnectionsOnHostHealthFailure: types.BoolPointerValue(backend.CloseConnectionsOnHostHealthFailure),
			TargetsType:                         types.StringValue(string(backend.TargetsType)),
			Targets:                             targets,
		})
	}

	tlsCertificatesIDsNames := make(map[string]string)
	var tlsCertificates []TLSCertificateModel
	for _, certificate := range lb.TLSCertificates {
		tlsCertificatesIDsNames[certificate.ID] = certificate.Name
		t := TLSCertificateModel{
			Description: types.StringPointerValue(certificate.Description),
			Name:        types.StringValue(certificate.Name),
			Certificate: types.StringNull(),
			PrivateKey:  types.StringNull(),
			ID:          types.StringValue(certificate.ID),
		}

		if certificate.ExpirationDate != nil {
			t.ExpirationDate = types.StringValue(certificate.ExpirationDate.String())
		}

		tlsCertificates = append(tlsCertificates, t)
	}

	var listenerModels []ListenerModel
	for _, listener := range lb.Listeners {
		lm := ListenerModel{
			ID:          types.StringValue(listener.ID),
			Name:        types.StringValue(listener.Name),
			Description: types.StringPointerValue(listener.Description),
			Port:        types.Int64Value(int64(listener.Port)),
			Protocol:    types.StringValue(string(listener.Protocol)),
		}

		if listener.TLSCertificateID != nil {
			if tlsName, exists := tlsCertificatesIDsNames[*listener.TLSCertificateID]; exists {
				lm.TLSCertificateName = types.StringValue(tlsName)
			}
		}

		if backendName, exists := backendIDsNames[listener.BackendID]; exists {
			lm.BackendName = types.StringValue(backendName)
		}

		listenerModels = append(listenerModels, lm)
	}

	loadBalancer := LoadBalancerModel{
		ID:              types.StringValue(lb.ID),
		Name:            types.StringValue(lb.Name),
		Description:     types.StringPointerValue(lb.Description),
		SubnetpoolID:    types.StringPointerValue(lb.SubnetPoolID),
		Type:            types.StringValue(lb.Type),
		Visibility:      types.StringValue(string(lb.Visibility)),
		VPCID:           types.StringValue(lb.VPCID),
		ACLs:            &aclModels,
		Backends:        backendModels,
		HealthChecks:    &healthCheckModels,
		Listeners:       listenerModels,
		TLSCertificates: &tlsCertificates,
	}

	if len(lb.PublicIPs) == 1 {
		loadBalancer.PublicIPID = types.StringValue(lb.PublicIPs[0].ID)
	} else {
		loadBalancer.PublicIPID = types.StringNull()
	}

	return loadBalancer
}

func (r *LoadBalancerResource) waitLoadBalancerState(ctx context.Context, lbID string, desiredState lbSDK.LoadBalancerStatus) (*lbSDK.NetworkLoadBalancerResponse, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, LoadBalancerTimeout)
	defer cancel()
	for {
		select {
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("timeout waiting for load balancer %s to reach status %s", lbID, desiredState)
		case <-time.After(10 * time.Second):
			lb, err := r.lbNetworkLB.Get(ctx, lbID)
			if err != nil {
				return nil, err
			}
			if lb.Status == lbSDK.LoadBalancerStatusFailed {
				return nil, fmt.Errorf("load balancer %s is in error state", lbID)
			}
			if lb.Status == desiredState {
				return &lb, nil
			}
		}
	}
}
