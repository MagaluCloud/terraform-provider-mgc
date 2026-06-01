package utils

import (
	sdk "github.com/MagaluCloud/mgc-sdk-go/client"
)

const (
	ServiceBlockStorage      = "block_storage"
	ServiceContainerRegistry = "container_registry"
	ServiceDatabase          = "database"
	ServiceKubernetes        = "kubernetes"
	ServiceLbaas             = "lbaas"
	ServiceNetwork           = "network"
	ServiceObjectStorage     = "object_storage"
	ServicePlatform          = "platform"
	ServiceSSH               = "ssh"
	ServiceVirtualMachine    = "virtual_machine"
)

type DataConfig struct {
	ApiKey           string
	Env              string
	Region           string
	KeyPairID        string
	KeyPairSecret    string
	CoreConfig       sdk.CoreClient
	serviceEndpoints map[string]string
}

// SetServiceEndpoints stores custom base URL overrides keyed by service name constant.
func (d *DataConfig) SetServiceEndpoints(endpoints map[string]string) {
	d.serviceEndpoints = endpoints
}

// CoreFor returns a *sdk.CoreClient configured with a custom base URL for the
// given service when one is set; otherwise it returns a pointer to the default CoreConfig.
func (d *DataConfig) CoreFor(service string) *sdk.CoreClient {
	if len(d.serviceEndpoints) == 0 {
		return &d.CoreConfig
	}
	url, ok := d.serviceEndpoints[service]
	if !ok || url == "" {
		return &d.CoreConfig
	}
	cfg := d.CoreConfig.GetConfig()
	return sdk.NewMgcClient(
		sdk.WithAPIKey(cfg.APIKey),
		sdk.WithBaseURL(sdk.MgcUrl(url)),
		sdk.WithUserAgent(cfg.UserAgent),
		sdk.WithHTTPClient(cfg.HTTPClient),
		sdk.WithTimeout(cfg.Timeout),
		sdk.WithRetryConfig(
			cfg.RetryConfig.MaxAttempts,
			cfg.RetryConfig.InitialInterval,
			cfg.RetryConfig.MaxInterval,
			cfg.RetryConfig.BackoffFactor,
		),
	)
}

// ObjectStorageEndpoint returns the custom S3 endpoint URL and true when a
// custom endpoint is configured for object storage; otherwise returns "", false.
func (d *DataConfig) ObjectStorageEndpoint() (string, bool) {
	if len(d.serviceEndpoints) == 0 {
		return "", false
	}
	url, ok := d.serviceEndpoints[ServiceObjectStorage]
	return url, ok && url != ""
}
