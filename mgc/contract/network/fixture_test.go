//go:build contract

package network_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"testing"

	ct "github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/contracttest"
)

type vpcAPI struct {
	mu               sync.Mutex
	t                *testing.T
	vpc              map[string]any
	creates, deletes int
	readStatus       int
}

func (a *vpcAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.Method == "POST" && r.URL.Path == "/network/v1/vpcs":
		if a.vpc != nil {
			a.t.Error("unexpected duplicate VPC creation")
			http.Error(w, "duplicate", 409)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			a.t.Error(err)
			w.WriteHeader(400)
			return
		}
		if body["name"] != "contract-vpc" {
			a.t.Errorf("unexpected VPC request: %v", body)
		}
		a.creates++
		a.vpc = map[string]any{"id": fmt.Sprintf("vpc-%d", a.creates), "name": body["name"], "description": body["description"], "status": "created"}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(a.vpc)
	case r.Method == "GET" && regexp.MustCompile(`^/network/v0/vpcs/vpc-\d+$`).MatchString(r.URL.Path):
		if a.readStatus != 0 {
			w.WriteHeader(a.readStatus)
			fmt.Fprint(w, `{"message":"injected read failure"}`)
			return
		}
		if a.vpc == nil {
			w.WriteHeader(404)
			fmt.Fprint(w, `{"message":"not found"}`)
			return
		}
		json.NewEncoder(w).Encode(a.vpc)
	case r.Method == "DELETE" && regexp.MustCompile(`^/network/v0/vpcs/vpc-\d+$`).MatchString(r.URL.Path):
		a.deletes++
		a.vpc = nil
		w.WriteHeader(204)
	default:
		a.t.Errorf("unexpected API call: %s %s", r.Method, r.URL)
		w.WriteHeader(400)
	}
}

func vpcConfig(endpoint string) string {
	return ct.ProviderConfig(endpoint) + `resource "mgc_network_vpcs" "test" { name = "contract-vpc" }`
}
