package lbaas

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
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
		Description: experimentalWarning + "Manages network load balancers in Magalu Cloud.",
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
			"acls": schema.SetNestedAttribute{
				Description: "Access Control Lists for the load balancer.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
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
			"backends": schema.SetNestedAttribute{
				Description: "Backend configurations for the load balancer.",
				Required:    true,
				PlanModifiers: []planmodifier.Set{
					utils.SetMembershipChangeRequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the backend.",
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"balance_algorithm": schema.StringAttribute{
							Description: "The load balancing algorithm.",
							Required:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
							Validators: []validator.String{
								stringvalidator.OneOf("round_robin"),
							},
						},
						"description": schema.StringAttribute{
							Description: "The description of the backend.",
							Optional:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"health_check_name": schema.StringAttribute{
							Description: "The name of the health check associated with this backend.",
							Optional:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"close_connections_on_host_health_failure": schema.BoolAttribute{
							Description: "Whether to close connections when a host health check fails.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
						"name": schema.StringAttribute{
							Description: "The name of the backend.",
							Required:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"panic_threshold": schema.Float64Attribute{
							Description: "The panic threshold percentage for the backend.",
							Optional:    true,
							Computed:    true,
							Validators: []validator.Float64{
								float64validator.Between(0, 100),
							},
						},
						"targets": schema.SetNestedAttribute{
							Description: "The targets for this backend.",
							Required:    true,
							Validators: []validator.Set{
								TargetValidator{},
							},
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
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
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
					},
				},
			},
			"health_checks": schema.SetNestedAttribute{
				Description: "Health check configurations for the load balancer.",
				Optional:    true,
				PlanModifiers: []planmodifier.Set{
					utils.SetMembershipChangeRequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the health check.",
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"description": schema.StringAttribute{
							Description: "The description of the health check.",
							Optional:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
								stringplanmodifier.UseStateForUnknown(),
							},
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
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
								stringplanmodifier.UseStateForUnknown(),
							},
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
			"listeners": schema.SetNestedAttribute{
				Description: "Listener configurations for the load balancer.",
				Required:    true,
				PlanModifiers: []planmodifier.Set{
					utils.SetMembershipChangeRequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the listener.",
							Computed:    true,
						},
						"backend_name": schema.StringAttribute{
							Description: "The name of the backend associated with this listener.",
							Required:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"description": schema.StringAttribute{
							Description: "The description of the listener.",
							Optional:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"name": schema.StringAttribute{
							Description: "The name of the listener.",
							Required:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"port": schema.Int64Attribute{
							Description: "The port for the listener.",
							Required:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.RequiresReplace(),
							},
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
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							}},
						"tls_certificate_name": schema.StringAttribute{
							Description: "The name of the TLS certificate for HTTPS listeners.",
							Optional:    true,
							Validators: []validator.String{
								ListenerTLSValidator{},
							},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
					},
				},
			},
			"tls_certificates": schema.ListNestedAttribute{
				Description: "TLS certificate configurations for the load balancer.",
				Optional:    true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the TLS certificate.",
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"certificate": schema.StringAttribute{
							Description: "The TLS certificate content. Must be base64 encoded.",
							Required:    true,
							WriteOnly:   true,
							Sensitive:   true,
						},
						"description": schema.StringAttribute{
							Description: "The description of the TLS certificate.",
							Optional:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"name": schema.StringAttribute{
							Description: "The name of the TLS certificate.",
							Required:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"private_key": schema.StringAttribute{
							Description: "The private key for the TLS certificate, must be base64 encoded.",
							Required:    true,
							Sensitive:   true,
							WriteOnly:   true,
						},
						"expiration_date": schema.StringAttribute{
							Description: "The expiration date of the TLS certificate.",
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
		},
	}
}

func (r *LoadBalancerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LoadBalancerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
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
		ACLs:            data.ConvertACLsToSDK(),
		Backends:        data.ConvertBackendsToSDK(),
		HealthChecks:    data.ConvertHealthChecksToSDK(),
		Listeners:       data.ConvertListenersToSDK(),
		TLSCertificates: data.ConvertTLSCertificatesToSDK(),
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

	data = data.ToTerraformNetworkResource(ctx, *getLB)
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

	data = data.ToTerraformNetworkResource(ctx, lb)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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

	if err := r.updateLBNameDescription(ctx, &planData, &stateData); err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	if err := r.replaceACLsIfChanged(ctx, &planData, &stateData); err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	if err := r.updateHealthChecks(ctx, &planData, &stateData); err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	if err := r.updateBackendsFields(ctx, &planData, &stateData); err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	if err := r.replaceBackendTargets(ctx, &planData, &stateData); err != nil {
		var nf healthCheckNotFoundError
		if errors.As(err, &nf) {
			resp.Diagnostics.AddError("Health Check Not Found", nf.Error())
			return
		}
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
}

func (r *LoadBalancerResource) updateLBNameDescription(ctx context.Context, plan, state *LoadBalancerModel) error {
	if !plan.Description.Equal(state.Description) || !plan.Name.Equal(state.Name) {
		state.Description = plan.Description
		state.Name = plan.Name
		_, err := r.lbNetworkLB.Update(ctx, plan.ID.ValueString(), lbSDK.UpdateNetworkLoadBalancerRequest{
			Description: plan.Description.ValueStringPointer(),
			Name:        plan.Name.ValueStringPointer(),
		})
		if err != nil {
			return err
		}
		_, err = r.waitLoadBalancerState(ctx, plan.ID.ValueString(), lbSDK.LoadBalancerStatusRunning)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *LoadBalancerResource) replaceACLsIfChanged(ctx context.Context, plan, state *LoadBalancerModel) error {
	if plan.hasACLChanges(*state) {
		state.ACLs = plan.ACLs
		updatedACL := plan.ConvertACLsToSDK()
		if updatedACL == nil {
			updatedACL = []lbSDK.CreateNetworkACLRequest{}
		}
		err := r.lbNetworkACL.Replace(ctx, plan.ID.ValueString(), lbSDK.UpdateNetworkACLRequest{
			Acls: updatedACL,
		})
		if err != nil {
			return err
		}
		_, err = r.waitLoadBalancerState(ctx, state.ID.ValueString(), lbSDK.LoadBalancerStatusRunning)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *LoadBalancerResource) updateHealthChecks(ctx context.Context, plan, state *LoadBalancerModel) error {
	if hasChange, updatedHealthChecks := plan.healthChecksToUpdate(*state); hasChange {
		state.HealthChecks = plan.HealthChecks
		for _, hc := range updatedHealthChecks {
			err := r.lbNetworkHealthCheck.Update(ctx, state.ID.ValueString(), hc.ID.ValueString(), lbSDK.UpdateNetworkHealthCheckRequest{
				Protocol:                lbSDK.HealthCheckProtocol(hc.Protocol.ValueString()),
				Port:                    int(hc.Port.ValueInt64()),
				Path:                    hc.Path.ValueStringPointer(),
				HealthyStatusCode:       utils.ConvertInt64PointerToIntPointer(hc.HealthyStatusCode.ValueInt64Pointer()),
				HealthyThresholdCount:   utils.ConvertInt64PointerToIntPointer(hc.HealthyThresholdCount.ValueInt64Pointer()),
				InitialDelaySeconds:     utils.ConvertInt64PointerToIntPointer(hc.InitialDelaySeconds.ValueInt64Pointer()),
				IntervalSeconds:         utils.ConvertInt64PointerToIntPointer(hc.IntervalSeconds.ValueInt64Pointer()),
				TimeoutSeconds:          utils.ConvertInt64PointerToIntPointer(hc.TimeoutSeconds.ValueInt64Pointer()),
				UnhealthyThresholdCount: utils.ConvertInt64PointerToIntPointer(hc.UnhealthyThresholdCount.ValueInt64Pointer()),
			})
			if err != nil {
				return err
			}
		}
		_, err := r.waitLoadBalancerState(ctx, plan.ID.ValueString(), lbSDK.LoadBalancerStatusRunning)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *LoadBalancerResource) updateBackendsFields(ctx context.Context, plan, state *LoadBalancerModel) error {
	backendFieldUpdates, _ := plan.backendsToUpdate(*state)
	if len(backendFieldUpdates) == 0 {
		return nil
	}
	for _, b := range backendFieldUpdates {
		backendIdx := slices.IndexFunc(state.Backends, func(sb BackendModel) bool {
			return sb.Name.Equal(b.Name)
		})
		if backendIdx != -1 {
			state.Backends[backendIdx].PanicThreshold = b.PanicThreshold
			state.Backends[backendIdx].CloseConnectionsOnHostHealthFailure = b.CloseConnectionsOnHostHealthFailure
		}
		_, err := r.lbNetworkBackend.Update(ctx, state.ID.ValueString(), b.ID.ValueString(), lbSDK.UpdateNetworkBackendRequest{
			PanicThreshold:                      b.PanicThreshold.ValueFloat64Pointer(),
			CloseConnectionsOnHostHealthFailure: b.CloseConnectionsOnHostHealthFailure.ValueBoolPointer(),
		})
		if err != nil {
			return err
		}
	}
	_, err := r.waitLoadBalancerState(ctx, plan.ID.ValueString(), lbSDK.LoadBalancerStatusRunning)
	if err != nil {
		return err
	}
	return nil
}

func (r *LoadBalancerResource) replaceBackendTargets(ctx context.Context, plan, state *LoadBalancerModel) error {
	_, backendTargetUpdates := plan.backendsToUpdate(*state)
	if len(backendTargetUpdates) == 0 {
		return nil
	}

	healthCheckNameToID := make(map[string]string)
	if state.HealthChecks != nil {
		for _, hc := range *state.HealthChecks {
			healthCheckNameToID[hc.Name.ValueString()] = hc.ID.ValueString()
		}
	}

	for _, bu := range backendTargetUpdates {
		var healthCheckID *string
		if !bu.HealthCheckName.IsNull() && !bu.HealthCheckName.IsUnknown() {
			if id, ok := healthCheckNameToID[bu.HealthCheckName.ValueString()]; ok {
				healthCheckID = &id
			} else {
				return healthCheckNotFoundError{backendName: bu.Name.ValueString(), hcName: bu.HealthCheckName.ValueString()}
			}
		}

		backendIdx := slices.IndexFunc(state.Backends, func(sb BackendModel) bool {
			return sb.Name.Equal(bu.Name)
		})
		if backendIdx != -1 {
			state.Backends[backendIdx].Targets = bu.Targets
		}

		var targets []lbSDK.NetworkBackendInstanceTargetRequest
		for _, target := range bu.Targets {
			targets = append(targets, lbSDK.NetworkBackendInstanceTargetRequest{
				NicID:     target.NICID.ValueStringPointer(),
				Port:      target.Port.ValueInt64(),
				IPAddress: target.IPAddress.ValueStringPointer(),
			})
		}

		_, err := r.lbNetworkTarget.Replace(ctx, state.ID.ValueString(), bu.ID.ValueString(), lbSDK.CreateNetworkBackendTargetRequest{
			HealthCheckID: healthCheckID,
			TargetsType:   lbSDK.BackendType(bu.TargetsType.ValueString()),
			Targets:       targets,
		})
		if err != nil {
			return err
		}
	}

	_, err := r.waitLoadBalancerState(ctx, plan.ID.ValueString(), lbSDK.LoadBalancerStatusRunning)
	if err != nil {
		return err
	}
	return nil
}

func (r *LoadBalancerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LoadBalancerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deletePublicIP := false
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
				return nil, fmt.Errorf("load balancer %s is in error state. %s", lbID, *lb.LastOperationStatus)
			}
			if lb.Status == desiredState {
				return &lb, nil
			}
		}
	}
}
