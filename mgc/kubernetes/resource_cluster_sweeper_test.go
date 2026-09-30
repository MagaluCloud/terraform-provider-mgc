package kubernetes_test

import (
	"context"
	"fmt"
	"log"
	"strings"
	"testing"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	k8sSDK "github.com/MagaluCloud/mgc-sdk-go/kubernetes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/acctest"
)

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func init() {
	resource.AddTestSweepers("mgc_kubernetes_cluster", &resource.Sweeper{
		Name: "mgc_kubernetes_cluster",
		F:    sweepClusters,
	})
}

func sweepClusterService() (k8sSDK.ClusterService, error) {
	if acctest.Endpoint() == "" || acctest.APIKey() == "" {
		return nil, fmt.Errorf("%s and %s must be set to sweep", acctest.EnvEndpoint, acctest.EnvAPIKey)
	}
	return k8sSDK.New(clientSDK.NewMgcClient(
		clientSDK.WithAPIKey(acctest.APIKey()),
		clientSDK.WithBaseURL(clientSDK.MgcUrl(acctest.Endpoint())),
	)).Clusters(), nil
}

func sweepClusters(region string) error {
	log.Printf("[INFO] sweep: starting kubernetes_cluster sweep (region=%q)", region)

	ctx := context.Background()
	svc, err := sweepClusterService()
	if err != nil {
		return err
	}

	const pageSize = 100
	limit := pageSize
	offset := 0
	swept := 0

	for {
		page, err := svc.List(ctx, k8sSDK.ListOptions{Limit: &limit, Offset: &offset})
		if err != nil {
			return fmt.Errorf("listing clusters to sweep: %w", err)
		}

		for _, c := range page {
			if !acctest.IsSweepableTestResource(c.Name) {
				continue
			}
			if c.Status != nil && strings.EqualFold(c.Status.State, "deleting") {
				continue
			}
			if err := svc.Delete(ctx, c.ID); err != nil {
				log.Printf("[WARN] sweep: deleting cluster %s (%s): %v", c.Name, c.ID, err)
				continue
			}
			log.Printf("[INFO] sweep: deleted leaked cluster %s (%s)", c.Name, c.ID)
			swept++
		}

		if len(page) < pageSize {
			break
		}
		offset += pageSize
	}

	log.Printf("[INFO] sweep: removed %d leaked kubernetes clusters", swept)
	return nil
}
