package kubernetes_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	vmSDK "github.com/MagaluCloud/mgc-sdk-go/compute"
	k8sSDK "github.com/MagaluCloud/mgc-sdk-go/kubernetes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/acctest"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

const nodepoolResourceName = "mgc_kubernetes_nodepool.test"

func testAccNodepoolConfig_basic(name, flavor string, replicas int) string {
	return acctest.ProviderConfig(utils.ServiceKubernetes) + fmt.Sprintf(`
resource "mgc_kubernetes_cluster" "test" {
  name = %[1]q
}

resource "mgc_kubernetes_nodepool" "test" {
  cluster_id  = mgc_kubernetes_cluster.test.id
  name        = %[1]q
  flavor_name = %[2]q
  replicas    = %[3]d
}
`, name, flavor, replicas)
}

func testAccNodepoolConfig_autoscale(name, flavor string, replicas, minReplicas, maxReplicas int) string {
	return acctest.ProviderConfig(utils.ServiceKubernetes) + fmt.Sprintf(`
resource "mgc_kubernetes_cluster" "test" {
  name = %[1]q
}

resource "mgc_kubernetes_nodepool" "test" {
  cluster_id   = mgc_kubernetes_cluster.test.id
  name         = %[1]q
  flavor_name  = %[2]q
  replicas     = %[3]d
  min_replicas = %[4]d
  max_replicas = %[5]d
}
`, name, flavor, replicas, minReplicas, maxReplicas)
}

func testAccNodepoolConfig_taints(name, flavor string) string {
	return acctest.ProviderConfig(utils.ServiceKubernetes) + fmt.Sprintf(`
resource "mgc_kubernetes_cluster" "test" {
  name = %[1]q
}

resource "mgc_kubernetes_nodepool" "test" {
  cluster_id  = mgc_kubernetes_cluster.test.id
  name        = %[1]q
  flavor_name = %[2]q
  replicas    = 1
  taints = [
    {
      key    = "tf-acctest"
      value  = "true"
      effect = "NoSchedule"
    },
  ]
}
`, name, flavor)
}

func testAccNodepoolConfig_labels(name, flavor string) string {
	return acctest.ProviderConfig(utils.ServiceKubernetes) + fmt.Sprintf(`
resource "mgc_kubernetes_cluster" "test" {
  name = %[1]q
}

resource "mgc_kubernetes_nodepool" "test" {
  cluster_id  = mgc_kubernetes_cluster.test.id
  name        = %[1]q
  flavor_name = %[2]q
  replicas    = 1
  labels = {
    environment = "acc-test"
    team        = "DevX"
    tier        = "backend"
  }
}
`, name, flavor)
}

func testAccNodepoolConfig_empty_labels(name, flavor string) string {
	return acctest.ProviderConfig(utils.ServiceKubernetes) + fmt.Sprintf(`
resource "mgc_kubernetes_cluster" "test" {
  name = %[1]q
}

resource "mgc_kubernetes_nodepool" "test" {
  cluster_id  = mgc_kubernetes_cluster.test.id
  name        = %[1]q
  flavor_name = %[2]q
  replicas    = 1
  labels = {
  }
}
`, name, flavor)
}

func nodepoolService(vcr *acctest.VCR) k8sSDK.NodePoolService {
	return k8sSDK.New(vcr.SDKClient(utils.ServiceKubernetes)).Nodepools()
}

func nodepoolFlavor(t *testing.T, vcr *acctest.VCR) string {
	t.Helper()
	flavors, err := vmSDK.New(vcr.SDKClient(utils.ServiceVirtualMachine)).InstanceTypes().List(context.Background(), vmSDK.InstanceTypeListOptions{})
	if err != nil {
		t.Fatalf("listing k8s flavors: %v", err)
	}
	nps := flavors.InstanceTypes
	if len(nps) == 0 {
		t.Skip("no nodepool flavors available")
	}
	sort.Slice(nps, func(i, j int) bool {
		if nps[i].VCPUs != nps[j].VCPUs {
			return nps[i].VCPUs < nps[j].VCPUs
		}
		if nps[i].RAM != nps[j].RAM {
			return nps[i].RAM < nps[j].RAM
		}
		return nps[i].Name < nps[j].Name
	})

	for _, np := range nps {
		if np.RAM >= 2048 &&
			np.Disk >= 40 &&
			np.VCPUs == 2 &&
			*np.GPU == 0 &&
			len(*np.AvailabilityZones) >= 2 {
			return np.Name
		}
	}
	return nps[0].Name
}

func nodepoolImportID(s *terraform.State) (string, error) {
	rs, ok := s.RootModule().Resources[nodepoolResourceName]
	if !ok {
		return "", fmt.Errorf("not found in state: %s", nodepoolResourceName)
	}
	return fmt.Sprintf("%s,%s", rs.Primary.Attributes["cluster_id"], rs.Primary.ID), nil
}

