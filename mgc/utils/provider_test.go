package utils

import (
	"testing"

	sdk "github.com/MagaluCloud/mgc-sdk-go/client"
	"github.com/stretchr/testify/assert"
)

func TestCoreForPreservesClientConfig(t *testing.T) {
	d := DataConfig{
		CoreConfig: *sdk.NewMgcClient(
			sdk.WithAPIKey("api-key"),
			sdk.WithJWToken("token"),
			sdk.WithCustomHeader("X-Custom", "value"),
			sdk.WithUserAgent("MgcTF/test"),
		),
	}
	d.SetServiceEndpoints(map[string]string{ServiceKubernetes: "https://localhost:8080"})

	cfg := d.CoreFor(ServiceKubernetes).GetConfig()

	assert.Equal(t, sdk.MgcUrl("https://localhost:8080"), cfg.BaseURL)
	assert.Equal(t, "api-key", cfg.APIKey)
	assert.Equal(t, "Bearer token", cfg.JWToken)
	assert.Equal(t, map[string]string{"X-Custom": "value"}, cfg.CustomHeaders)
	assert.Equal(t, "MgcTF/test", cfg.UserAgent)
	assert.Same(t, d.CoreConfig.GetConfig().Logger, cfg.Logger)
}

func TestCoreForWithoutCustomEndpointReturnsSharedClient(t *testing.T) {
	d := DataConfig{CoreConfig: *sdk.NewMgcClient(sdk.WithAPIKey("api-key"))}

	assert.Same(t, &d.CoreConfig, d.CoreFor(ServiceKubernetes))
}
