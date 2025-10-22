package kubernetes

import (
	"context"

	k8sSDK "github.com/MagaluCloud/mgc-sdk-go/kubernetes"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NodePool struct {
	Flavor            types.String `tfsdk:"flavor_name"`
	Name              types.String `tfsdk:"name"`
	Replicas          types.Int64  `tfsdk:"replicas"`
	MaxReplicas       types.Int64  `tfsdk:"max_replicas"`
	MinReplicas       types.Int64  `tfsdk:"min_replicas"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
	ID                types.String `tfsdk:"id"`
	Labels            types.Map    `tfsdk:"labels"`
	SecurityGroups    types.Set    `tfsdk:"security_groups"`
	Taints            *[]Taint     `tfsdk:"taints"`
	MaxPodsPerNode    types.Int64  `tfsdk:"max_pods_per_node"`
	AvailabilityZones types.Set    `tfsdk:"availability_zones"`
}

type Taint struct {
	Effect types.String `tfsdk:"effect"`
	Key    types.String `tfsdk:"key"`
	Value  types.String `tfsdk:"value"`
}

func ConvertToNodePoolToTFModel(np *k8sSDK.NodePool, region string) NodePool {
	if np == nil {
		return NodePool{
			Flavor:            types.StringNull(),
			Name:              types.StringNull(),
			Replicas:          types.Int64Null(),
			MaxReplicas:       types.Int64Null(),
			MinReplicas:       types.Int64Null(),
			CreatedAt:         types.StringNull(),
			UpdatedAt:         types.StringNull(),
			ID:                types.StringNull(),
			Labels:            types.MapNull(types.StringType),
			SecurityGroups:    types.SetNull(types.StringType),
			MaxPodsPerNode:    types.Int64Null(),
			AvailabilityZones: types.SetNull(types.StringType),
		}
	}

	nodePool := NodePool{
		Name:      types.StringValue(np.Name),
		Replicas:  types.Int64Value(int64(np.Replicas)),
		CreatedAt: types.StringPointerValue(utils.ConvertTimeToRFC3339(np.CreatedAt)),
		UpdatedAt: types.StringPointerValue(utils.ConvertTimeToRFC3339(np.UpdatedAt)),
		ID:        types.StringValue(np.ID),
	}

	if np.AvailabilityZones != nil {
		availabilityZones := make([]attr.Value, len(*np.AvailabilityZones))
		for i, zone := range *np.AvailabilityZones {
			availabilityZones[i] = types.StringValue(utils.ConvertXZoneToAvailabilityZone(region, zone))
		}
		nodePool.AvailabilityZones = types.SetValueMust(types.StringType, availabilityZones)
	} else {
		nodePool.AvailabilityZones = types.SetNull(types.StringType)
	}

	if np.MaxPodsPerNode != nil {
		nodePool.MaxPodsPerNode = types.Int64Value(int64(*np.MaxPodsPerNode))
	} else {
		nodePool.MaxPodsPerNode = types.Int64Null()
	}

	if np.AutoScale != nil {
		if np.AutoScale.MaxReplicas != nil {
			nodePool.MaxReplicas = types.Int64Value(int64(*np.AutoScale.MaxReplicas))
		} else {
			nodePool.MaxReplicas = types.Int64Null()
		}

		if np.AutoScale.MinReplicas != nil {
			nodePool.MinReplicas = types.Int64Value(int64(*np.AutoScale.MinReplicas))
		} else {
			nodePool.MinReplicas = types.Int64Null()
		}
	} else {
		nodePool.MaxReplicas = types.Int64Null()
		nodePool.MinReplicas = types.Int64Null()
	}

	if np.InstanceTemplate.Flavor.Name != "" {
		nodePool.Flavor = types.StringValue(np.InstanceTemplate.Flavor.Name)
	} else {
		nodePool.Flavor = types.StringNull()
	}

	if np.Labels != nil {
		labelsMap, _ := types.MapValueFrom(context.Background(), types.StringType, np.Labels)
		nodePool.Labels = labelsMap
	} else {
		nodePool.Labels = types.MapNull(types.StringType)
	}

	if np.SecurityGroups != nil {
		securityGroupsSet, _ := types.SetValueFrom(context.Background(), types.StringType, *np.SecurityGroups)
		nodePool.SecurityGroups = securityGroupsSet
	} else {
		nodePool.SecurityGroups = types.SetNull(types.StringType)
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
