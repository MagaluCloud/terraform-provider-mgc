package kubernetes

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	k8sSDK "github.com/MagaluCloud/mgc-sdk-go/kubernetes"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	NodepoolRunningState = "Running"
	NodepoolDeletedState = "Deleted"
)

var (
	NodepoolTimeout  = time.Minute * 90
	NodepoolInterval = time.Second * 30
)

type NodePoolResourceModel struct {
	ClusterID types.String `tfsdk:"cluster_id"`
	NodePool
}

type NewNodePoolResource struct {
	sdkNodepool k8sSDK.NodePoolService
	region      string
}

func NewNewNodePoolResource() resource.Resource {
	return &NewNodePoolResource{}
}

func (r *NewNodePoolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_nodepool"
}

func (r *NewNodePoolResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.region = dataConfig.Region
	r.sdkNodepool = k8sSDK.New(&dataConfig.CoreConfig).Nodepools()
}

func (r *NewNodePoolResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	azRegex := regexp.MustCompile(`^[a-z]{2}-[a-z]+[0-9]+-[a-z]$`)
	resp.Schema = schema.Schema{
		Description: "An array representing a set of nodes within a Kubernetes cluster.",
		Attributes: map[string]schema.Attribute{
			"flavor_name": schema.StringAttribute{
				Description: "Definition of the CPU, RAM, and storage capacity of the nodes.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cluster_id": schema.StringAttribute{
				Description: "UUID of the Kubernetes cluster.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the node pool.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"replicas": schema.Int64Attribute{
				Description: "Number of replicas of the nodes in the node pool.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},

			"labels": schema.MapAttribute{
				Description: "Map of labels for the node pool.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"security_groups": schema.SetAttribute{
				Description: "List of security groups for the node pool.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"max_replicas": schema.Int64Attribute{
				Description: "Maximum number of replicas for autoscaling.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
			"min_replicas": schema.Int64Attribute{
				Description: "Minimum number of replicas for autoscaling.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Date of creation of the Kubernetes Node.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Date of the last change to the Kubernetes Node.",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "Node pool's UUID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"max_pods_per_node": schema.Int64Attribute{
				Description: "Maximum number of pods per node.",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
			"availability_zones": schema.SetAttribute{
				Description: "List of availability zones where the node pool is deployed.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplaceIf(func(ctx context.Context, req planmodifier.SetRequest, resp *setplanmodifier.RequiresReplaceIfFuncResponse) {
						var oldSet, newSet []types.String
						req.StateValue.ElementsAs(ctx, &oldSet, false)
						req.PlanValue.ElementsAs(ctx, &newSet, false)

						if len(oldSet) != len(newSet) {
							resp.RequiresReplace = true
							return
						}
						for i := range oldSet {
							if oldSet[i].ValueString() != newSet[i].ValueString() {
								resp.RequiresReplace = true
								return
							}
						}
						resp.RequiresReplace = false
					}, "availability_zones changed", "availability_zones changed"),
				},
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.RegexMatches(azRegex, "The availability zone must be in the format 'country-region-availability', example 'br-se1-a'")),
				},
				ElementType: types.StringType,
			},
			"taints": schema.ListNestedAttribute{
				Description: "Property associating a set of nodes.",
				Optional:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"effect": schema.StringAttribute{
							Description: "The effect of the taint on pods that do not tolerate the taint.",
							Optional:    true,
						},
						"key": schema.StringAttribute{
							Description: "Key of the taint to be applied to the node.",
							Optional:    true,
						},
						"value": schema.StringAttribute{
							Description: "Value corresponding to the taint key.",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func (r *NewNodePoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NodePoolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nodepool, err := r.sdkNodepool.Get(ctx, data.ClusterID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.NodePool = ConvertToNodePoolToTFModel(nodepool, r.region)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *NewNodePoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NodePoolResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createParams := k8sSDK.CreateNodePoolRequest{
		Flavor:         data.Flavor.ValueString(),
		Name:           data.Name.ValueString(),
		Replicas:       int(data.Replicas.ValueInt64()),
		Taints:         convertTaintsNP(data.Taints),
		MaxPodsPerNode: utils.ConvertInt64PointerToIntPointer(data.MaxPodsPerNode.ValueInt64Pointer()),
	}

	if !data.MaxReplicas.IsNull() || !data.MinReplicas.IsNull() {
		createParams.AutoScale = &k8sSDK.AutoScale{}
	}

	if !data.MaxReplicas.IsNull() {
		createParams.AutoScale.MaxReplicas = utils.ConvertInt64PointerToIntPointer(data.MaxReplicas.ValueInt64Pointer())
	}

	if !data.MinReplicas.IsNull() {
		createParams.AutoScale.MinReplicas = utils.ConvertInt64PointerToIntPointer(data.MinReplicas.ValueInt64Pointer())
	}

	if !data.AvailabilityZones.IsNull() && !data.AvailabilityZones.IsUnknown() {
		azList, err := convertStringSetTFToSliceString(ctx, data.AvailabilityZones)
		if err != nil {
			resp.Diagnostics.AddError("Error converting availability zones", err.Error())
			return
		}
		if azList != nil {
			var azListConverted []string
			for _, az := range *azList {
				converted, err := utils.ConvertAvailabilityZoneToXZone(az)
				if err != nil {
					resp.Diagnostics.AddError("Error converting availability zone", err.Error())
					return
				}
				azListConverted = append(azListConverted, converted)
			}
			createParams.AvailabilityZones = &azListConverted
		}
	}

	nodepool, err := r.sdkNodepool.Create(ctx, data.ClusterID.ValueString(), createParams)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.NodePool = ConvertToNodePoolToTFModel(nodepool, r.region)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err = r.waitNodePoolState(ctx, nodepool.ID, data.ClusterID.ValueString(), NodepoolRunningState, NodepoolTimeout, NodepoolInterval)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
}

func (r *NewNodePoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data NodePoolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state NodePoolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repli := int(data.Replicas.ValueInt64())
	updateParam := k8sSDK.PatchNodePoolRequest{
		Replicas: &repli,
	}

	if !data.MaxReplicas.IsUnknown() || !data.MinReplicas.IsUnknown() {
		updateParam.AutoScale = &k8sSDK.AutoScale{}
	}

	if !data.MaxReplicas.IsUnknown() {
		updateParam.AutoScale.MaxReplicas = utils.ConvertInt64PointerToIntPointer(data.MaxReplicas.ValueInt64Pointer())
	}
	if !data.MinReplicas.IsUnknown() {
		updateParam.AutoScale.MinReplicas = utils.ConvertInt64PointerToIntPointer(data.MinReplicas.ValueInt64Pointer())
	}

	nodepool, err := r.sdkNodepool.Update(ctx, data.ClusterID.ValueString(), data.ID.ValueString(), updateParam)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
	data.NodePool = ConvertToNodePoolToTFModel(nodepool, r.region)

	err = r.waitNodePoolState(ctx, data.ID.ValueString(), data.ClusterID.ValueString(), NodepoolRunningState, NodepoolTimeout, NodepoolInterval)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NewNodePoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NodePoolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.sdkNodepool.Delete(ctx, data.ClusterID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	if err := r.waitNodePoolState(ctx, data.ID.ValueString(), data.ClusterID.ValueString(), NodepoolDeletedState, NodepoolTimeout, NodepoolInterval); err != nil {
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

func (r *NewNodePoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ids := strings.Split(req.ID, ",")
	if len(ids) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: cluster_id,node_pool_id")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &NodePoolResourceModel{
		ClusterID: types.StringValue(ids[0]),
		NodePool: NodePool{
			ID: types.StringValue(ids[1]),
		},
	})...)
}

func convertTaintsNP(taints *[]Taint) *[]k8sSDK.Taint {
	if taints == nil {
		return nil
	}
	taintsNP := make([]k8sSDK.Taint, len(*taints))
	for i, taint := range *taints {
		taintsNP[i] = k8sSDK.Taint{
			Effect: taint.Effect.ValueString(),
			Key:    taint.Key.ValueString(),
			Value:  taint.Value.ValueString(),
		}
	}
	return &taintsNP
}

func convertStringArrayTFToSliceString(tags *[]types.String) *[]string {
	if tags == nil {
		return nil
	}
	tagsSlice := make([]string, len(*tags))
	for i, tag := range *tags {
		tagsSlice[i] = tag.ValueString()
	}
	return &tagsSlice
}

func convertStringSetTFToSliceString(ctx context.Context, set types.Set) (*[]string, error) {
	if set.IsNull() || set.IsUnknown() {
		return nil, nil
	}

	var elements []types.String
	diags := set.ElementsAs(ctx, &elements, false)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to convert set elements: %v", diags.Errors())
	}

	if len(elements) == 0 {
		return &[]string{}, nil
	}

	result := make([]string, len(elements))
	for i, element := range elements {
		result[i] = element.ValueString()
	}
	return &result, nil
}

func (r *NewNodePoolResource) waitNodePoolState(ctx context.Context, nodepoolid, clusterId, state string, timeout, interval time.Duration) error {
	for startTime := time.Now(); time.Since(startTime) < timeout; {
		time.Sleep(interval)

		nodepool, err := r.sdkNodepool.Get(ctx, clusterId, nodepoolid)
		if err != nil {
			return err
		}
		if nodepool.Status.State == state {
			return nil
		}

		tflog.Debug(ctx, fmt.Sprintf("Node pool %s is in state %s", nodepoolid, nodepool.Status.State))
	}
	return fmt.Errorf("timeout waiting for node pool creation")
}
