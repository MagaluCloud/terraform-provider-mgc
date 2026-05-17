package kubernetes

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	sdkK8s "github.com/MagaluCloud/mgc-sdk-go/kubernetes"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func stringPtr(s string) *string { return &s }
func intPtr(i int) *int          { return &i }
func boolPtr(b bool) *bool       { return &b }

func TestConvertToControlplane(t *testing.T) {
	t.Run("should handle minimal sdk input correctly", func(t *testing.T) {
		sdkCP := &sdkK8s.NodePool{
			ID: "cp-id-123",
			InstanceTemplate: sdkK8s.InstanceTemplate{
				DiskSize: 50,
				DiskType: "ssd",
				Flavor: sdkK8s.Flavor{
					ID:   "flavor-id-abc",
					Name: "c1.small",
				},
				NodeImage: "ubuntu-22.04",
			},
			Name:     "controlplane-0",
			Replicas: 1,
			Status:   sdkK8s.Status{State: "Running"},
		}

		result := convertToControlplane(sdkCP)

		assert.NotNil(t, result)
		assert.Equal(t, types.StringValue("cp-id-123"), result.ID)
		assert.Equal(t, types.StringValue("c1.small"), result.FlavorName)
		assert.Equal(t, types.Int64Value(1), result.Replicas)
		assert.Equal(t, types.StringValue("Running"), result.State)

		assert.True(t, result.MaxReplicas.IsNull())
		assert.True(t, result.MinReplicas.IsNull())
		assert.True(t, result.Labels.IsNull())
		assert.Nil(t, result.SecurityGroups)
		assert.Nil(t, result.Taints)
	})

	maxReplicas := 5
	minReplicas := 2
	t.Run("should convert full sdk input correctly", func(t *testing.T) {
		now := time.Now()
		sdkCP := &sdkK8s.NodePool{
			ID: "cp-id-123",
			InstanceTemplate: sdkK8s.InstanceTemplate{
				DiskSize: 50,
				DiskType: "ssd",
				Flavor: sdkK8s.Flavor{
					ID:   "flavor-id-abc",
					Name: "c1.small",
				},
				NodeImage: "ubuntu-22.04",
			},
			Name:      "controlplane-0",
			Replicas:  3,
			CreatedAt: &now,
			UpdatedAt: &now,
			AutoScale: &sdkK8s.AutoScale{
				MaxReplicas: &maxReplicas,
				MinReplicas: &minReplicas,
			},
			SecurityGroups: &[]string{"sg-default"},
			Status: sdkK8s.Status{
				State:    "Running",
				Messages: []string{"ok"},
			},
			Taints: &[]sdkK8s.Taint{
				{Effect: "NoSchedule", Key: "app", Value: "critical"},
			},
		}

		result := convertToControlplane(sdkCP)
		expectedTime := types.StringValue(now.Format(time.RFC3339))

		assert.NotNil(t, result)
		assert.Equal(t, types.StringValue("cp-id-123"), result.ID)
		assert.Equal(t, types.Int64Value(50), result.DiskSize)
		assert.Equal(t, types.Int64Value(3), result.Replicas)
		assert.Equal(t, expectedTime, result.CreatedAt)
		assert.Equal(t, expectedTime, result.UpdatedAt)
		assert.Equal(t, types.Int64Value(5), result.MaxReplicas)
		assert.Equal(t, types.Int64Value(2), result.MinReplicas)
		assert.Equal(t, types.StringValue("sg-default"), result.SecurityGroups[0])
		assert.Equal(t, types.StringValue("ok"), result.StatusMessages[0])
		assert.Equal(t, "NoSchedule", result.Taints[0].Effect.ValueString())
		assert.Equal(t, "app", result.Taints[0].Key.ValueString())
		assert.Equal(t, "critical", result.Taints[0].Value.ValueString())
	})
}

