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
	"github.com/hashicorp/terraform-plugin-log/tflog"

	lbSDK "github.com/MagaluCloud/mgc-sdk-go/lbaas"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
)

const LoadBalancerTimeout = 10 * time.Minute

// LoadBalancerModel representa o modelo de dados do load balancer
type LoadBalancerModel struct {
	ID              types.String `tfsdk:"id"`
	Description     types.String `tfsdk:"description"`
	PanicThreshold  types.Int64  `tfsdk:"panic_threshold"`
	PublicIPID      types.String `tfsdk:"public_ip_id"`
	SubnetpoolID    types.String `tfsdk:"subnetpool_id"`
	Type            types.String `tfsdk:"type"`
	Visibility      types.String `tfsdk:"visibility"`
	VPCID           types.String `tfsdk:"vpc_id"`
	ACLs            types.List   `tfsdk:"acls"`
	Backends        types.List   `tfsdk:"backends"`
	HealthChecks    types.List   `tfsdk:"health_checks"`
	Listeners       types.List   `tfsdk:"listeners"`
	TLSCertificates types.List   `tfsdk:"tls_certificates"`
}

// ACLModel representa o modelo de dados para ACLs
type ACLModel struct {
	Action         types.String `tfsdk:"action"`
	Ethertype      types.String `tfsdk:"ethertype"`
	Name           types.String `tfsdk:"name"`
	Protocol       types.String `tfsdk:"protocol"`
	RemoteIPPrefix types.String `tfsdk:"remote_ip_prefix"`
}

// BackendModel representa o modelo de dados para backends
type BackendModel struct {
	BalanceAlgorithm types.String `tfsdk:"balance_algorithm"`
	Description      types.String `tfsdk:"description"`
	HealthCheckName  types.String `tfsdk:"health_check_name"`
	Name             types.String `tfsdk:"name"`
	Targets          types.List   `tfsdk:"targets"`
	TargetsType      types.String `tfsdk:"targets_type"`
}

// TargetModel representa o modelo de dados para targets
type TargetModel struct {
	NICID     types.String `tfsdk:"nic_id"`
	IPAddress types.String `tfsdk:"ip_address"`
	Port      types.Int64  `tfsdk:"port"`
}

