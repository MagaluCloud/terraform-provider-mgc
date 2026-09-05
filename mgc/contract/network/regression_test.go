//go:build contract

package network_test

import (
	"fmt"
	"regexp"
	"testing"

	ct "github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/contracttest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestContractVPCDisappears(t *testing.T) {
	a := &vpcAPI{t: t}
	s := ct.Server(t, a)
	config := vpcConfig(s.URL)
	c := ct.Case()
	c.Steps = []resource.TestStep{
		{Config: config},
		{PreConfig: func() { a.mu.Lock(); a.vpc = nil; a.mu.Unlock() }, Config: config,
			ConfigPlanChecks: ct.StablePlan("mgc_network_vpcs.test", plancheck.ResourceActionCreate),
			Check:            resource.TestCheckResourceAttr("mgc_network_vpcs.test", "id", "vpc-2")},
	}
	c.CheckDestroy = func(_ *terraform.State) error {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.vpc != nil || a.creates != 2 || a.deletes != 1 {
			return fmt.Errorf("unexpected counts: %d creates, %d deletes", a.creates, a.deletes)
		}
		return nil
	}
	resource.Test(t, c)
}

// A create response followed by a read failure must not orphan the new VPC.
// Terraform taints partial creates, so the next apply must explicitly replace
// the recorded ID, rather than silently creating another unmanaged resource.
func TestContractVPCPartialCreateRecovery(t *testing.T) {
	a := &vpcAPI{t: t}
	s := ct.Server(t, a)
	config := vpcConfig(s.URL)
	c := ct.Case()
	c.Steps = []resource.TestStep{
		{PreConfig: func() { a.mu.Lock(); a.readStatus = 403; a.mu.Unlock() }, Config: config, ExpectError: regexp.MustCompile("(?s)Error waiting for VPC creation.*(injected read failure|403)")},
		{PreConfig: func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			if a.vpc == nil || a.creates != 1 {
				t.Errorf("expected one partial creation: %v", a.vpc)
			}
			a.readStatus = 0
		}, Config: config,
			ConfigPlanChecks: ct.StablePlan("mgc_network_vpcs.test", plancheck.ResourceActionDestroyBeforeCreate), Check: resource.TestCheckResourceAttr("mgc_network_vpcs.test", "id", "vpc-2")},
	}
	c.CheckDestroy = func(_ *terraform.State) error {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.vpc != nil || a.creates != 2 || a.deletes != 2 {
			return fmt.Errorf("partial creation was not recovered: creates=%d deletes=%d remaining=%v", a.creates, a.deletes, a.vpc)
		}
		return nil
	}
	resource.Test(t, c)
}