func testAccCheckNodepoolExists(vcr *acctest.VCR, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found in state: %s", n)
		}
		if rs.Primary.ID == "" {
			return errors.New("no nodepool id set in state")
		}

		np, err := nodepoolService(vcr).Get(context.Background(), rs.Primary.Attributes["cluster_id"], rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("fetching nodepool %s from API: %w", rs.Primary.ID, err)
		}

		if np.Name != rs.Primary.Attributes["name"] {
			return fmt.Errorf("name: API %q, state %q", np.Name, rs.Primary.Attributes["name"])
		}
		if np.InstanceTemplate.Flavor.Name != rs.Primary.Attributes["flavor_name"] {
			return fmt.Errorf("flavor_name: API %q, state %q", np.InstanceTemplate.Flavor.Name, rs.Primary.Attributes["flavor_name"])
		}
		if got := fmt.Sprintf("%d", np.Replicas); got != rs.Primary.Attributes["replicas"] {
			return fmt.Errorf("replicas: API %s, state %s", got, rs.Primary.Attributes["replicas"])
		}
		return nil
	}
}

func testAccCheckNodepoolDisappears(vcr *acctest.VCR, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found in state: %s", n)
		}

		ctx := context.Background()
		svc := nodepoolService(vcr)
		clusterID := rs.Primary.Attributes["cluster_id"]

		if err := svc.Delete(ctx, clusterID, rs.Primary.ID); err != nil {
			return fmt.Errorf("deleting nodepool %s out of band: %w", rs.Primary.ID, err)
		}

		for start := time.Now(); time.Since(start) < acctest.PollTimeout(30*time.Minute); {
			np, err := svc.Get(ctx, clusterID, rs.Primary.ID)
			if err != nil {
				var httpErr *clientSDK.HTTPError
				if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
					return nil
				}
				return fmt.Errorf("polling deletion of nodepool %s: %w", rs.Primary.ID, err)
			}
			if strings.EqualFold(np.Status.State, "deleted") {
				return nil
			}
			time.Sleep(acctest.PollInterval(1 * time.Minute))
		}
		return fmt.Errorf("timeout waiting for nodepool %s to disappear", rs.Primary.ID)
	}
}

func testAccCheckNodepoolDestroyed(vcr *acctest.VCR) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		svc := nodepoolService(vcr)
		for name, rs := range s.RootModule().Resources {
			if rs.Type != "mgc_kubernetes_nodepool" {
				continue
			}

			np, err := svc.Get(context.Background(), rs.Primary.Attributes["cluster_id"], rs.Primary.ID)
			if err != nil {
				var httpErr *clientSDK.HTTPError
				if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
					continue
				}
				return fmt.Errorf("checking destruction of %s (%s): %w", name, rs.Primary.ID, err)
			}
			if strings.EqualFold(np.Status.State, "deleted") {
				continue
			}
			return fmt.Errorf("nodepool %s (%s) still exists after destroy", name, rs.Primary.ID)
		}
		return nil
	}
}

