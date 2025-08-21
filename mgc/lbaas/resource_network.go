package lbaas

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
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
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
							Computed: true,
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
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
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
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
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
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
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
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
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
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
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

		resp.Diagnostics.Append(r.waitLoadBalancerRunning(ctx, planData.ID.ValueString(), resp.Diagnostics)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if planData.hasACLChanges(stateData) {
		stateData.ACLs = planData.ACLs
		updatedACL := planData.ConvertACLsToSDK()
		if updatedACL == nil {
			updatedACL = []lbSDK.CreateNetworkACLRequest{}
		}

		planData.ConvertACLsToSDK()
		err := r.lbNetworkACL.Replace(ctx, planData.ID.ValueString(), lbSDK.UpdateNetworkACLRequest{
			Acls: updatedACL,
		})
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}

		updated, err := r.waitLoadBalancerState(ctx, stateData.ID.ValueString(), lbSDK.LoadBalancerStatusRunning)
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
		if updated == nil {
			resp.Diagnostics.AddError("Load Balancer not found", fmt.Sprintf("Load Balancer with ID %s not found after ACL update.", stateData.ID.ValueString()))
			return
		}

		stateData.ACLs = stateData.ToTerraformNetworkResource(ctx, *updated).ACLs
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

func (r *LoadBalancerResource) waitLoadBalancerRunning(ctx context.Context, lbID string, diag diag.Diagnostics) diag.Diagnostics {
	_, err := r.waitLoadBalancerState(ctx, lbID, lbSDK.LoadBalancerStatusRunning)
	if err != nil {
		if httpErr, ok := err.(*clientSDK.HTTPError); ok && httpErr.StatusCode == http.StatusNotFound {
			diag.AddWarning("Load Balancer Not Found", fmt.Sprintf("Load balancer with ID %s not found during wait for running state.", lbID))
			return diag
		}
		diag.AddError("Failed to wait for Load Balancer Running", fmt.Sprintf("Error waiting for load balancer %s to reach running state: %s", lbID, err.Error()))
		return diag
	}
	return diag
}
