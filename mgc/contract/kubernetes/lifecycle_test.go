//go:build contract

package kubernetes_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	ct "github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/contracttest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestContractClusterUpgradeDoesNotReplace(t *testing.T) {
	a := &clusterAPI{t: t, delayPatchReads: 1}
	s := ct.Server(t, a)
	c := ct.Case()
	c.Steps = []resource.TestStep{
		{Config: clusterConfig(s.URL, "v1.31.0", "original", ""), ConfigPlanChecks: ct.StablePlan("mgc_kubernetes_cluster.test", plancheck.ResourceActionCreate), Check: a.check("v1.31.0", "original", 0)},
		{Config: clusterConfig(s.URL, "v1.32.0", "original", ""), ConfigPlanChecks: ct.StablePlan("mgc_kubernetes_cluster.test", plancheck.ResourceActionUpdate), Check: resource.ComposeTestCheckFunc(a.check("v1.32.0", "original", 1), resource.TestCheckResourceAttr("mgc_kubernetes_cluster.test", "id", "cluster-1"))},
		{Config: clusterConfig(s.URL, "v1.32.0", "edited", ""), ConfigPlanChecks: ct.StablePlan("mgc_kubernetes_cluster.test", plancheck.ResourceActionUpdate), Check: a.check("v1.32.0", "edited", 2)},
		{ResourceName: "mgc_kubernetes_cluster.test", ImportState: true, ImportStateVerify: true},
		{Config: clusterConfig(s.URL, "v1.32.0", "edited", `subnet_ids = ["subnet-b", "subnet-a"]`), ConfigPlanChecks: ct.StablePlan("mgc_kubernetes_cluster.test", plancheck.ResourceActionNoop), Check: a.check("v1.32.0", "edited", 2)},
		{Config: clusterConfig(s.URL, "v1.32.0", "edited", `subnet_ids = ["subnet-a", "subnet-b"]`), ConfigPlanChecks: ct.StablePlan("mgc_kubernetes_cluster.test", plancheck.ResourceActionNoop), Check: a.check("v1.32.0", "edited", 2)},
		{Config: clusterConfig(s.URL, "v1.32.0", "edited", ""), ConfigPlanChecks: ct.StablePlan("mgc_kubernetes_cluster.test", plancheck.ResourceActionNoop)},
	}
	c.CheckDestroy = func(_ *terraform.State) error {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.cluster != nil || a.creates != 1 || a.deletes != 1 {
			return fmt.Errorf("unexpected cluster destruction: creates=%d deletes=%d remaining=%v", a.creates, a.deletes, a.cluster)
		}
		return nil
	}
	resource.Test(t, c)
}

func TestContractClusterPreventDestroy(t *testing.T) {
	a := &clusterAPI{t: t}
	s := ct.Server(t, a)
	c := ct.Case()
	config := clusterConfig(s.URL, "v1.31.0", "original", `lifecycle { prevent_destroy = true }`)
	c.Steps = []resource.TestStep{
		{Config: config},
		{Config: strings.Replace(config, `name = "contract-cluster"`, `name = "replacement-cluster"`, 1), PlanOnly: true, ExpectError: regexp.MustCompile("cannot be destroyed")},
		{Config: config, ConfigPlanChecks: ct.StablePlan("mgc_kubernetes_cluster.test", plancheck.ResourceActionNoop), Check: a.check("v1.31.0", "original", 0)},
		// Remove the guard explicitly to allow this test's local cleanup.
		{Config: clusterConfig(s.URL, "v1.31.0", "original", ""), ConfigPlanChecks: ct.StablePlan("mgc_kubernetes_cluster.test", plancheck.ResourceActionNoop)},
	}
	c.CheckDestroy = func(_ *terraform.State) error {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.cluster != nil || a.creates != 1 || a.deletes != 1 {
			return fmt.Errorf("unexpected cluster lifecycle: creates=%d deletes=%d", a.creates, a.deletes)
		}
		return nil
	}
	resource.Test(t, c)
}

func TestContractClusterDeleteFailurePreservesState(t *testing.T) {
	a := &clusterAPI{t: t}
	s := ct.Server(t, a)
	config := clusterConfig(s.URL, "v1.31.0", "original", "")
	c := ct.Case()
	c.Steps = []resource.TestStep{
		{Config: config},
		{PreConfig: func() { a.mu.Lock(); a.deleteReadFailure = true; a.mu.Unlock() }, Config: config, Destroy: true, ExpectError: regexp.MustCompile("injected delete polling failure|403")},
		{PreConfig: func() { a.mu.Lock(); a.readStatus = 0; a.deleteReadFailure = false; a.mu.Unlock() }, Config: config, Destroy: true},
	}
	c.CheckDestroy = func(_ *terraform.State) error {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.cluster != nil || a.creates != 1 || a.deletes != 2 {
			return fmt.Errorf("failed delete lost cluster state: creates=%d deletes=%d remaining=%v", a.creates, a.deletes, a.cluster)
		}
		return nil
	}
	resource.Test(t, c)
}

func TestContractClusterImportExisting(t *testing.T) {
	a := &clusterAPI{t: t, cluster: map[string]any{
		"id": "cluster-1", "name": "contract-cluster", "version": "v1.31.0", "description": "original",
		"allowed_cidrs": []string{"192.0.2.0/24"}, "status": map[string]any{"state": "running"},
		"created_at": "2025-01-01T00:00:00Z", "updated_at": "2025-01-01T00:00:00Z", "region": "br-se1",
		"cluster_ipv4_cidr": "10.0.0.0/16", "services_ipv4_cidr": "10.96.0.0/16",
		"network": map[string]any{"subnets": []map[string]string{{"id": "subnet-a"}, {"id": "subnet-b"}}},
	}}
	s := ct.Server(t, a)
	config := clusterConfig(s.URL, "v1.31.0", "original", "")
	c := ct.Case()
	c.Steps = []resource.TestStep{
		{Config: config + `
import {
 to = mgc_kubernetes_cluster.test
 id = "cluster-1"
}`, ConfigPlanChecks: ct.StablePlan("mgc_kubernetes_cluster.test", plancheck.ResourceActionNoop)},
		{Config: config, ConfigPlanChecks: ct.StablePlan("mgc_kubernetes_cluster.test", plancheck.ResourceActionNoop), Check: resource.TestCheckResourceAttr("mgc_kubernetes_cluster.test", "id", "cluster-1")},
	}
	c.CheckDestroy = func(_ *terraform.State) error {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.cluster != nil || a.creates != 0 || a.patches != 0 || a.deletes != 1 {
			return fmt.Errorf("import modified cluster: creates=%d patches=%d deletes=%d", a.creates, a.patches, a.deletes)
		}
		return nil
	}
	resource.Test(t, c)
}
