package kubernetes

import (
	"context"

	k8sSDK "github.com/MagaluCloud/mgc-sdk-go/kubernetes"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NodePool struct {
	Flavor            types.String    `tfsdk:"flavor_name"`
	Name              types.String    `tfsdk:"name"`
	Replicas          types.Int64     `tfsdk:"replicas"`
	MaxReplicas       types.Int64     `tfsdk:"max_replicas"`
	MinReplicas       types.Int64     `tfsdk:"min_replicas"`
	CreatedAt         types.String    `tfsdk:"created_at"`
	UpdatedAt         types.String    `tfsdk:"updated_at"`
	ID                types.String    `tfsdk:"id"`
	Labels            types.Map       `tfsdk:"labels"`
	SecurityGroups    types.Set       `tfsdk:"security_groups"`
	Taints            *[]Taint        `tfsdk:"taints"`
	MaxPodsPerNode    types.Int64     `tfsdk:"max_pods_per_node"`
	AvailabilityZones *[]types.String `tfsdk:"availability_zones"`
}

type Taint struct {
	Effect types.String `tfsdk:"effect"`
	Key    types.String `tfsdk:"key"`
	Value  types.String `tfsdk:"value"`
}

func ConvertToNodePoolToTFModel(np *k8sSDK.NodePool, region string) NodePool {
	if np == nil {
		return NodePool{}
	}

	nodePool := NodePool{
		Name:      types.StringValue(np.Name),
		Replicas:  types.Int64Value(int64(np.Replicas)),
		CreatedAt: types.StringPointerValue(utils.ConvertTimeToRFC3339(np.CreatedAt)),
		UpdatedAt: types.StringPointerValue(utils.ConvertTimeToRFC3339(np.UpdatedAt)),
		ID:        types.StringValue(np.ID),
	}

	if np.AvailabilityZones != nil {
		availabilityZones := make([]types.String, len(*np.AvailabilityZones))
		for i, zone := range *np.AvailabilityZones {
			availabilityZones[i] = types.StringValue(utils.ConvertXZoneToAvailabilityZone(region, zone))
		}
		nodePool.AvailabilityZones = &availabilityZones
	}

	if np.MaxPodsPerNode != nil {
		nodePool.MaxPodsPerNode = types.Int64Value(int64(*np.MaxPodsPerNode))
	}

	if np.AutoScale != nil {
		if np.AutoScale.MaxReplicas != nil {
			nodePool.MaxReplicas = types.Int64Value(int64(*np.AutoScale.MaxReplicas))
		}

		if np.AutoScale.MinReplicas != nil {
			nodePool.MinReplicas = types.Int64Value(int64(*np.AutoScale.MinReplicas))
		}
	}

	if np.InstanceTemplate.Flavor.Name != "" {
		nodePool.Flavor = types.StringValue(np.InstanceTemplate.Flavor.Name)
	}

	if np.Labels != nil {
		labelsMap, _ := types.MapValueFrom(context.Background(), types.StringType, np.Labels)
		nodePool.Labels = labelsMap
	}

	if np.SecurityGroups != nil {
		securityGroupsSet, _ := types.SetValueFrom(context.Background(), types.StringType, *np.SecurityGroups)
		nodePool.SecurityGroups = securityGroupsSet
	}

	if np.Taints != nil {
		taints := make([]Taint, len(*np.Taints))
		for i, taint := range *np.Taints {
			taints[i] = Taint{
				Effect: types.StringValue(taint.Effect),
				Key:    types.StringValue(taint.Key),
				Value:  types.StringValue(taint.Value),
			}
		}
		nodePool.Taints = &taints
	}

	return nodePool
}
