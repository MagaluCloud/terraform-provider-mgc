package resources

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	lbSDK "github.com/MagaluCloud/mgc-sdk-go/lbaas"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
)

const LoadBalancerTimeout = 90 * time.Minute

type LoadBalancerModel struct {
	ID              types.String           `tfsdk:"id"`
	Name            types.String           `tfsdk:"name"`
	Description     types.String           `tfsdk:"description"`
	PublicIPID      types.String           `tfsdk:"public_ip_id"`
	SubnetpoolID    types.String           `tfsdk:"subnetpool_id"`
	Type            types.String           `tfsdk:"type"`
	Visibility      types.String           `tfsdk:"visibility"`
	VPCID           types.String           `tfsdk:"vpc_id"`
	ACLs            *[]ACLModel            `tfsdk:"acls"`
	Backends        []BackendModel         `tfsdk:"backends"`
	HealthChecks    *[]HealthCheckModel    `tfsdk:"health_checks"`
	Listeners       []ListenerModel        `tfsdk:"listeners"`
	TLSCertificates *[]TLSCertificateModel `tfsdk:"tls_certificates"`
}

type ACLModel struct {
	ID             types.String `tfsdk:"id"`
	Action         types.String `tfsdk:"action"`
	Ethertype      types.String `tfsdk:"ethertype"`
	Name           types.String `tfsdk:"name"`
	Protocol       types.String `tfsdk:"protocol"`
	RemoteIPPrefix types.String `tfsdk:"remote_ip_prefix"`
}

type BackendModel struct {
	ID                                  types.String  `tfsdk:"id"`
	BalanceAlgorithm                    types.String  `tfsdk:"balance_algorithm"`
	Description                         types.String  `tfsdk:"description"`
	HealthCheckName                     types.String  `tfsdk:"health_check_name"`
	Name                                types.String  `tfsdk:"name"`
	Targets                             []TargetModel `tfsdk:"targets"`
	TargetsType                         types.String  `tfsdk:"targets_type"`
	PanicThreshold                      types.Float64 `tfsdk:"panic_threshold"`
	CloseConnectionsOnHostHealthFailure types.Bool    `tfsdk:"close_connections_on_host_health_failure"`
}

type TargetModel struct {
	ID        types.String `tfsdk:"id"`
	NICID     types.String `tfsdk:"nic_id"`
	IPAddress types.String `tfsdk:"ip_address"`
	Port      types.Int64  `tfsdk:"port"`
}

type HealthCheckModel struct {
	ID                      types.String `tfsdk:"id"`
	Description             types.String `tfsdk:"description"`
	HealthyStatusCode       types.Int64  `tfsdk:"healthy_status_code"`
	HealthyThresholdCount   types.Int64  `tfsdk:"healthy_threshold_count"`
	InitialDelaySeconds     types.Int64  `tfsdk:"initial_delay_seconds"`
	IntervalSeconds         types.Int64  `tfsdk:"interval_seconds"`
	Name                    types.String `tfsdk:"name"`
	Path                    types.String `tfsdk:"path"`
	Port                    types.Int64  `tfsdk:"port"`
	Protocol                types.String `tfsdk:"protocol"`
	TimeoutSeconds          types.Int64  `tfsdk:"timeout_seconds"`
	UnhealthyThresholdCount types.Int64  `tfsdk:"unhealthy_threshold_count"`
}

type ListenerModel struct {
	ID                 types.String `tfsdk:"id"`
	BackendName        types.String `tfsdk:"backend_name"`
	Description        types.String `tfsdk:"description"`
	Name               types.String `tfsdk:"name"`
	Port               types.Int64  `tfsdk:"port"`
	Protocol           types.String `tfsdk:"protocol"`
	TLSCertificateName types.String `tfsdk:"tls_certificate_name"`
}

