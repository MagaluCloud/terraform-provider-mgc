package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	k8sSDK "github.com/MagaluCloud/mgc-sdk-go/kubernetes"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	defaultClusterPollingTimeout  = 160 * time.Minute
	defaultClusterPollingInterval = 1 * time.Minute
)

type KubernetesClusterCreateResourceModel struct {
	Name               types.String `tfsdk:"name"`
	AllowedCidrs       types.Set    `tfsdk:"allowed_cidrs"`
	Description        types.String `tfsdk:"description"`
	EnabledServerGroup types.Bool   `tfsdk:"enabled_server_group"`
	Version            types.String `tfsdk:"version"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
	ID                 types.String `tfsdk:"id"`
	Region             types.String `tfsdk:"region"`
	ServicesIpV4CIDR   types.String `tfsdk:"services_ipv4_cidr"`
	ClusterIPv4CIDR    types.String `tfsdk:"cluster_ipv4_cidr"`
	MachineTypesSource types.String `tfsdk:"machine_types_source"`
	PlatformVersion    types.String `tfsdk:"platform_version"`
	SubnetIDs          types.Set    `tfsdk:"subnet_ids"`
}

type k8sClusterResource struct {
	k8sCluster      k8sSDK.ClusterService
	pollingInterval time.Duration
	pollingTimeout  time.Duration
}

func NewK8sClusterResource() resource.Resource {
	return &k8sClusterResource{}
}

func (r *k8sClusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_cluster"
}

func (r *k8sClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.k8sCluster = k8sSDK.New(dataConfig.CoreFor(utils.ServiceKubernetes)).Clusters()
	r.pollingInterval = dataConfig.PollingIntervalOr(defaultClusterPollingInterval)
	r.pollingTimeout = dataConfig.PollingTimeoutOr(defaultClusterPollingTimeout)
}

func (r *k8sClusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	nameRule := regexp.MustCompile(`^[a-z]([-a-z0-9]{0,61}[a-z0-9])?$`)
	resp.Schema = schema.Schema{
		Description: "Kubernetes cluster resource in MGC",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Kubernetes cluster name. Must be unique within a namespace and follow naming rules.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtMost(63),
					stringvalidator.RegexMatches(nameRule, "must contain only lowercase alphanumeric characters or '-'"),
				},
			},
			"allowed_cidrs": schema.SetAttribute{
				Description: "List of allowed CIDR blocks for API server access.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"description": schema.StringAttribute{
				Description: "A brief description of the Kubernetes cluster.",
				Optional:    true,
			},
			//deprecated
			"enabled_server_group": schema.BoolAttribute{
				Description:        "[DEPRECATED] Enables the use of a server group with anti-affinity policy during the creation of the cluster and its node pools. Default is true.",
				Optional:           true,
				Computed:           true,
				DeprecationMessage: "This attribute has been marked as obsolete and has no effect.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
					boolplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				Description: "The native Kubernetes version of the cluster. Use the standard \"vX.Y.Z\" format. " +
					"Changing this value upgrades the control plane in place (no replacement); Terraform holds the apply until the cluster returns to a running state on the new version. " +
					"Upgrade the control plane before the node pools",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^v\d+\.\d+\.\d+$`), "must follow the standard \"vX.Y.Z\" format, e.g. v1.31.0"),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Creation date of the Kubernetes cluster.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Last update date of the Kubernetes cluster.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region": schema.StringAttribute{
				Description: "Region where the Kubernetes cluster is located.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				Description: "Cluster's UUID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cluster_ipv4_cidr": schema.StringAttribute{
				Description: "The IP address range of the Kubernetes cluster.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					utils.ReplaceIfChangeAndNotIsNotSetOnPlan{},
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"services_ipv4_cidr": schema.StringAttribute{
				Description: "The IP address range of the Kubernetes cluster service.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					utils.ReplaceIfChangeAndNotIsNotSetOnPlan{},
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"machine_types_source": schema.StringAttribute{
				Description: "Source of machine types for the cluster.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"platform_version": schema.StringAttribute{
				Description: "Platform version of the cluster.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"subnet_ids": ResourceSubnetIDsAttribute(`List of subnet ids. When omitted, the subnets chosen are inherited.
							You must specify exactly one subnet per availability zone.
							The subnets must belong to the same VPC.
							This field cannot be changed after the node pool is created`),
		},
	}
}

func (r *k8sClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KubernetesClusterCreateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cluster, err := r.k8sCluster.Get(ctx, data.ID.ValueString())
	if err != nil {
		var httpErr *clientSDK.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			resp.Diagnostics.AddWarning(
				"MGC Resource not found the cluster during refresh",
				"The cluster has been automatically removed from the state and will be provisioned",
			)
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data = flattenCluster(data, *cluster)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *k8sClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if resp.Diagnostics.HasError() {
		return
	}
	var plan KubernetesClusterCreateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	cluster, err := r.k8sCluster.Create(ctx, k8sSDK.ClusterRequest{
		AllowedCIDRs:       utils.ConvertTypeSetToStringArray(plan.AllowedCidrs),
		Description:        utils.KnownStringPointer(plan.Description),
		Name:               plan.Name.ValueString(),
		Version:            utils.KnownStringPointer(plan.Version),
		EnabledServerGroup: utils.KnownBoolPointer(plan.EnabledServerGroup),
		ClusterIPv4CIDR:    utils.KnownStringPointer(plan.ClusterIPv4CIDR),
		ServicesIpV4CIDR:   utils.KnownStringPointer(plan.ServicesIpV4CIDR),
		Network:            CreateKubernetesSDKNetworkRequest(plan.SubnetIDs),
	})

	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	createdCluster, err := r.GetClusterPooling(ctx, cluster.ID, "", "running", "provisioned")

	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		resp.State.Set(ctx, &plan)
		return
	}
	plan = flattenCluster(plan, createdCluster)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *k8sClusterResource) GetClusterPooling(ctx context.Context, clusterId, expectedVersion string, states ...string) (k8sSDK.Cluster, error) {
	var result *k8sSDK.Cluster
	var err error
	for startTime := time.Now(); time.Since(startTime) < r.pollingTimeout; {
		time.Sleep(r.pollingInterval)
		result, err = r.k8sCluster.Get(ctx, clusterId)
		if err != nil {
			return k8sSDK.Cluster{}, err
		}
		state := strings.ToLower(result.Status.State)

		if slices.Contains(states, state) && (expectedVersion == "" || result.Version == expectedVersion) {
			return *result, nil
		}
		if state == "failed" {
			return *result, errors.New("cluster failed to provision")
		}

		tflog.Debug(ctx, fmt.Sprintf("current cluster state: [%s]", state))
	}

	return *result, errors.New("timeout waiting for cluster to provision")
}

func (r *k8sClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan KubernetesClusterCreateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state KubernetesClusterCreateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	patch := buildPatchClusterRequest(state, plan)

	_, err := r.k8sCluster.Update(ctx, state.ID.ValueString(), patch)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	expectedVersion := ""
	if !plan.Version.IsUnknown() && !plan.Version.IsNull() {
		expectedVersion = plan.Version.ValueString()
	}

	out, err := r.GetClusterPooling(ctx, state.ID.ValueString(), expectedVersion, "running")

	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, flattenCluster(plan, out))...)
}

func (r *k8sClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KubernetesClusterCreateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.k8sCluster.Delete(ctx, data.ID.ValueString())
	if err != nil {
		var httpErr *clientSDK.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			resp.Diagnostics.AddWarning(
				"MGC Resource not found the cluster during delete",
				"The cluster has been removed",
			)
			return
		} else if !(errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusConflict) { //Provides protection if the status is “deleting”
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
	}

	if _, err := r.GetClusterPooling(ctx, data.ID.ValueString(), "", "deleted"); err != nil {
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

func (r *k8sClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import ID", "The ID must be provided")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func buildPatchClusterRequest(state, plan KubernetesClusterCreateResourceModel) k8sSDK.PatchClusterRequest {
	patch := k8sSDK.PatchClusterRequest{}

	if utils.ConvertTypeSetToStringArray(plan.AllowedCidrs) != utils.ConvertTypeSetToStringArray(state.AllowedCidrs) {
		allowedCidrs := utils.ConvertTypeSetToStringArray(plan.AllowedCidrs)
		if allowedCidrs == nil {
			allowedCidrs = &[]string{}
		}
		patch.AllowedCIDRs = allowedCidrs
	}

	if plan.Description.ValueString() != state.Description.ValueString() {
		v := plan.Description.ValueString()
		patch.Description = &v
	}

	if !plan.Version.IsUnknown() && !plan.Version.IsNull() && plan.Version.ValueString() != state.Version.ValueString() {
		version := plan.Version.ValueStringPointer()
		patch.Version = version
	}
	return patch
}

func flattenCluster(tfData KubernetesClusterCreateResourceModel, cluster k8sSDK.Cluster) KubernetesClusterCreateResourceModel {
	tfData.Version = utils.FlattenStringValue(tfData.Version, &cluster.Version)
	tfData.CreatedAt = utils.FlattenStringValue(tfData.CreatedAt, utils.ConvertTimeToRFC3339(cluster.CreatedAt))
	tfData.UpdatedAt = utils.FlattenStringValue(tfData.CreatedAt, utils.ConvertTimeToRFC3339(cluster.UpdatedAt))
	tfData.ID = types.StringValue(cluster.ID)
	tfData.ClusterIPv4CIDR = utils.FlattenStringValue(tfData.ClusterIPv4CIDR, cluster.ClusterIPv4CIDR)
	tfData.ServicesIpV4CIDR = utils.FlattenStringValue(tfData.ServicesIpV4CIDR, cluster.ServicesIpV4CIDR)

	v := string(*cluster.MachineTypesSource)
	tfData.MachineTypesSource = utils.FlattenStringValue(tfData.MachineTypesSource, &v)

	if cluster.Platform != nil {
		tfData.PlatformVersion = utils.FlattenStringValue(tfData.PlatformVersion, &cluster.Platform.Version)
	} else {
		tfData.PlatformVersion = types.StringNull()
	}

	tfData.Region = utils.FlattenStringValue(tfData.Region, cluster.Region)
	tfData.Region = utils.FlattenStringValue(tfData.Region, cluster.Region)
	tfData.SubnetIDs = GetSubnetIDs(cluster.Network)

	if tfData.EnabledServerGroup.IsUnknown() {
		tfData.EnabledServerGroup = types.BoolNull()
	}

	tfData.AllowedCidrs = utils.FlattenTypeSetStringArray(tfData.AllowedCidrs, cluster.AllowedCIDRs)

	tfData.Description = utils.FlattenStringValue(tfData.Description, cluster.Description)
	tfData.Name = utils.FlattenStringValue(tfData.Name, &cluster.Name)

	return tfData
}
