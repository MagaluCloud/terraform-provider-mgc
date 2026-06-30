package kubernetes_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	k8sSDK "github.com/MagaluCloud/mgc-sdk-go/kubernetes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/acctest"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

const clusterResourceName = "mgc_kubernetes_cluster.test"

func testAccClusterConfig_basic(name string) string {
	return acctest.ProviderConfig(utils.ServiceKubernetes) + fmt.Sprintf(`
			resource "mgc_kubernetes_cluster" "test" {
				name = %[1]q
			}
		`, name)
}

func testAccClusterConfig_description(name, description string) string {
	return acctest.ProviderConfig(utils.ServiceKubernetes) + fmt.Sprintf(`
resource "mgc_kubernetes_cluster" "test" {
  name        = %[1]q
  description = %[2]q
}
`, name, description)
}

func testAccClusterConfig_allowedCIDRs(name string, cidrs ...string) string {
	quoted := make([]string, len(cidrs))
	for i, c := range cidrs {
		quoted[i] = fmt.Sprintf("%q", c)
	}
	return acctest.ProviderConfig(utils.ServiceKubernetes) + fmt.Sprintf(`
resource "mgc_kubernetes_cluster" "test" {
  name          = %[1]q
  allowed_cidrs = [%[2]s]
}
`, name, strings.Join(quoted, ", "))
}

func testAccClusterConfig_version(name, version string) string {
	return acctest.ProviderConfig(utils.ServiceKubernetes) + fmt.Sprintf(`
resource "mgc_kubernetes_cluster" "test" {
  name    = %[1]q
  version = %[2]q
}
`, name, version)
}

func testAccClusterConfig_subnets(name string) string {
	return acctest.ProviderConfig(utils.ServiceKubernetes, utils.ServiceNetwork) + fmt.Sprintf(`
resource "mgc_network_vpcs" "vpc" {
  name        = "%[1]s-vpc"
  description = "created by acceptance test"
}

resource "mgc_network_subnetpools" "vpc_subnetpool" {
  name        = "%[1]s-pool"
  description = "created by acceptance test"
  cidr        = "10.20.0.0/16"
}

resource "mgc_network_vpcs_subnets" "vpc_subnet_ipv4_a" {
  cidr_block        = "10.20.1.0/24"
  ip_version        = "IPv4"
  name              = "%[1]s-a"
  subnetpool_id     = mgc_network_subnetpools.vpc_subnetpool.id
  vpc_id            = mgc_network_vpcs.vpc.id
  availability_zone = "br-se1-a"
}

resource "mgc_network_vpcs_subnets" "vpc_subnet_ipv4_b" {
  cidr_block        = "10.20.2.0/24"
  ip_version        = "IPv4"
  name              = "%[1]s-b"
  subnetpool_id     = mgc_network_subnetpools.vpc_subnetpool.id
  vpc_id            = mgc_network_vpcs.vpc.id
  availability_zone = "br-se1-b"
}

resource "mgc_network_vpcs_subnets" "vpc_subnet_ipv4_c" {
  cidr_block        = "10.20.3.0/24"
  ip_version        = "IPv4"
  name              = "%[1]s-c"
  subnetpool_id     = mgc_network_subnetpools.vpc_subnetpool.id
  vpc_id            = mgc_network_vpcs.vpc.id
  availability_zone = "br-se1-c"
}

resource "mgc_kubernetes_cluster" "test" {
  name = %[1]q
  subnet_ids = [
    mgc_network_vpcs_subnets.vpc_subnet_ipv4_a.id,
    mgc_network_vpcs_subnets.vpc_subnet_ipv4_b.id,
    mgc_network_vpcs_subnets.vpc_subnet_ipv4_c.id,
  ]
}
`, name)
}

func clusterService(vcr *acctest.VCR) k8sSDK.ClusterService {
	return k8sSDK.New(clientSDK.NewMgcClient(
		clientSDK.WithAPIKey(acctest.APIKey()),
		clientSDK.WithBaseURL(clientSDK.MgcUrl(acctest.Endpoint())),
		clientSDK.WithHTTPClient(vcr.HTTPClient()),
	)).Clusters()
}