func TestAccKubernetesNodepool_basic(t *testing.T) {
	vcr := acctest.NewVCR(t)
	name := vcr.RandomName("nodepool")
	flavor := nodepoolFlavor(t, vcr)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: vcr.ProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckNodepoolDestroyed(vcr),
		Steps: []resource.TestStep{
			{
				Config: testAccNodepoolConfig_basic(name, flavor, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNodepoolExists(vcr, nodepoolResourceName),
					resource.TestCheckResourceAttr(nodepoolResourceName, "name", name),
					resource.TestCheckResourceAttr(nodepoolResourceName, "flavor_name", flavor),
					resource.TestCheckResourceAttr(nodepoolResourceName, "replicas", "1"),
					resource.TestCheckResourceAttrSet(nodepoolResourceName, "id"),
					resource.TestCheckResourceAttrSet(nodepoolResourceName, "cluster_id"),
					resource.TestCheckResourceAttrSet(nodepoolResourceName, "created_at"),
					resource.TestCheckResourceAttrSet(nodepoolResourceName, "version"),
				),
			},
			{
				ResourceName:      nodepoolResourceName,
				ImportState:       true,
				ImportStateIdFunc: nodepoolImportID,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccKubernetesNodepool_disappears(t *testing.T) {
	t.Parallel()
	vcr := acctest.NewVCR(t)
	name := vcr.RandomName("nodepool")
	flavor := nodepoolFlavor(t, vcr)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: vcr.ProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckNodepoolDestroyed(vcr),
		Steps: []resource.TestStep{
			{
				Config: testAccNodepoolConfig_basic(name, flavor, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNodepoolExists(vcr, nodepoolResourceName),
					testAccCheckNodepoolDisappears(vcr, nodepoolResourceName),
				),
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(nodepoolResourceName, plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func TestAccKubernetesNodepool_parentDisappears(t *testing.T) {
	t.Parallel()
	vcr := acctest.NewVCR(t)
	name := vcr.RandomName("nodepool")
	flavor := nodepoolFlavor(t, vcr)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: vcr.ProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckNodepoolDestroyed(vcr),
		Steps: []resource.TestStep{
			{
				Config: testAccNodepoolConfig_basic(name, flavor, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNodepoolExists(vcr, nodepoolResourceName),
					testAccCheckClusterDisappears(vcr, clusterResourceName),
				),
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterResourceName, plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction(nodepoolResourceName, plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func TestAccKubernetesNodepool_replicasIgnoredAfterCreate(t *testing.T) {
	t.Parallel()
	vcr := acctest.NewVCR(t)
	name := vcr.RandomName("nodepool")
	flavor := nodepoolFlavor(t, vcr)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: vcr.ProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckNodepoolDestroyed(vcr),
		Steps: []resource.TestStep{
			{
				Config: testAccNodepoolConfig_basic(name, flavor, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNodepoolExists(vcr, nodepoolResourceName),
					resource.TestCheckResourceAttr(nodepoolResourceName, "replicas", "1"),
				),
			},
			{
				Config: testAccNodepoolConfig_basic(name, flavor, 2),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(nodepoolResourceName, plancheck.ResourceActionUpdate),
					},
				},
			},
			{
				Config: testAccNodepoolConfig_basic(name, flavor, 2),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(nodepoolResourceName, plancheck.ResourceActionNoop),
					},
				},
			},
		},
	})
}

func TestAccKubernetesNodepool_autoscale(t *testing.T) {
	t.Parallel()
	vcr := acctest.NewVCR(t)
	name := vcr.RandomName("nodepool")
	flavor := nodepoolFlavor(t, vcr)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: vcr.ProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckNodepoolDestroyed(vcr),
		Steps: []resource.TestStep{
			{
				Config: testAccNodepoolConfig_autoscale(name, flavor, 1, 1, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNodepoolExists(vcr, nodepoolResourceName),
					resource.TestCheckResourceAttr(nodepoolResourceName, "min_replicas", "1"),
					resource.TestCheckResourceAttr(nodepoolResourceName, "max_replicas", "2"),
				),
			},
			{
				ResourceName:            nodepoolResourceName,
				ImportState:             true,
				ImportStateIdFunc:       nodepoolImportID,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"replicas"},
			},
			{
				Config: testAccNodepoolConfig_autoscale(name, flavor, 1, 2, 3),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(nodepoolResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNodepoolExists(vcr, nodepoolResourceName),
					resource.TestCheckResourceAttr(nodepoolResourceName, "min_replicas", "2"),
					resource.TestCheckResourceAttr(nodepoolResourceName, "max_replicas", "3"),
				),
			},
			{
				ResourceName:            nodepoolResourceName,
				ImportState:             true,
				ImportStateIdFunc:       nodepoolImportID,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"replicas"},
			},
		},
	})
}

func TestAccKubernetesNodepool_taints(t *testing.T) {
	t.Parallel()
	vcr := acctest.NewVCR(t)
	name := vcr.RandomName("nodepool")
	flavor := nodepoolFlavor(t, vcr)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: vcr.ProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckNodepoolDestroyed(vcr),
		Steps: []resource.TestStep{
			{
				Config: testAccNodepoolConfig_taints(name, flavor),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNodepoolExists(vcr, nodepoolResourceName),
					resource.TestCheckResourceAttr(nodepoolResourceName, "taints.#", "1"),
					resource.TestCheckResourceAttr(nodepoolResourceName, "taints.0.key", "tf-acctest"),
					resource.TestCheckResourceAttr(nodepoolResourceName, "taints.0.value", "true"),
					resource.TestCheckResourceAttr(nodepoolResourceName, "taints.0.effect", "NoSchedule"),
				),
			},
			{
				ResourceName:            nodepoolResourceName,
				ImportState:             true,
				ImportStateIdFunc:       nodepoolImportID,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"replicas"},
			},
		},
	})
}

func TestAccKubernetesNodepool_labels(t *testing.T) {
	t.Parallel()
	vcr := acctest.NewVCR(t)
	name := vcr.RandomName("nodepool")
	flavor := nodepoolFlavor(t, vcr)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: vcr.ProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckNodepoolDestroyed(vcr),
		Steps: []resource.TestStep{
			{
				Config: testAccNodepoolConfig_labels(name, flavor),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNodepoolExists(vcr, nodepoolResourceName),
				),
			},
			{
				Config: testAccNodepoolConfig_empty_labels(name, flavor),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(nodepoolResourceName, plancheck.ResourceActionReplace),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(nodepoolResourceName, plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNodepoolExists(vcr, nodepoolResourceName),
					resource.TestCheckResourceAttr(nodepoolResourceName, "labels", ""),
				),
			},
			{
				ResourceName:            nodepoolResourceName,
				ImportState:             true,
				ImportStateIdFunc:       nodepoolImportID,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"replicas"},
			},
		},
	})
}
