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
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	ClusterPoolingTimeout = 100 * time.Minute
)

type KubernetesClusterCreateResourceModel struct {
	Name               types.String   `tfsdk:"name"`
	AllowedCidrs       []types.String `tfsdk:"allowed_cidrs"`
	Description        types.String   `tfsdk:"description"`
	EnabledServerGroup types.Bool     `tfsdk:"enabled_server_group"`
	Version            types.String   `tfsdk:"version"`
	CreatedAt          types.String   `tfsdk:"created_at"`
	ID                 types.String   `tfsdk:"id"`
	ServicesIpV4CIDR   types.String   `tfsdk:"services_ipv4_cidr"`
	ClusterIPv4CIDR    types.String   `tfsdk:"cluster_ipv4_cidr"`
	CNI                types.String   `tfsdk:"cni"`
}

type k8sClusterResource struct {
	k8sCluster k8sSDK.ClusterService
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

	r.k8sCluster = k8sSDK.New(&dataConfig.CoreConfig).Clusters()
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
			"allowed_cidrs": schema.ListAttribute{
				Description: "List of allowed CIDR blocks for API server access.",
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description: "A brief description of the Kubernetes cluster.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled_server_group": schema.BoolAttribute{
				Description: "Enables the use of a server group with anti-affinity policy during the creation of the cluster and its node pools. Default is true.",
				Optional:    true,
				WriteOnly:   true,
			},
			"version": schema.StringAttribute{
				Description: "The native Kubernetes version of the cluster. Use the standard \"vX.Y.Z\" format.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Creation date of the Kubernetes cluster.",
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
			"cni": schema.StringAttribute{
				Description: "The CNI (Container Network Interface) used by the Kubernetes cluster.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					utils.ReplaceIfChangeAndNotIsNotSetOnPlan{},
					stringplanmodifier.UseStateForUnknown(),
				},
			},
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
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	out := convertSDKCreateResultToTerraformCreateClsuterModel(cluster)
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}

func (r *k8sClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data KubernetesClusterCreateResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.EnabledServerGroup.IsNull() {
		data.EnabledServerGroup = types.BoolValue(true)
	}

	cluster, err := r.k8sCluster.Create(ctx, k8sSDK.ClusterRequest{
		AllowedCIDRs:       createAllowedCidrs(data.AllowedCidrs),
		Description:        data.Description.ValueStringPointer(),
		Name:               data.Name.ValueString(),
		Version:            data.Version.ValueStringPointer(),
		EnabledServerGroup: data.EnabledServerGroup.ValueBoolPointer(),
		ClusterIPv4CIDR:    data.ClusterIPv4CIDR.ValueStringPointer(),
		ServicesIpV4CIDR:   data.ServicesIpV4CIDR.ValueStringPointer(),
		CNI:                data.CNI.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	createdCluster, err := r.GetClusterPooling(ctx, cluster.ID, "running", "provisioned")
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		data.ID = types.StringValue(cluster.ID)
		resp.State.Set(ctx, &data)
		return
	}

	newState := convertSDKCreateResultToTerraformCreateClsuterModel(&createdCluster)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *k8sClusterResource) GetClusterPooling(ctx context.Context, clusterId string, states ...string) (k8sSDK.Cluster, error) {
	var result *k8sSDK.Cluster
	var err error
	for startTime := time.Now(); time.Since(startTime) < ClusterPoolingTimeout; {
		time.Sleep(1 * time.Minute)
		result, err = r.k8sCluster.Get(ctx, clusterId)
		if err != nil {
			return k8sSDK.Cluster{}, err
		}
		state := strings.ToLower(result.Status.State)

		if slices.Contains(states, state) {
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
	var data KubernetesClusterCreateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state KubernetesClusterCreateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.AllowedCidrs = data.AllowedCidrs

	cidrs := []string{}
	for _, c := range state.AllowedCidrs {
		cidrs = append(cidrs, c.ValueString())
	}

	_, err := r.k8sCluster.Update(ctx, state.ID.ValueString(), k8sSDK.AllowedCIDRsUpdateRequest{
		AllowedCIDRs: cidrs,
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *k8sClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KubernetesClusterCreateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.k8sCluster.Delete(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	if _, err := r.GetClusterPooling(ctx, data.ID.ValueString(), "deleted"); err != nil {
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

	resp.Diagnostics.Append(resp.State.Set(ctx, &KubernetesClusterCreateResourceModel{
		ID:                 types.StringValue(req.ID),
		EnabledServerGroup: types.BoolValue(true),
	})...)
}

func createAllowedCidrs(data []types.String) *[]string {
	var allowedCidrs []string
	for _, c := range data {
		allowedCidrs = append(allowedCidrs, c.ValueString())
	}
	if len(allowedCidrs) == 0 {
		return nil
	}

	return &allowedCidrs
}

func convertSDKCreateResultToTerraformCreateClsuterModel(sdkResult *k8sSDK.Cluster) *KubernetesClusterCreateResourceModel {
	if sdkResult == nil {
		return nil
	}

	tfModel := &KubernetesClusterCreateResourceModel{
		Name:             types.StringValue(sdkResult.Name),
		ID:               types.StringValue(sdkResult.ID),
		Version:          types.StringValue(sdkResult.Version),
		CreatedAt:        types.StringPointerValue(utils.ConvertTimeToRFC3339(sdkResult.CreatedAt)),
		ServicesIpV4CIDR: types.StringPointerValue(sdkResult.ServicesIpV4CIDR),
		ClusterIPv4CIDR:  types.StringPointerValue(sdkResult.ClusterIPv4CIDR),
		CNI:              types.StringPointerValue(sdkResult.CNI),
	}

	if sdkResult.Description != nil {
		if *sdkResult.Description == "" {
			sdkResult.Description = nil
		} else {
			tfModel.Description = types.StringValue(*sdkResult.Description)
		}
	}

	if sdkResult.AllowedCIDRs != nil {
		if len(*sdkResult.AllowedCIDRs) == 0 {
			sdkResult.AllowedCIDRs = nil
		} else {
			tfModel.AllowedCidrs = convertStringSliceToTypesStringSlice(*sdkResult.AllowedCIDRs)
		}
	}

	return tfModel
}

func convertStringSliceToTypesStringSlice(input []string) []types.String {
	result := make([]types.String, len(input))
	for i, v := range input {
		result[i] = types.StringValue(v)
	}
	return result
}