func testAccCheckClusterExists(vcr *acctest.VCR, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found in state: %s", n)
		}
		if rs.Primary.ID == "" {
			return errors.New("no cluster id set in state")
		}

		cluster, err := clusterService(vcr).Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("fetching cluster %s from API: %w", rs.Primary.ID, err)
		}

		if cluster.Name != rs.Primary.Attributes["name"] {
			return fmt.Errorf("name: API %q, state %q", cluster.Name, rs.Primary.Attributes["name"])
		}
		if cluster.Version != rs.Primary.Attributes["version"] {
			return fmt.Errorf("version: API %q, state %q", cluster.Version, rs.Primary.Attributes["version"])
		}
		if desc := rs.Primary.Attributes["description"]; desc != "" {
			if cluster.Description == nil || *cluster.Description != desc {
				return fmt.Errorf("description: API %v, state %q", cluster.Description, desc)
			}
		}
		return nil
	}
}

func testAccCheckClusterDisappears(vcr *acctest.VCR, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found in state: %s", n)
		}
		if err := clusterService(vcr).Delete(context.Background(), rs.Primary.ID); err != nil {
			return fmt.Errorf("deleting cluster %s out of band: %w", rs.Primary.ID, err)
		}
		return nil
	}
}

func testAccCheckClusterDestroyed(vcr *acctest.VCR) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		svc := clusterService(vcr)
		for name, rs := range s.RootModule().Resources {
			if rs.Type != "mgc_kubernetes_cluster" {
				continue
			}

			cluster, err := svc.Get(context.Background(), rs.Primary.ID)
			if err != nil {
				var httpErr *clientSDK.HTTPError
				if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
					continue
				}
				return fmt.Errorf("checking destruction of %s (%s): %w", name, rs.Primary.ID, err)
			}
			if cluster.Status != nil && strings.EqualFold(cluster.Status.State, "deleted") {
				continue
			}
			return fmt.Errorf("cluster %s (%s) still exists after destroy", name, rs.Primary.ID)
		}
		return nil
	}
}

