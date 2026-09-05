//go:build contract

package objects_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ct "github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/contracttest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestContractObjectSameLengthUpdate(t *testing.T) {
	a := &objectAPI{t: t}
	s := s3Server(t, a)
	c := ct.Case()
	c.Steps = []resource.TestStep{
		{Config: objectConfig(s.URL, "abc"), ConfigPlanChecks: ct.StablePlan("mgc_object_storage_objects.test", plancheck.ResourceActionCreate), Check: a.checkContent("abc", 1)},
		{Config: objectConfig(s.URL, "xyz"), ConfigPlanChecks: ct.StablePlan("mgc_object_storage_objects.test", plancheck.ResourceActionUpdate), Check: a.checkContent("xyz", 2)},
		{Config: objectConfig(s.URL, "xyz"), ConfigPlanChecks: ct.StablePlan("mgc_object_storage_objects.test", plancheck.ResourceActionNoop), Check: a.checkContent("xyz", 2)},
		{ResourceName: "mgc_object_storage_objects.test", ImportState: true, ImportStateId: "contract-bucket/config.txt", ImportStateVerify: true, ImportStateVerifyIgnore: []string{"content"}, ImportStateVerifyIdentifierAttribute: "key"},
	}
	c.CheckDestroy = func(_ *terraform.State) error {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.exists || a.deletes != 1 {
			return fmt.Errorf("object not destroyed exactly once: exists=%v deletes=%d", a.exists, a.deletes)
		}
		return nil
	}
	resource.Test(t, c)
}

func TestContractObjectDisappears(t *testing.T) {
	a := &objectAPI{t: t}
	s := s3Server(t, a)
	c := ct.Case()
	config := objectConfig(s.URL, "abc")
	c.Steps = []resource.TestStep{
		{Config: config},
		{PreConfig: func() { a.mu.Lock(); a.exists = false; a.mu.Unlock() }, Config: config, ConfigPlanChecks: ct.StablePlan("mgc_object_storage_objects.test", plancheck.ResourceActionCreate), Check: a.checkContent("abc", 2)},
	}
	c.CheckDestroy = func(_ *terraform.State) error {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.exists {
			return fmt.Errorf("object still exists")
		}
		return nil
	}
	resource.Test(t, c)
}

func TestContractObjectSourceChangeSameLength(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.txt")
	second := filepath.Join(dir, "second.txt")
	if err := os.WriteFile(first, []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("xyz"), 0600); err != nil {
		t.Fatal(err)
	}
	a := &objectAPI{t: t}
	s := s3Server(t, a)
	c := ct.Case()
	config := func(source string) string {
		return strings.Replace(objectConfig(s.URL, "unused"), `content = "unused"`, fmt.Sprintf("source = %q", source), 1)
	}
	c.Steps = []resource.TestStep{
		{Config: config(first), ConfigPlanChecks: ct.StablePlan("mgc_object_storage_objects.test", plancheck.ResourceActionCreate), Check: a.checkContent("abc", 1)},
		{Config: config(second), ConfigPlanChecks: ct.StablePlan("mgc_object_storage_objects.test", plancheck.ResourceActionUpdate), Check: a.checkContent("xyz", 2)},
		{Config: config(second), ConfigPlanChecks: ct.StablePlan("mgc_object_storage_objects.test", plancheck.ResourceActionNoop), Check: a.checkContent("xyz", 2)},
	}
	c.CheckDestroy = func(_ *terraform.State) error {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.exists || a.deletes != 1 {
			return fmt.Errorf("unexpected object destruction: exists=%v deletes=%d", a.exists, a.deletes)
		}
		return nil
	}
	resource.Test(t, c)
}