func TestConvertToKubernetesCluster(t *testing.T) {
	t.Run("should return nil for nil input", func(t *testing.T) {
		result := convertToKubernetesCluster(nil, "br-se1")
		assert.Nil(t, result)
	})

	t.Run("should handle minimal sdk input correctly", func(t *testing.T) {
		region := "br-se1"
		sdkCluster := &sdkK8s.Cluster{
			ID:      "cluster-id-123",
			Name:    "test-cluster",
			Version: "1.28.0",
			Region:  &region,
		}

		result := convertToKubernetesCluster(sdkCluster, "br-se1")

		assert.NotNil(t, result)
		assert.Equal(t, types.StringValue("cluster-id-123"), result.ID)
		assert.Equal(t, types.StringValue("test-cluster"), result.Name)
		assert.Equal(t, types.StringValue("1.28.0"), result.Version)

		assert.True(t, result.Description.IsNull())
		assert.Nil(t, result.AllowedCIDRs)
		assert.True(t, result.KubeAPIPort.IsNull())
		assert.Nil(t, result.NodePools)
		assert.Nil(t, result.Controlplane)
	})

	t.Run("should convert full sdk input correctly", func(t *testing.T) {
		now := time.Now()
		sdkCluster := &sdkK8s.Cluster{
			ID:          "cluster-id-123",
			Name:        "prod-cluster",
			Description: stringPtr("Production cluster"),
			Version:     "1.28.0",
			Region:      stringPtr("br-se1"),
			CreatedAt:   &now,
			UpdatedAt:   &now,
			AllowedCIDRs: &[]string{
				"192.168.1.0/24",
			},
			Addons: &sdkK8s.Addons{
				Loadbalance: "enabled",
				Secrets:     "enabled",
				Volume:      "enabled",
			},
			KubeApiServer: &sdkK8s.KubeApiServer{
				DisableApiServerFip: boolPtr(false),
				FixedIp:             stringPtr("10.0.0.5"),
				FloatingIp:          stringPtr("200.1.2.3"),
				Port:                intPtr(6443),
			},
			Network: &sdkK8s.Network{
				VPCID:   "vpc-id-abc",
				Subnets: *MockedSubnets(),
			},
			Status: &sdkK8s.MessageState{
				Message: "Cluster is healthy",
				State:   "Running",
			},
			NodePools: &[]sdkK8s.NodePool{
				{ID: "np-1",
					Name: "worker-pool-1",
					Network: &sdkK8s.Network{
						VPCID:   "vpc-id-abc",
						Subnets: *MockedSubnets(),
					},
				},
			},
			ControlPlane: &sdkK8s.NodePool{
				ID: "cp-1",
			},
		}

		result := convertToKubernetesCluster(sdkCluster, "br-se1")
		expectedTime := types.StringValue(now.Format(time.RFC3339))

		assert.NotNil(t, result)
		assert.Equal(t, types.StringValue("cluster-id-123"), result.ID)
		assert.Equal(t, types.StringValue("Production cluster"), result.Description)
		assert.Equal(t, expectedTime, result.CreatedAt)
		assert.Equal(t, expectedTime, result.UpdatedAt)
		assert.Equal(t, types.StringValue("192.168.1.0/24"), result.AllowedCIDRs[0])
		assert.Equal(t, types.StringValue("enabled"), result.AddonsLoadbalance)
		assert.Equal(t, types.BoolValue(false), result.KubeAPIDisableAPIServerFIP)
		assert.Equal(t, types.StringValue("200.1.2.3"), result.KubeAPIFloatingIP)
		assert.Equal(t, types.Int64Value(6443), result.KubeAPIPort)
		assert.Equal(t, types.StringValue("Running"), result.State)
		assert.Equal(t, types.StringValue("np-1"), result.NodePools[0].ID)
		AssertSubnetIDs(t, result.SubnetIDs, sdkCluster.Network)
		for _, np := range result.NodePools {
			AssertSubnetIDs(t, np.SubnetIDs, sdkCluster.Network)
		}
	})

}

func TestDataSourceClusterNodePoolsSchemaMatchesStruct(t *testing.T) {
	ctx := context.Background()
	d := &DataSourceKubernetesCluster{}

	resp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, resp)

	nodePoolsAttr, ok := resp.Schema.Attributes["node_pools"].(schema.ListNestedAttribute)
	assert.True(t, ok, "node_pools must be a ListNestedAttribute")

	declared := map[string]struct{}{}
	for name := range nodePoolsAttr.NestedObject.Attributes {
		declared[name] = struct{}{}
	}

	npType := reflect.TypeOf(NodePool{})
	for i := 0; i < npType.NumField(); i++ {
		tag := npType.Field(i).Tag.Get("tfsdk")
		if tag == "" || tag == "-" {
			continue
		}
		tag = strings.Split(tag, ",")[0]
		t.Run("the data source schema for node_pools declares the '"+tag+"' attribute that the NodePool struct exposes", func(t *testing.T) {
			_, present := declared[tag]
			assert.True(t, present,
				"NodePool struct field %q has tfsdk tag %q, but the data source's nested node_pools schema does not declare it; this causes a schema/struct mismatch at runtime",
				npType.Field(i).Name, tag)
		})
	}
}

func MockedSubnets() *[]sdkK8s.Subnet {
	return &[]sdkK8s.Subnet{
		{
			ID:               "subnet-id-1",
			CIDR:             "10.0.0.0/24",
			AvailabilityZone: "br-se1-a",
		},
		{
			ID:               "subnet-id-2",
			CIDR:             "10.0.1.0/24",
			AvailabilityZone: "br-se1-b",
		},
		{
			ID:               "subnet-id-3",
			CIDR:             "10.0.0.0/24",
			AvailabilityZone: "br-se1-c",
		},
	}
}

func AssertSubnetIDs(t *testing.T, subnetIDs types.Set, sdkNetwork *sdkK8s.Network) {
	assert.False(t, subnetIDs.IsNull())

	var subnets []types.String
	subnetIDs.ElementsAs(context.Background(), &subnets, false)
	assert.Equal(t, len(sdkNetwork.Subnets), len(subnets))

	expected := make([]string, len(sdkNetwork.Subnets))
	for i, s := range sdkNetwork.Subnets {
		expected[i] = s.ID
	}
	actual := make([]string, len(subnets))
	for i, id := range subnets {
		actual[i] = id.ValueString()
	}
	assert.ElementsMatch(t, expected, actual)
}
