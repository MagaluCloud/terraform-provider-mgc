package tfutil

import (
	k8sSDK "github.com/MagaluCloud/mgc-sdk-go/kubernetes"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NodePool struct {
	Flavor      types.String    `tfsdk:"flavor_name"`
	Name        types.String    `tfsdk:"name"`
	Replicas    types.Int64     `tfsdk:"replicas"`
	MaxReplicas types.Int64     `tfsdk:"max_replicas"`
	MinReplicas types.Int64     `tfsdk:"min_replicas"`
	Tags        *[]types.String `tfsdk:"tags"`
	CreatedAt   types.String    `tfsdk:"created_at"`
	UpdatedAt   types.String    `tfsdk:"updated_at"`
	ID          types.String    `tfsdk:"id"`
	Taints      *[]Taint        `tfsdk:"taints"`
}

type Taint struct {
	Effect types.String `tfsdk:"effect"`
	Key    types.String `tfsdk:"key"`
	Value  types.String `tfsdk:"value"`
}

func ConvertToNodePoolToTFModel(np *k8sSDK.NodePool) NodePool {
	if np == nil {
		return NodePool{}
	}

	nodePool := NodePool{
		Name:      types.StringValue(np.Name),
		Replicas:  types.Int64Value(int64(np.Replicas)),
		CreatedAt: types.StringPointerValue(ConvertTimeToRFC3339(np.CreatedAt)),
		UpdatedAt: types.StringPointerValue(ConvertTimeToRFC3339(np.UpdatedAt)),
		ID:        types.StringValue(np.ID),
	}

	if np.AutoScale.MaxReplicas != nil {
		nodePool.MaxReplicas = types.Int64Value(int64(*np.AutoScale.MaxReplicas))
	}

	if np.AutoScale.MinReplicas != nil {
		nodePool.MinReplicas = types.Int64Value(int64(*np.AutoScale.MinReplicas))
	}

	if np.InstanceTemplate.Flavor.Name != "" {
		nodePool.Flavor = types.StringValue(np.InstanceTemplate.Flavor.Name)
	}

	if np.Tags != nil {
		tags := make([]types.String, len(*np.Tags))
		for i, tag := range *np.Tags {
			tags[i] = types.StringValue(tag)
		}
		if len(tags) > 0 {
			nodePool.Tags = &tags
		}
	}

	if np.Taints != nil {
		var taints []Taint
		for _, taint := range *np.Taints {
			taints = append(taints, Taint{
				Effect: types.StringValue(taint.Effect),
				Key:    types.StringValue(taint.Key),
				Value:  types.StringValue(taint.Value),
			})
		}
		nodePool.Taints = &taints
	}

	return nodePool
}
