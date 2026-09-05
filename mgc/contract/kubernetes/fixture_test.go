//go:build contract

package kubernetes_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	ct "github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/contracttest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

type clusterAPI struct {
	mu                        sync.Mutex
	t                         *testing.T
	cluster                   map[string]any
	creates, patches, deletes int
	deleteReadFailure         bool
	readStatus                int
	delayPatchReads           int
	remainingPatchReads       int
	pendingPatch              map[string]any
}

func (a *clusterAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.Method == "POST" && r.URL.Path == "/kubernetes/v0/clusters":
		if a.cluster != nil {
			a.t.Error("unexpected cluster replacement")
			w.WriteHeader(409)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			a.t.Error(err)
			w.WriteHeader(400)
			return
		}
		if body["enabled_server_group"] != true {
			a.t.Errorf("server group default changed: %v", body)
		}
		a.creates++
		a.cluster = body
		a.cluster["id"] = fmt.Sprintf("cluster-%d", a.creates)
		a.cluster["status"] = map[string]any{"state": "running"}
		a.cluster["created_at"] = "2025-01-01T00:00:00Z"
		a.cluster["updated_at"] = "2025-01-01T00:00:00Z"
		a.cluster["region"] = "br-se1"
		a.cluster["cluster_ipv4_cidr"] = "10.0.0.0/16"
		a.cluster["services_ipv4_cidr"] = "10.96.0.0/16"
		a.cluster["network"] = map[string]any{"subnets": []map[string]string{{"id": "subnet-a"}, {"id": "subnet-b"}}}
		delete(a.cluster, "enabled_server_group")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(a.cluster)
	case strings.HasPrefix(r.URL.Path, "/kubernetes/v0/clusters/cluster-"):
		if a.cluster == nil {
			w.WriteHeader(404)
			fmt.Fprint(w, `{"message":"not found"}`)
			return
		}
		switch r.Method {
		case "GET":
			if a.pendingPatch != nil {
				if a.remainingPatchReads > 0 {
					a.remainingPatchReads--
				} else {
					for k, v := range a.pendingPatch {
						a.cluster[k] = v
					}
					a.pendingPatch = nil
				}
			}
			if a.readStatus != 0 {
				w.WriteHeader(a.readStatus)
				fmt.Fprint(w, `{"message":"injected delete polling failure"}`)
				return
			}
			json.NewEncoder(w).Encode(a.cluster)
		case "PATCH":
			var patch map[string]any
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				a.t.Error(err)
				w.WriteHeader(400)
				return
			}
			for k, v := range patch {
				if k != "allowed_cidrs" && k != "description" && k != "version" {
					a.t.Errorf("unexpected cluster patch field %q", k)
				}
				if a.delayPatchReads == 0 {
					a.cluster[k] = v
				}
			}
			if a.delayPatchReads > 0 {
				a.pendingPatch = patch
				a.remainingPatchReads = a.delayPatchReads
			}
			a.patches++
			json.NewEncoder(w).Encode(patch)
		case "DELETE":
			a.deletes++
			if a.deleteReadFailure {
				a.readStatus = 403
				w.WriteHeader(204)
				return
			}
			a.cluster = nil
			w.WriteHeader(204)
		default:
			a.t.Errorf("unexpected cluster request: %s %s", r.Method, r.URL)
			w.WriteHeader(400)
		}
	default:
		a.t.Errorf("unexpected API request: %s %s", r.Method, r.URL)
		w.WriteHeader(400)
	}
}

func clusterConfig(endpoint, version, description, subnets string) string {
	return ct.ProviderConfig(endpoint) + fmt.Sprintf(`resource "mgc_kubernetes_cluster" "test" {
  name = "contract-cluster"
  version = %q
  description = %q
  allowed_cidrs = ["192.0.2.0/24"]
  %s
}`, version, description, subnets)
}

func (a *clusterAPI) check(version, description string, patches int) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.cluster == nil || a.cluster["version"] != version || a.cluster["description"] != description || a.patches != patches || a.creates != 1 || a.deletes != 0 {
			return fmt.Errorf("unexpected cluster state or destructive operation: cluster=%v creates=%d patches=%d deletes=%d", a.cluster, a.creates, a.patches, a.deletes)
		}
		return nil
	}
}