func TestAccKubernetesCluster_basic(t *testing.T) {
	vcr := acctest.NewVCR(t)
	name := vcr.RandomName("cluster-basic")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: vcr.ProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckClusterDestroyed(vcr),
		Steps: []resource.TestStep{
			{
				Config: testAccClusterConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckClusterExists(vcr, clusterResourceName),
					resource.TestCheckResourceAttr(clusterResourceName, "name", name),
					resource.TestCheckResourceAttrSet(clusterResourceName, "id"),
					resource.TestCheckResourceAttrSet(clusterResourceName, "version"),
					resource.TestCheckResourceAttrSet(clusterResourceName, "created_at"),
					resource.TestCheckResourceAttrSet(clusterResourceName, "region"),
					resource.TestCheckResourceAttrSet(clusterResourceName, "platform_version"),
					resource.TestCheckResourceAttrSet(clusterResourceName, "machine_types_source"),
				),
			},
			{
				ResourceName:      clusterResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccKubernetesCluster_disappears(t *testing.T) {
	t.Parallel()
	vcr := acctest.NewVCR(t)
	name := vcr.RandomName("cluster")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: vcr.ProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckClusterDestroyed(vcr),
		Steps: []resource.TestStep{
			{
				Config: testAccClusterConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckClusterExists(vcr, clusterResourceName),
					testAccCheckClusterDisappears(vcr, clusterResourceName),
				),
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterResourceName, plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func TestAccKubernetesCluster_requiresReplace(t *testing.T) {
	t.Parallel()
	vcr := acctest.NewVCR(t)
	nameA := vcr.RandomName("cluster")
	nameB := vcr.RandomName("cluster")

	var firstID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: vcr.ProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckClusterDestroyed(vcr),
		Steps: []resource.TestStep{
			{
				Config: testAccClusterConfig_basic(nameA),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckClusterExists(vcr, clusterResourceName),
					resource.TestCheckResourceAttrWith(clusterResourceName, "id", func(v string) error {
						if v == "" {
							return errors.New("id is empty")
						}
						firstID = v
						return nil
					}),
				),
			},
			{
				Config: testAccClusterConfig_basic(nameB),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterResourceName, plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckClusterExists(vcr, clusterResourceName),
					resource.TestCheckResourceAttr(clusterResourceName, "name", nameB),
					resource.TestCheckResourceAttrWith(clusterResourceName, "id", func(v string) error {
						if v == firstID {
							return fmt.Errorf("expected a new id after replace, still got %q", v)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccKubernetesCluster_description(t *testing.T) {
	t.Parallel()
	vcr := acctest.NewVCR(t)
	name := vcr.RandomName("cluster")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: vcr.ProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckClusterDestroyed(vcr),
		Steps: []resource.TestStep{
			{
				Config: testAccClusterConfig_description(name, "created by acceptance test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckClusterExists(vcr, clusterResourceName),
					resource.TestCheckResourceAttr(clusterResourceName, "description", "created by acceptance test"),
				),
			},
			{
				ResourceName:      clusterResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccClusterConfig_description(name, "updated by acceptance test"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckClusterExists(vcr, clusterResourceName),
					resource.TestCheckResourceAttr(clusterResourceName, "description", "updated by acceptance test"),
				),
			},
			{
				ResourceName:      clusterResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccClusterConfig_description(name, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: testAccCheckClusterExists(vcr, clusterResourceName),
			},
			{
				Config: testAccClusterConfig_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterResourceName, plancheck.ResourceActionUpdate),
					},
				},
			},
			{
				ResourceName:            clusterResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"description"},
			},
		},
	})
}

func TestAccKubernetesCluster_allowedCIDRs(t *testing.T) {
	t.Parallel()
	vcr := acctest.NewVCR(t)
	name := vcr.RandomName("cluster")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: vcr.ProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckClusterDestroyed(vcr),
		Steps: []resource.TestStep{
			{
				Config: testAccClusterConfig_allowedCIDRs(name, "192.168.0.0/24"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckClusterExists(vcr, clusterResourceName),
					resource.TestCheckResourceAttr(clusterResourceName, "allowed_cidrs.#", "1"),
					resource.TestCheckResourceAttr(clusterResourceName, "allowed_cidrs.0", "192.168.0.0/24"),
				),
			},
			{
				ResourceName:      clusterResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccClusterConfig_allowedCIDRs(name, "192.168.0.0/24", "10.0.0.0/16"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckClusterExists(vcr, clusterResourceName),
					resource.TestCheckResourceAttr(clusterResourceName, "allowed_cidrs.#", "2"),
				),
			},
			{
				ResourceName:      clusterResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccClusterConfig_allowedCIDRs(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: testAccCheckClusterExists(vcr, clusterResourceName),
			},
			{

				Config: testAccClusterConfig_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
			},
			{
				ResourceName:      clusterResourceName,
				ImportState:       true,
				ImportStateVerify: true,

				ImportStateVerifyIgnore: []string{"allowed_cidrs", "allowed_cidrs.#"},
			},
		},
	})
}

func TestAccKubernetesCluster_version(t *testing.T) {
	t.Parallel()
	version := os.Getenv("MGC_K8S_VERSION")
	upgrade := os.Getenv("MGC_K8S_VERSION_UPGRADE")
	if version == "" || upgrade == "" || version == upgrade {
		t.Skip("set MGC_K8S_VERSION and a distinct MGC_K8S_VERSION_UPGRADE to run the version upgrade test")
	}

	vcr := acctest.NewVCR(t)
	name := vcr.RandomName("cluster")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: vcr.ProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckClusterDestroyed(vcr),
		Steps: []resource.TestStep{
			{
				Config: testAccClusterConfig_version(name, version),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckClusterExists(vcr, clusterResourceName),
					resource.TestCheckResourceAttr(clusterResourceName, "version", version),
				),
			},
			{
				ResourceName:      clusterResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccClusterConfig_version(name, upgrade),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckClusterExists(vcr, clusterResourceName),
					resource.TestCheckResourceAttr(clusterResourceName, "version", upgrade),
				),
			},
			{
				ResourceName:      clusterResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccKubernetesCluster_subnets(t *testing.T) {
	t.Parallel()
	vcr := acctest.NewVCR(t)
	name := vcr.RandomName("subnets")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: vcr.ProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckClusterDestroyed(vcr),
		Steps: []resource.TestStep{
			{
				Config: testAccClusterConfig_subnets(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckClusterExists(vcr, clusterResourceName),
					resource.TestCheckResourceAttr(clusterResourceName, "name", name),
					resource.TestCheckResourceAttr(clusterResourceName, "subnet_ids.#", "3"),
				),
			},
			{
				ResourceName:      clusterResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
