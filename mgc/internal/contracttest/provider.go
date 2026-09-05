//go:build contract

package contracttest

import (
	"fmt"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ProviderConfig configures every service endpoint with a local fixture and synthetic credentials.
func ProviderConfig(endpoint string) string {
	// Synthetic credentials; only the schema format matters for these local tests.
	return fmt.Sprintf(`provider "mgc" {
  api_key = "00000000-0000-4000-8000-000000000000"
  key_pair_id = "00000000-0000-4000-8000-000000000000"
  key_pair_secret = "00000000-0000-4000-8000-000000000000"
  region = "br-se1"
  endpoints {
    network = %[1]q
    kubernetes = %[1]q
    object_storage = %[1]q
    virtual_machine = %[1]q
    block_storage = %[1]q
    database = %[1]q
    ssh = %[1]q
    container_registry = %[1]q
    lbaas = %[1]q
  }
}
`, endpoint)
}

// Case uses the production provider through protocol v6. Local tests do not require TF_ACC.
func Case() resource.TestCase {
	return resource.TestCase{
		IsUnitTest: true, // Local API only: never gated by TF_ACC.
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"mgc": providerserver.NewProtocol6WithError(mgc.New("contract-test")()),
		},
	}
}
