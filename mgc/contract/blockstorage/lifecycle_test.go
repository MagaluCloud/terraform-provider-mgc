//go:build contract

package blockstorage_test

import (
	"encoding/json"
	"fmt"
	ct "github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/contracttest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"net/http"
	"regexp"
	"sync"
	"testing"
)

type attachmentAPI struct {
	mu                 sync.Mutex
	t                  *testing.T
	attached           bool
	failRead           bool
	attaches, detaches int
}

func (a *attachmentAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.Method == "POST" && r.URL.Path == "/volume/v1/volumes/volume-1/attach/vm-1":
		if a.attached {
			a.t.Error("duplicate attachment without recovery detach")
			w.WriteHeader(409)
			return
		}
		a.attached = true
		a.attaches++
		w.WriteHeader(204)
	case r.Method == "POST" && r.URL.Path == "/volume/v1/volumes/volume-1/detach":
		if !a.attached {
			a.t.Error("unexpected detach")
			w.WriteHeader(409)
			return
		}
		a.attached = false
		a.detaches++
		w.WriteHeader(204)
	case r.Method == "GET" && r.URL.Path == "/volume/v1/volumes/volume-1":
		if a.failRead {
			w.WriteHeader(403)
			fmt.Fprint(w, `{"message":"injected attachment read failure"}`)
			return
		}
		result := map[string]any{"id": "volume-1", "status": "completed"}
		if a.attached {
			result["attachment"] = map[string]any{"instance": map[string]any{"id": "vm-1"}}
		}
		json.NewEncoder(w).Encode(result)
	default:
		a.t.Errorf("unexpected volume mutation or request: %s %s", r.Method, r.URL)
		w.WriteHeader(400)
	}
}
func TestContractAttachmentPartialCreateRecovery(t *testing.T) {
	a := &attachmentAPI{t: t, failRead: true}
	s := ct.Server(t, a)
	config := ct.ProviderConfig(s.URL) + `resource "mgc_block_storage_volume_attachment" "test" {
 block_storage_id="volume-1"
 virtual_machine_id="vm-1"
 }`
	c := ct.Case()
	c.Steps = []resource.TestStep{
		{Config: config, ExpectError: regexp.MustCompile("injected attachment read failure|403")},
		{PreConfig: func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			if !a.attached || a.attaches != 1 {
				t.Error("API must retain accepted attachment")
			}
			a.failRead = false
		}, Config: config, ConfigPlanChecks: ct.StablePlan("mgc_block_storage_volume_attachment.test", plancheck.ResourceActionDestroyBeforeCreate), Check: resource.TestCheckResourceAttr("mgc_block_storage_volume_attachment.test", "block_storage_id", "volume-1")},
		{Config: config, ConfigPlanChecks: ct.StablePlan("mgc_block_storage_volume_attachment.test", plancheck.ResourceActionNoop)},
	}
	c.CheckDestroy = func(_ *terraform.State) error {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.attached || a.attaches != 2 || a.detaches != 2 {
			return fmt.Errorf("incorrect attachment recovery: attached=%v attaches=%d detaches=%d", a.attached, a.attaches, a.detaches)
		}
		return nil
	}
	resource.Test(t, c)
}