// HealthCheckModel representa o modelo de dados para health checks
type HealthCheckModel struct {
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

// ListenerModel representa o modelo de dados para listeners
type ListenerModel struct {
	BackendName        types.String `tfsdk:"backend_name"`
	Description        types.String `tfsdk:"description"`
	Name               types.String `tfsdk:"name"`
	Port               types.Int64  `tfsdk:"port"`
	Protocol           types.String `tfsdk:"protocol"`
	TLSCertificateName types.String `tfsdk:"tls_certificate_name"`
}

// TLSCertificateModel representa o modelo de dados para certificados TLS
type TLSCertificateModel struct {
	Certificate types.String `tfsdk:"certificate"`
	Description types.String `tfsdk:"description"`
	Name        types.String `tfsdk:"name"`
	PrivateKey  types.String `tfsdk:"private_key"`
}

// LoadBalancerResource representa o recurso do load balancer
type LoadBalancerResource struct {
	lbNetworkBackend        lbSDK.NetworkBackendService
	lbNetworkACL            lbSDK.NetworkACLService
	lbNetworkHealthCheck    lbSDK.NetworkHealthCheckService
	lbNetworkListener       lbSDK.NetworkListenerService
	lbNetworkTLSCertificate lbSDK.NetworkCertificateService
	lbNetworkTarget         lbSDK.NetworkBackendTargetService
}

func NewLoadBalancerResource() resource.Resource {
	return &LoadBalancerResource{}
}

func (r *LoadBalancerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_loadbalancers_network_loadbalancers"
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
	r.lbNetworkTarget = lbaasClient.NetworkBackends().Targets()
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
			"description": schema.StringAttribute{
				Description: "The description of the load balancer.",
				Optional:    true,
			},
			"panic_threshold": schema.Int64Attribute{
				Description: "The panic threshold percentage for the load balancer.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 100),
				},
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
					stringvalidator.OneOf("public", "private"),
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
						"balance_algorithm": schema.StringAttribute{
							Description: "The load balancing algorithm.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("round_robin"), //, "least_connections", "source_ip"
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
						"name": schema.StringAttribute{
							Description: "The name of the backend.",
							Required:    true,
						},
						"targets": schema.ListNestedAttribute{
							Description: "The targets for this backend.",
							Required:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
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
								stringvalidator.OneOf("nic", "ip"),
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
						"certificate": schema.StringAttribute{
							Description: "The TLS certificate content.",
							Required:    true,
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

	tflog.Info(ctx, "Creating load balancer", map[string]interface{}{
		"description": data.Description.ValueString(),
		"type":        data.Type.ValueString(),
		"vpc_id":      data.VPCID.ValueString(),
	})

	// TODO: Implementar a criação do load balancer usando o SDK
	// createdLB, err := r.loadBalancerService.Create(ctx, loadBalancerSDK.CreateLoadBalancerRequest{
	//     Description:     data.Description.ValueStringPointer(),
	//     PanicThreshold: data.PanicThreshold.ValueInt64Pointer(),
	//     PublicIPID:     data.PublicIPID.ValueString(),
	//     SubnetpoolID:   data.SubnetpoolID.ValueString(),
	//     Type:           data.Type.ValueString(),
	//     Visibility:     data.Visibility.ValueString(),
	//     VPCID:          data.VPCID.ValueString(),
	//     ACLs:           r.convertACLsToSDK(data.ACLs),
	//     Backends:       r.convertBackendsToSDK(data.Backends),
	//     HealthChecks:   r.convertHealthChecksToSDK(data.HealthChecks),
	//     Listeners:      r.convertListenersToSDK(data.Listeners),
	//     TLSCertificates: r.convertTLSCertificatesToSDK(data.TLSCertificates),
	// })

	// if err != nil {
	//     resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
	//     return
	// }

	// data.ID = types.StringValue(createdLB.ID)
	// resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *LoadBalancerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LoadBalancerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Implementar a leitura do load balancer usando o SDK
	// lb, err := r.loadBalancerService.Get(ctx, data.ID.ValueString())
	// if err != nil {
	//     resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
	//     return
	// }

	// data = r.toTerraformModel(ctx, lb)
	// resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *LoadBalancerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data LoadBalancerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Implementar a atualização do load balancer usando o SDK
	// _, err := r.loadBalancerService.Update(ctx, data.ID.ValueString(), loadBalancerSDK.UpdateLoadBalancerRequest{
	//     Description:     data.Description.ValueStringPointer(),
	//     PanicThreshold: data.PanicThreshold.ValueInt64Pointer(),
	//     ACLs:           r.convertACLsToSDK(data.ACLs),
	//     Backends:       r.convertBackendsToSDK(data.Backends),
	//     HealthChecks:   r.convertHealthChecksToSDK(data.HealthChecks),
	//     Listeners:      r.convertListenersToSDK(data.Listeners),
	//     TLSCertificates: r.convertTLSCertificatesToSDK(data.TLSCertificates),
	// })

	// if err != nil {
	//     resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
	//     return
	// }

	// resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *LoadBalancerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LoadBalancerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Implementar a exclusão do load balancer usando o SDK
	// err := r.loadBalancerService.Delete(ctx, data.ID.ValueString())
	// if err != nil {
	//     resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
	//     return
	// }
}

func (r *LoadBalancerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// TODO: Implementar métodos auxiliares para conversão de dados
// func (r *LoadBalancerResource) convertACLsToSDK(acls types.List) []loadBalancerSDK.ACL {
//     // Implementar conversão
// }

// func (r *LoadBalancerResource) convertBackendsToSDK(backends types.List) []loadBalancerSDK.Backend {
//     // Implementar conversão
// }

// func (r *LoadBalancerResource) convertHealthChecksToSDK(healthChecks types.List) []loadBalancerSDK.HealthCheck {
//     // Implementar conversão
// }

// func (r *LoadBalancerResource) convertListenersToSDK(listeners types.List) []loadBalancerSDK.Listener {
//     // Implementar conversão
// }

// func (r *LoadBalancerResource) convertTLSCertificatesToSDK(certificates types.List) []loadBalancerSDK.TLSCertificate {
//     // Implementar conversão
// }

// func (r *LoadBalancerResource) toTerraformModel(ctx context.Context, lb *loadBalancerSDK.LoadBalancer) LoadBalancerModel {
//     // Implementar conversão
// }