type TLSCertificateModel struct {
	Certificate    types.String `tfsdk:"certificate"`
	Description    types.String `tfsdk:"description"`
	Name           types.String `tfsdk:"name"`
	PrivateKey     types.String `tfsdk:"private_key"`
	ID             types.String `tfsdk:"id"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
}

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
	resp.TypeName = req.ProviderTypeName + "_lbaas_networklbs"
}

func (r *LoadBalancerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
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
				Description: "The ID of the public IP associated with the load balancer.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subnetpool_id": schema.StringAttribute{
				Description: "The ID of the subnet pool for the load balancer.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The type of the load balancer.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("application", "network"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"visibility": schema.StringAttribute{
				Description: "The visibility of the load balancer.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("internal", "external"),
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
							Description: "The action for the ACL rule.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("allow", "deny"),
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
						},
						"name": schema.StringAttribute{
							Description: "The name of the backend.",
							Required:    true,
						},
						"panic_threshold": schema.Int64Attribute{
							Description: "The panic threshold percentage for the backend.",
							Optional:    true,
							Validators: []validator.Int64{
								int64validator.Between(0, 100),
							},
						},
						"targets": schema.ListNestedAttribute{
							Description: "The targets for this backend.",
							Required:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Description: "The unique identifier of the target.",
										Computed:    true,
									},
									"nic_id": schema.StringAttribute{
										Description: "The NIC ID of the target.",
										Optional:    true,
									},
									"ip_address": schema.StringAttribute{
										Description: "The IP address of the target.",
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
							Description: "The type of targets.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("instance"),
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
							Description: "The protocol for health checks.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("HTTP", "HTTPS", "TCP", "UDP"),
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
							Description: "The protocol for the listener.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("HTTP", "HTTPS", "TCP", "UDP"),
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
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.ID = types.StringValue(createdLB)
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
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
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
		_, err := r.lbNetworkLB.Update(ctx, planData.ID.ValueString(), lbSDK.UpdateNetworkLoadBalancerRequest{
			Description: planData.Description.ValueStringPointer(),
			Name:        planData.Name.ValueStringPointer(),
		})
		if err != nil {
			resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, planData)...)
}

func (r *LoadBalancerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LoadBalancerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.lbNetworkLB.Delete(ctx, data.ID.ValueString(), lbSDK.DeleteNetworkLoadBalancerRequest{
		// DeletePublicIP: *bool,
	})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
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
			HealthyStatusCode:       tfutil.ConvertInt64PointerToIntPointer(healthCheck.HealthyStatusCode.ValueInt64Pointer()),
			IntervalSeconds:         tfutil.ConvertInt64PointerToIntPointer(healthCheck.IntervalSeconds.ValueInt64Pointer()),
			TimeoutSeconds:          tfutil.ConvertInt64PointerToIntPointer(healthCheck.TimeoutSeconds.ValueInt64Pointer()),
			InitialDelaySeconds:     tfutil.ConvertInt64PointerToIntPointer(healthCheck.InitialDelaySeconds.ValueInt64Pointer()),
			HealthyThresholdCount:   tfutil.ConvertInt64PointerToIntPointer(healthCheck.HealthyThresholdCount.ValueInt64Pointer()),
			UnhealthyThresholdCount: tfutil.ConvertInt64PointerToIntPointer(healthCheck.UnhealthyThresholdCount.ValueInt64Pointer()),
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

		var healthCheckName string
		if backend.HealthCheckID != nil {
			healthCheckName = healthCheckIDsNames[*backend.HealthCheckID]
		}

		backendModels = append(backendModels, BackendModel{
			ID:                                  types.StringValue(backend.ID),
			Name:                                types.StringValue(backend.Name),
			Description:                         types.StringPointerValue(backend.Description),
			BalanceAlgorithm:                    types.StringValue(string(backend.BalanceAlgorithm)),
			HealthCheckName:                     types.StringValue(healthCheckName),
			PanicThreshold:                      types.Float64PointerValue(backend.PanicThreshold),
			CloseConnectionsOnHostHealthFailure: types.BoolValue(backend.CloseConnectionsOnHostHealthFailure),
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
		var tlsCertificateName string
		if listener.TLSCertificateID != nil {
			tlsCertificateName = tlsCertificatesIDsNames[*listener.TLSCertificateID]
		}
		listenerModels = append(listenerModels, ListenerModel{
			ID:                 types.StringValue(listener.ID),
			Name:               types.StringValue(listener.Name),
			Description:        types.StringPointerValue(listener.Description),
			Port:               types.Int64Value(int64(listener.Port)),
			Protocol:           types.StringValue(string(listener.Protocol)),
			BackendName:        types.StringValue(backendIDsNames[listener.BackendID]),
			TLSCertificateName: types.StringValue(tlsCertificateName),
		})
	}

	var publicIPID string
	if len(lb.PublicIPs) == 1 {
		publicIPID = lb.PublicIPs[0].ID
	}
	loadBalancer := LoadBalancerModel{
		ID:              types.StringValue(lb.ID),
		Name:            types.StringValue(lb.Name),
		Description:     types.StringPointerValue(lb.Description),
		PublicIPID:      types.StringValue(publicIPID),
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

	return loadBalancer
}
