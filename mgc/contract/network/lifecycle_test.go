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

func TestContractVPCLifecycle(t *testing.T) {
	a := &vpcAPI{t: t}
	s := ct.Server(t, a)
	config := vpcConfig(s.URL)
	c := ct.Case()
	c.CheckDestroy = func(_ *terraform.State) error {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.vpc != nil || a.creates != 1 || a.deletes != 1 {
			return fmt.Errorf("unexpected lifecycle: creates=%d deletes=%d remaining=%v", a.creates, a.deletes, a.vpc)
		}
		return nil
	}
	c.Steps = []resource.TestStep{
		{Config: config, ConfigPlanChecks: ct.StablePlan("mgc_network_vpcs.test", plancheck.ResourceActionCreate)},
		{Config: config, ConfigPlanChecks: ct.StablePlan("mgc_network_vpcs.test", plancheck.ResourceActionNoop)},
		{ResourceName: "mgc_network_vpcs.test", ImportState: true, ImportStateVerify: true},
		{Config: config, ConfigPlanChecks: ct.StablePlan("mgc_network_vpcs.test", plancheck.ResourceActionNoop)},
	}
	resource.Test(t, c)
}

func TestContractVPCReadFailurePreservesState(t *testing.T) {
	a := &vpcAPI{t: t}
	s := ct.Server(t, a)
	config := vpcConfig(s.URL)
	c := ct.Case()
	c.Steps = []resource.TestStep{
		{Config: config},
		{PreConfig: func() { a.mu.Lock(); a.readStatus = 403; a.mu.Unlock() }, Config: config, PlanOnly: true, ExpectError: regexp.MustCompile("injected read failure|403")},
		{PreConfig: func() { a.mu.Lock(); a.readStatus = 0; a.mu.Unlock() }, Config: config,
			ConfigPlanChecks: ct.StablePlan("mgc_network_vpcs.test", plancheck.ResourceActionNoop), Check: resource.TestCheckResourceAttr("mgc_network_vpcs.test", "id", "vpc-1")},
	}
	c.CheckDestroy = func(_ *terraform.State) error {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.vpc != nil || a.creates != 1 || a.deletes != 1 {
			return fmt.Errorf("read error caused an unexpected lifecycle: %d creates, %d deletes", a.creates, a.deletes)
		}
		return nil
	}
	resource.Test(t, c)
}

func TestContractVPCImportExisting(t *testing.T) {
	a := &vpcAPI{t: t, vpc: map[string]any{"id": "vpc-1", "name": "contract-vpc", "status": "created"}}
	s := ct.Server(t, a)
	config := vpcConfig(s.URL)
	c := ct.Case()
	c.Steps = []resource.TestStep{
		{Config: config + `
import {
   to = mgc_network_vpcs.test
   id = "vpc-1"
  }`, ConfigPlanChecks: ct.StablePlan("mgc_network_vpcs.test", plancheck.ResourceActionNoop)},
		{Config: config, ConfigPlanChecks: ct.StablePlan("mgc_network_vpcs.test", plancheck.ResourceActionNoop), Check: resource.TestCheckResourceAttr("mgc_network_vpcs.test", "id", "vpc-1")},
	}
	c.CheckDestroy = func(_ *terraform.State) error {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.vpc != nil || a.creates != 0 || a.deletes != 1 {
			return fmt.Errorf("import created or replaced resource: creates=%d deletes=%d", a.creates, a.deletes)
		}
		return nil
	}
	resource.Test(t, c)
}
