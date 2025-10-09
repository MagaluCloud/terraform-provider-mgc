package lbaas

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	lbSDK "github.com/MagaluCloud/mgc-sdk-go/lbaas"
)

func TestLoadBalancerModel_ConvertACLsToSDK(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *[]ACLModel
		expected []lbSDK.CreateNetworkACLRequest
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty slice",
			input:    &[]ACLModel{},
			expected: nil,
		},
		{
			name: "single ACL with all fields",
			input: &[]ACLModel{
				{
					Action:         types.StringValue("ALLOW"),
					Name:           types.StringValue("test-acl"),
					Ethertype:      types.StringValue("IPv4"),
					Protocol:       types.StringValue("TCP"),
					RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
				},
			},
			expected: []lbSDK.CreateNetworkACLRequest{
				{
					Action:         lbSDK.AclActionType("ALLOW"),
					Name:           stringPtr("test-acl"),
					Ethertype:      lbSDK.AclEtherType("IPv4"),
					Protocol:       lbSDK.AclProtocol("TCP"),
					RemoteIPPrefix: "192.168.1.0/24",
				},
			},
		},
		{
			name: "multiple ACLs",
			input: &[]ACLModel{
				{
					Action:         types.StringValue("ALLOW"),
					Name:           types.StringValue("acl-1"),
					Ethertype:      types.StringValue("IPv4"),
					Protocol:       types.StringValue("TCP"),
					RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
				},
				{
					Action:         types.StringValue("DENY"),
					Name:           types.StringValue("acl-2"),
					Ethertype:      types.StringValue("IPv6"),
					Protocol:       types.StringValue("UDP"),
					RemoteIPPrefix: types.StringValue("10.0.0.0/8"),
				},
			},
			expected: []lbSDK.CreateNetworkACLRequest{
				{
					Action:         lbSDK.AclActionType("ALLOW"),
					Name:           stringPtr("acl-1"),
					Ethertype:      lbSDK.AclEtherType("IPv4"),
					Protocol:       lbSDK.AclProtocol("TCP"),
					RemoteIPPrefix: "192.168.1.0/24",
				},
				{
					Action:         lbSDK.AclActionType("DENY"),
					Name:           stringPtr("acl-2"),
					Ethertype:      lbSDK.AclEtherType("IPv6"),
					Protocol:       lbSDK.AclProtocol("UDP"),
					RemoteIPPrefix: "10.0.0.0/8",
				},
			},
		},
		{
			name: "ACL with null name",
			input: &[]ACLModel{
				{
					Action:         types.StringValue("ALLOW"),
					Name:           types.StringNull(),
					Ethertype:      types.StringValue("IPv4"),
					Protocol:       types.StringValue("TCP"),
					RemoteIPPrefix: types.StringValue("0.0.0.0/0"),
				},
			},
			expected: []lbSDK.CreateNetworkACLRequest{
				{
					Action:         lbSDK.AclActionType("ALLOW"),
					Name:           nil,
					Ethertype:      lbSDK.AclEtherType("IPv4"),
					Protocol:       lbSDK.AclProtocol("TCP"),
					RemoteIPPrefix: "0.0.0.0/0",
				},
			},
		},
		{
			name: "ACL with empty string values",
			input: &[]ACLModel{
				{
					Action:         types.StringValue(""),
					Name:           types.StringValue(""),
					Ethertype:      types.StringValue(""),
					Protocol:       types.StringValue(""),
					RemoteIPPrefix: types.StringValue(""),
				},
			},
			expected: []lbSDK.CreateNetworkACLRequest{
				{
					Action:         lbSDK.AclActionType(""),
					Name:           stringPtr(""),
					Ethertype:      lbSDK.AclEtherType(""),
					Protocol:       lbSDK.AclProtocol(""),
					RemoteIPPrefix: "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &LoadBalancerModel{ACLs: tt.input}
			result := model.ConvertACLsToSDK()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadBalancerModel_ConvertBackendsToSDK(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []BackendModel
		expected []lbSDK.CreateNetworkBackendRequest
	}{
		{
			name:     "empty slice",
			input:    []BackendModel{},
			expected: nil,
		},
		{
			name: "single backend with all fields",
			input: []BackendModel{
				{
					Name:                                types.StringValue("test-backend"),
					Description:                         types.StringValue("Test backend description"),
					HealthCheckName:                     types.StringValue("health-check-1"),
					PanicThreshold:                      types.Float64Value(0.8),
					CloseConnectionsOnHostHealthFailure: types.BoolValue(true),
					BalanceAlgorithm:                    types.StringValue("ROUND_ROBIN"),
					TargetsType:                         types.StringValue("INSTANCE"),
					Targets: []TargetModel{
						{
							NICID:     types.StringValue("nic-123"),
							IPAddress: types.StringValue("192.168.1.10"),
							Port:      types.Int64Value(8080),
						},
					},
				},
			},
			expected: []lbSDK.CreateNetworkBackendRequest{
				{
					Name:                                "test-backend",
					Description:                         stringPtr("Test backend description"),
					HealthCheckName:                     stringPtr("health-check-1"),
					PanicThreshold:                      float64Ptr(0.8),
					CloseConnectionsOnHostHealthFailure: boolPtr(true),
					BalanceAlgorithm:                    lbSDK.BackendBalanceAlgorithm("ROUND_ROBIN"),
					TargetsType:                         lbSDK.BackendType("INSTANCE"),
					Targets: &[]lbSDK.NetworkBackendInstanceTargetRequest{
						{
							NicID:     stringPtr("nic-123"),
							IPAddress: stringPtr("192.168.1.10"),
							Port:      8080,
						},
					},
				},
			},
		},
		{
			name: "backend with multiple targets",
			input: []BackendModel{
				{
					Name:             types.StringValue("multi-target-backend"),
					BalanceAlgorithm: types.StringValue("LEAST_CONNECTIONS"),
					TargetsType:      types.StringValue("INSTANCE"),
					Targets: []TargetModel{
						{
							NICID:     types.StringValue("nic-1"),
							IPAddress: types.StringValue("192.168.1.10"),
							Port:      types.Int64Value(8080),
						},
						{
							NICID:     types.StringValue("nic-2"),
							IPAddress: types.StringValue("192.168.1.20"),
							Port:      types.Int64Value(8081),
						},
					},
				},
			},
			expected: []lbSDK.CreateNetworkBackendRequest{
				{
					Name:             "multi-target-backend",
					BalanceAlgorithm: lbSDK.BackendBalanceAlgorithm("LEAST_CONNECTIONS"),
					TargetsType:      lbSDK.BackendType("INSTANCE"),
					Targets: &[]lbSDK.NetworkBackendInstanceTargetRequest{
						{
							NicID:     stringPtr("nic-1"),
							IPAddress: stringPtr("192.168.1.10"),
							Port:      8080,
						},
						{
							NicID:     stringPtr("nic-2"),
							IPAddress: stringPtr("192.168.1.20"),
							Port:      8081,
						},
					},
				},
			},
		},
		{
			name: "backend with null optional fields",
			input: []BackendModel{
				{
					Name:                                types.StringValue("minimal-backend"),
					Description:                         types.StringNull(),
					HealthCheckName:                     types.StringNull(),
					PanicThreshold:                      types.Float64Null(),
					CloseConnectionsOnHostHealthFailure: types.BoolNull(),
					BalanceAlgorithm:                    types.StringValue("ROUND_ROBIN"),
					TargetsType:                         types.StringValue("INSTANCE"),
					Targets:                             []TargetModel{},
				},
			},
			expected: []lbSDK.CreateNetworkBackendRequest{
				{
					Name:                                "minimal-backend",
					Description:                         nil,
					HealthCheckName:                     nil,
					PanicThreshold:                      nil,
					CloseConnectionsOnHostHealthFailure: nil,
					BalanceAlgorithm:                    lbSDK.BackendBalanceAlgorithm("ROUND_ROBIN"),
					TargetsType:                         lbSDK.BackendType("INSTANCE"),
					Targets:                             &[]lbSDK.NetworkBackendInstanceTargetRequest{},
				},
			},
		},
		{
			name: "target with null optional fields",
			input: []BackendModel{
				{
					Name:             types.StringValue("backend-null-targets"),
					BalanceAlgorithm: types.StringValue("ROUND_ROBIN"),
					TargetsType:      types.StringValue("INSTANCE"),
					Targets: []TargetModel{
						{
							NICID:     types.StringNull(),
							IPAddress: types.StringNull(),
							Port:      types.Int64Value(80),
						},
					},
				},
			},
			expected: []lbSDK.CreateNetworkBackendRequest{
				{
					Name:             "backend-null-targets",
					BalanceAlgorithm: lbSDK.BackendBalanceAlgorithm("ROUND_ROBIN"),
					TargetsType:      lbSDK.BackendType("INSTANCE"),
					Targets: &[]lbSDK.NetworkBackendInstanceTargetRequest{
						{
							NicID:     nil,
							IPAddress: nil,
							Port:      80,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &LoadBalancerModel{Backends: tt.input}
			result := model.ConvertBackendsToSDK()

			// Compare length first
			assert.Equal(t, len(tt.expected), len(result))

			// Compare each backend individually
			for i, expectedBackend := range tt.expected {
				if i < len(result) {
					actualBackend := result[i]

					// Compare all fields except Targets
					assert.Equal(t, expectedBackend.Name, actualBackend.Name)
					assert.Equal(t, expectedBackend.Description, actualBackend.Description)
					assert.Equal(t, expectedBackend.HealthCheckName, actualBackend.HealthCheckName)
					assert.Equal(t, expectedBackend.PanicThreshold, actualBackend.PanicThreshold)
					assert.Equal(t, expectedBackend.CloseConnectionsOnHostHealthFailure, actualBackend.CloseConnectionsOnHostHealthFailure)
					assert.Equal(t, expectedBackend.BalanceAlgorithm, actualBackend.BalanceAlgorithm)
					assert.Equal(t, expectedBackend.TargetsType, actualBackend.TargetsType)

					// Compare Targets content
					if expectedBackend.Targets != nil && actualBackend.Targets != nil {
						assert.Equal(t, len(*expectedBackend.Targets), len(*actualBackend.Targets))
						for j, expectedTarget := range *expectedBackend.Targets {
							if j < len(*actualBackend.Targets) {
								actualTarget := (*actualBackend.Targets)[j]
								assert.Equal(t, expectedTarget, actualTarget)
							}
						}
					} else {
						assert.Equal(t, expectedBackend.Targets == nil, actualBackend.Targets == nil)
					}
				}
			}
		})
	}
}

func TestLoadBalancerModel_ConvertHealthChecksToSDK(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *[]HealthCheckModel
		expected []lbSDK.CreateNetworkHealthCheckRequest
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty slice",
			input:    &[]HealthCheckModel{},
			expected: nil,
		},
		{
			name: "single health check with all fields",
			input: &[]HealthCheckModel{
				{
					Name:                    types.StringValue("test-health-check"),
					Description:             types.StringValue("Test health check"),
					Protocol:                types.StringValue("HTTP"),
					Port:                    types.Int64Value(80),
					Path:                    types.StringValue("/health"),
					HealthyStatusCode:       types.Int64Value(200),
					IntervalSeconds:         types.Int64Value(30),
					TimeoutSeconds:          types.Int64Value(5),
					InitialDelaySeconds:     types.Int64Value(10),
					HealthyThresholdCount:   types.Int64Value(3),
					UnhealthyThresholdCount: types.Int64Value(2),
				},
			},
			expected: []lbSDK.CreateNetworkHealthCheckRequest{
				{
					Name:                    "test-health-check",
					Description:             stringPtr("Test health check"),
					Protocol:                lbSDK.HealthCheckProtocol("HTTP"),
					Port:                    80,
					Path:                    stringPtr("/health"),
					HealthyStatusCode:       intPtr(200),
					IntervalSeconds:         intPtr(30),
					TimeoutSeconds:          intPtr(5),
					InitialDelaySeconds:     intPtr(10),
					HealthyThresholdCount:   intPtr(3),
					UnhealthyThresholdCount: intPtr(2),
				},
			},
		},
		{
			name: "health check with null optional fields",
			input: &[]HealthCheckModel{
				{
					Name:                    types.StringValue("minimal-health-check"),
					Description:             types.StringNull(),
					Protocol:                types.StringValue("TCP"),
					Port:                    types.Int64Value(443),
					Path:                    types.StringNull(),
					HealthyStatusCode:       types.Int64Null(),
					IntervalSeconds:         types.Int64Null(),
					TimeoutSeconds:          types.Int64Null(),
					InitialDelaySeconds:     types.Int64Null(),
					HealthyThresholdCount:   types.Int64Null(),
					UnhealthyThresholdCount: types.Int64Null(),
				},
			},
			expected: []lbSDK.CreateNetworkHealthCheckRequest{
				{
					Name:                    "minimal-health-check",
					Description:             nil,
					Protocol:                lbSDK.HealthCheckProtocol("TCP"),
					Port:                    443,
					Path:                    nil,
					HealthyStatusCode:       nil,
					IntervalSeconds:         nil,
					TimeoutSeconds:          nil,
					InitialDelaySeconds:     nil,
					HealthyThresholdCount:   nil,
					UnhealthyThresholdCount: nil,
				},
			},
		},
		{
			name: "multiple health checks",
			input: &[]HealthCheckModel{
				{
					Name:     types.StringValue("http-check"),
					Protocol: types.StringValue("HTTP"),
					Port:     types.Int64Value(80),
					Path:     types.StringValue("/"),
				},
				{
					Name:     types.StringValue("tcp-check"),
					Protocol: types.StringValue("TCP"),
					Port:     types.Int64Value(443),
				},
			},
			expected: []lbSDK.CreateNetworkHealthCheckRequest{
				{
					Name:     "http-check",
					Protocol: lbSDK.HealthCheckProtocol("HTTP"),
					Port:     80,
					Path:     stringPtr("/"),
				},
				{
					Name:     "tcp-check",
					Protocol: lbSDK.HealthCheckProtocol("TCP"),
					Port:     443,
				},
			},
		},
		{
			name: "health check with zero values",
			input: &[]HealthCheckModel{
				{
					Name:                    types.StringValue("zero-values"),
					Protocol:                types.StringValue("HTTP"),
					Port:                    types.Int64Value(0),
					HealthyStatusCode:       types.Int64Value(0),
					IntervalSeconds:         types.Int64Value(0),
					TimeoutSeconds:          types.Int64Value(0),
					InitialDelaySeconds:     types.Int64Value(0),
					HealthyThresholdCount:   types.Int64Value(0),
					UnhealthyThresholdCount: types.Int64Value(0),
				},
			},
			expected: []lbSDK.CreateNetworkHealthCheckRequest{
				{
					Name:                    "zero-values",
					Protocol:                lbSDK.HealthCheckProtocol("HTTP"),
					Port:                    0,
					HealthyStatusCode:       intPtr(0),
					IntervalSeconds:         intPtr(0),
					TimeoutSeconds:          intPtr(0),
					InitialDelaySeconds:     intPtr(0),
					HealthyThresholdCount:   intPtr(0),
					UnhealthyThresholdCount: intPtr(0),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &LoadBalancerModel{HealthChecks: tt.input}
			result := model.ConvertHealthChecksToSDK()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadBalancerModel_ConvertListenersToSDK(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []ListenerModel
		expected []lbSDK.NetworkListenerRequest
	}{
		{
			name:     "empty slice",
			input:    []ListenerModel{},
			expected: nil,
		},
		{
			name: "single listener with all fields",
			input: []ListenerModel{
				{
					Name:               types.StringValue("test-listener"),
					Description:        types.StringValue("Test listener description"),
					Port:               types.Int64Value(443),
					Protocol:           types.StringValue("HTTPS"),
					TLSCertificateName: types.StringValue("test-cert"),
					BackendName:        types.StringValue("test-backend"),
				},
			},
			expected: []lbSDK.NetworkListenerRequest{
				{
					Name:               "test-listener",
					Description:        stringPtr("Test listener description"),
					Port:               443,
					Protocol:           lbSDK.ListenerProtocol("HTTPS"),
					TLSCertificateName: stringPtr("test-cert"),
					BackendName:        "test-backend",
				},
			},
		},
		{
			name: "listener with null optional fields",
			input: []ListenerModel{
				{
					Name:               types.StringValue("minimal-listener"),
					Description:        types.StringNull(),
					Port:               types.Int64Value(80),
					Protocol:           types.StringValue("HTTP"),
					TLSCertificateName: types.StringNull(),
					BackendName:        types.StringValue("backend-1"),
				},
			},
			expected: []lbSDK.NetworkListenerRequest{
				{
					Name:               "minimal-listener",
					Description:        nil,
					Port:               80,
					Protocol:           lbSDK.ListenerProtocol("HTTP"),
					TLSCertificateName: nil,
					BackendName:        "backend-1",
				},
			},
		},
		{
			name: "multiple listeners",
			input: []ListenerModel{
				{
					Name:        types.StringValue("http-listener"),
					Port:        types.Int64Value(80),
					Protocol:    types.StringValue("HTTP"),
					BackendName: types.StringValue("web-backend"),
				},
				{
					Name:               types.StringValue("https-listener"),
					Port:               types.Int64Value(443),
					Protocol:           types.StringValue("HTTPS"),
					TLSCertificateName: types.StringValue("ssl-cert"),
					BackendName:        types.StringValue("secure-backend"),
				},
			},
			expected: []lbSDK.NetworkListenerRequest{
				{
					Name:        "http-listener",
					Port:        80,
					Protocol:    lbSDK.ListenerProtocol("HTTP"),
					BackendName: "web-backend",
				},
				{
					Name:               "https-listener",
					Port:               443,
					Protocol:           lbSDK.ListenerProtocol("HTTPS"),
					TLSCertificateName: stringPtr("ssl-cert"),
					BackendName:        "secure-backend",
				},
			},
		},
		{
			name: "listener with zero port",
			input: []ListenerModel{
				{
					Name:        types.StringValue("zero-port"),
					Port:        types.Int64Value(0),
					Protocol:    types.StringValue("TCP"),
					BackendName: types.StringValue("backend"),
				},
			},
			expected: []lbSDK.NetworkListenerRequest{
				{
					Name:        "zero-port",
					Port:        0,
					Protocol:    lbSDK.ListenerProtocol("TCP"),
					BackendName: "backend",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &LoadBalancerModel{Listeners: tt.input}
			result := model.ConvertListenersToSDK()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadBalancerModel_ConvertTLSCertificatesToSDK(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *[]TLSCertificateModel
		expected []lbSDK.CreateNetworkCertificateRequest
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty slice",
			input:    &[]TLSCertificateModel{},
			expected: nil,
		},
		{
			name: "single certificate with all fields",
			input: &[]TLSCertificateModel{
				{
					Name:        types.StringValue("test-cert"),
					Description: types.StringValue("Test certificate"),
					Certificate: types.StringValue("-----BEGIN CERTIFICATE-----\nMIIC..."),
					PrivateKey:  types.StringValue("-----BEGIN PRIVATE KEY-----\nMIIE..."),
				},
			},
			expected: []lbSDK.CreateNetworkCertificateRequest{
				{
					Name:        "test-cert",
					Description: stringPtr("Test certificate"),
					Certificate: "-----BEGIN CERTIFICATE-----\nMIIC...",
					PrivateKey:  "-----BEGIN PRIVATE KEY-----\nMIIE...",
				},
			},
		},
		{
			name: "certificate with null description",
			input: &[]TLSCertificateModel{
				{
					Name:        types.StringValue("minimal-cert"),
					Description: types.StringNull(),
					Certificate: types.StringValue("cert-data"),
					PrivateKey:  types.StringValue("key-data"),
				},
			},
			expected: []lbSDK.CreateNetworkCertificateRequest{
				{
					Name:        "minimal-cert",
					Description: nil,
					Certificate: "cert-data",
					PrivateKey:  "key-data",
				},
			},
		},
		{
			name: "multiple certificates",
			input: &[]TLSCertificateModel{
				{
					Name:        types.StringValue("cert-1"),
					Certificate: types.StringValue("cert-data-1"),
					PrivateKey:  types.StringValue("key-data-1"),
				},
				{
					Name:        types.StringValue("cert-2"),
					Description: types.StringValue("Second certificate"),
					Certificate: types.StringValue("cert-data-2"),
					PrivateKey:  types.StringValue("key-data-2"),
				},
			},
			expected: []lbSDK.CreateNetworkCertificateRequest{
				{
					Name:        "cert-1",
					Certificate: "cert-data-1",
					PrivateKey:  "key-data-1",
				},
				{
					Name:        "cert-2",
					Description: stringPtr("Second certificate"),
					Certificate: "cert-data-2",
					PrivateKey:  "key-data-2",
				},
			},
		},
		{
			name: "certificate with empty strings",
			input: &[]TLSCertificateModel{
				{
					Name:        types.StringValue(""),
					Description: types.StringValue(""),
					Certificate: types.StringValue(""),
					PrivateKey:  types.StringValue(""),
				},
			},
			expected: []lbSDK.CreateNetworkCertificateRequest{
				{
					Name:        "",
					Description: stringPtr(""),
					Certificate: "",
					PrivateKey:  "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &LoadBalancerModel{TLSCertificates: tt.input}
			result := model.ConvertTLSCertificatesToSDK()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadBalancerModel_ToTerraformNetworkResource(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name     string
		input    lbSDK.NetworkLoadBalancerResponse
		expected LoadBalancerModel
	}{
		{
			name: "complete load balancer",
			input: lbSDK.NetworkLoadBalancerResponse{
				ID:           "lb-123",
				Name:         "test-lb",
				Description:  stringPtr("Test load balancer"),
				Type:         "APPLICATION",
				Visibility:   lbSDK.LoadBalancerVisibility("PUBLIC"),
				VPCID:        "vpc-123",
				SubnetPoolID: stringPtr("subnet-pool-123"),
				PublicIP:     &lbSDK.NetworkPublicIPResponse{ExternalID: "public-ip-123"},
				HealthChecks: []lbSDK.NetworkHealthCheckResponse{
					{
						ID:                      "hc-123",
						Name:                    "test-hc",
						Description:             stringPtr("Test health check"),
						Protocol:                lbSDK.HealthCheckProtocol("HTTP"),
						Port:                    80,
						Path:                    stringPtr("/health"),
						IntervalSeconds:         30,
						TimeoutSeconds:          5,
						HealthyStatusCode:       200,
						HealthyThresholdCount:   3,
						InitialDelaySeconds:     10,
						UnhealthyThresholdCount: 2,
					},
				},
				ACLs: []lbSDK.NetworkAclResponse{
					{
						ID:             "acl-123",
						Action:         "ALLOW",
						Ethertype:      lbSDK.AclEtherType("IPv4"),
						Protocol:       lbSDK.AclProtocol("TCP"),
						Name:           stringPtr("test-acl"),
						RemoteIPPrefix: "0.0.0.0/0",
					},
				},
				Backends: []lbSDK.NetworkBackendResponse{
					{
						ID:                                  "backend-123",
						Name:                                "test-backend",
						Description:                         stringPtr("Test backend"),
						BalanceAlgorithm:                    lbSDK.BackendBalanceAlgorithm("ROUND_ROBIN"),
						HealthCheckID:                       stringPtr("hc-123"),
						PanicThreshold:                      float64Ptr(0.8),
						CloseConnectionsOnHostHealthFailure: boolPtr(true),
						TargetsType:                         lbSDK.BackendType("INSTANCE"),
						Targets: []lbSDK.NetworkBackedTarget{
							{
								ID:        "target-123",
								Port:      int64Ptr(8080),
								NicID:     stringPtr("nic-123"),
								IPAddress: stringPtr("192.168.1.10"),
							},
						},
					},
				},
				TLSCertificates: []lbSDK.NetworkTLSCertificateResponse{
					{
						ID:          "cert-123",
						Name:        "test-cert",
						Description: stringPtr("Test certificate"),
					},
				},
				Listeners: []lbSDK.NetworkListenerResponse{
					{
						ID:               "listener-123",
						Name:             "test-listener",
						Description:      stringPtr("Test listener"),
						Port:             443,
						Protocol:         lbSDK.ListenerProtocol("HTTPS"),
						BackendID:        "backend-123",
						TLSCertificateID: stringPtr("cert-123"),
					},
				},
			},
			expected: LoadBalancerModel{
				ID:           types.StringValue("lb-123"),
				Name:         types.StringValue("test-lb"),
				Description:  types.StringValue("Test load balancer"),
				PublicIPID:   types.StringValue("public-ip-123"),
				SubnetpoolID: types.StringValue("subnet-pool-123"),
				Type:         types.StringValue("APPLICATION"),
				Visibility:   types.StringValue("PUBLIC"),
				VPCID:        types.StringValue("vpc-123"),
				ACLs: &[]ACLModel{
					{
						Action:         types.StringValue("ALLOW"),
						Ethertype:      types.StringValue("IPv4"),
						Protocol:       types.StringValue("TCP"),
						Name:           types.StringValue("test-acl"),
						RemoteIPPrefix: types.StringValue("0.0.0.0/0"),
					},
				},
				Backends: []BackendModel{
					{
						ID:                                  types.StringValue("backend-123"),
						Name:                                types.StringValue("test-backend"),
						Description:                         types.StringValue("Test backend"),
						BalanceAlgorithm:                    types.StringValue("ROUND_ROBIN"),
						HealthCheckName:                     types.StringValue("test-hc"),
						PanicThreshold:                      types.Float64Value(0.8),
						CloseConnectionsOnHostHealthFailure: types.BoolValue(true),
						TargetsType:                         types.StringValue("INSTANCE"),
						Targets: []TargetModel{
							{
								Port:      types.Int64Value(8080),
								NICID:     types.StringValue("nic-123"),
								IPAddress: types.StringValue("192.168.1.10"),
							},
						},
					},
				},
				HealthChecks: &[]HealthCheckModel{
					{
						ID:                      types.StringValue("hc-123"),
						Name:                    types.StringValue("test-hc"),
						Description:             types.StringValue("Test health check"),
						Protocol:                types.StringValue("HTTP"),
						Port:                    types.Int64Value(80),
						Path:                    types.StringValue("/health"),
						IntervalSeconds:         types.Int64Value(30),
						TimeoutSeconds:          types.Int64Value(5),
						HealthyStatusCode:       types.Int64Value(200),
						HealthyThresholdCount:   types.Int64Value(3),
						InitialDelaySeconds:     types.Int64Value(10),
						UnhealthyThresholdCount: types.Int64Value(2),
					},
				},
				Listeners: []ListenerModel{
					{
						ID:                 types.StringValue("listener-123"),
						Name:               types.StringValue("test-listener"),
						Description:        types.StringValue("Test listener"),
						Port:               types.Int64Value(443),
						Protocol:           types.StringValue("HTTPS"),
						BackendName:        types.StringValue("test-backend"),
						TLSCertificateName: types.StringValue("test-cert"),
					},
				},
				TLSCertificates: &[]TLSCertificateModel{
					{
						ID:             types.StringValue("cert-123"),
						Name:           types.StringValue("test-cert"),
						Description:    types.StringValue("Test certificate"),
						Certificate:    types.StringNull(),
						PrivateKey:     types.StringNull(),
						ExpirationDate: types.StringValue(""),
					},
				},
			},
		},
		{
			name: "minimal load balancer",
			input: lbSDK.NetworkLoadBalancerResponse{
				ID:              "lb-456",
				Name:            "minimal-lb",
				Type:            "NETWORK",
				Visibility:      lbSDK.LoadBalancerVisibility("PRIVATE"),
				VPCID:           "vpc-456",
				HealthChecks:    []lbSDK.NetworkHealthCheckResponse{},
				ACLs:            []lbSDK.NetworkAclResponse{},
				Backends:        []lbSDK.NetworkBackendResponse{},
				TLSCertificates: []lbSDK.NetworkTLSCertificateResponse{},
				Listeners:       []lbSDK.NetworkListenerResponse{},
				PublicIP:        &lbSDK.NetworkPublicIPResponse{},
			},
			expected: LoadBalancerModel{
				ID:              types.StringValue("lb-456"),
				Name:            types.StringValue("minimal-lb"),
				Description:     types.StringNull(),
				PublicIPID:      types.StringNull(),
				SubnetpoolID:    types.StringNull(),
				Type:            types.StringValue("NETWORK"),
				Visibility:      types.StringValue("PRIVATE"),
				VPCID:           types.StringValue("vpc-456"),
				ACLs:            &[]ACLModel{},
				Backends:        []BackendModel{},
				HealthChecks:    &[]HealthCheckModel{},
				Listeners:       []ListenerModel{},
				TLSCertificates: &[]TLSCertificateModel{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: The original implementation has a bug with uninitialized maps
			// This test will help identify that issue
			model := &LoadBalancerModel{}
			result := model.ToTerraformNetworkResource(ctx, tt.input)

			// Compare each field individually for better error messages
			assert.Equal(t, tt.expected.ID, result.ID)
			assert.Equal(t, tt.expected.Name, result.Name)
			assert.Equal(t, tt.expected.Description, result.Description)
			assert.Equal(t, tt.expected.PublicIPID, result.PublicIPID)
			assert.Equal(t, tt.expected.SubnetpoolID, result.SubnetpoolID)
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.Visibility, result.Visibility)
			assert.Equal(t, tt.expected.VPCID, result.VPCID)

			// Compare slices
			require.NotNil(t, result.ACLs)
			assert.Equal(t, len(*tt.expected.ACLs), len(*result.ACLs))
			for i, expectedACL := range *tt.expected.ACLs {
				if i < len(*result.ACLs) {
					actualACL := (*result.ACLs)[i]
					assert.Equal(t, expectedACL, actualACL)
				}
			}

			assert.Equal(t, len(tt.expected.Backends), len(result.Backends))
			for i, expectedBackend := range tt.expected.Backends {
				if i < len(result.Backends) {
					actualBackend := result.Backends[i]
					assert.Equal(t, expectedBackend, actualBackend)
				}
			}

			require.NotNil(t, result.HealthChecks)
			assert.Equal(t, len(*tt.expected.HealthChecks), len(*result.HealthChecks))

			assert.Equal(t, len(tt.expected.Listeners), len(result.Listeners))

			require.NotNil(t, result.TLSCertificates)
			if tt.expected.TLSCertificates != nil {
				assert.Equal(t, len(*tt.expected.TLSCertificates), len(*result.TLSCertificates))
			}
		})
	}
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func int64Ptr(i int64) *int64 {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func boolPtr(b bool) *bool {
	return &b
}

// Additional edge case tests for better coverage
func TestLoadBalancerModel_ConvertACLsToSDK_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *[]ACLModel
		expected []lbSDK.CreateNetworkACLRequest
	}{
		{
			name: "large slice of ACLs",
			input: func() *[]ACLModel {
				acls := make([]ACLModel, 100)
				for i := 0; i < 100; i++ {
					acls[i] = ACLModel{
						Action:         types.StringValue("ALLOW"),
						Name:           types.StringValue(fmt.Sprintf("acl-%d", i)),
						Ethertype:      types.StringValue("IPv4"),
						Protocol:       types.StringValue("TCP"),
						RemoteIPPrefix: types.StringValue("0.0.0.0/0"),
					}
				}
				return &acls
			}(),
			expected: func() []lbSDK.CreateNetworkACLRequest {
				acls := make([]lbSDK.CreateNetworkACLRequest, 100)
				for i := 0; i < 100; i++ {
					acls[i] = lbSDK.CreateNetworkACLRequest{
						Action:         lbSDK.AclActionType("ALLOW"),
						Name:           stringPtr(fmt.Sprintf("acl-%d", i)),
						Ethertype:      lbSDK.AclEtherType("IPv4"),
						Protocol:       lbSDK.AclProtocol("TCP"),
						RemoteIPPrefix: "0.0.0.0/0",
					}
				}
				return acls
			}(),
		},
		{
			name: "ACL with special characters in name",
			input: &[]ACLModel{
				{
					Action:         types.StringValue("DENY"),
					Name:           types.StringValue("acl-with-special-chars-!@#$%^&*()"),
					Ethertype:      types.StringValue("IPv6"),
					Protocol:       types.StringValue("UDP"),
					RemoteIPPrefix: types.StringValue("::/0"),
				},
			},
			expected: []lbSDK.CreateNetworkACLRequest{
				{
					Action:         lbSDK.AclActionType("DENY"),
					Name:           stringPtr("acl-with-special-chars-!@#$%^&*()"),
					Ethertype:      lbSDK.AclEtherType("IPv6"),
					Protocol:       lbSDK.AclProtocol("UDP"),
					RemoteIPPrefix: "::/0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &LoadBalancerModel{ACLs: tt.input}
			result := model.ConvertACLsToSDK()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadBalancerModel_ConvertBackendsToSDK_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []BackendModel
		expected []lbSDK.CreateNetworkBackendRequest
	}{
		{
			name: "backend with extreme values",
			input: []BackendModel{
				{
					Name:                                types.StringValue("extreme-backend"),
					PanicThreshold:                      types.Float64Value(1.0),
					CloseConnectionsOnHostHealthFailure: types.BoolValue(false),
					BalanceAlgorithm:                    types.StringValue("WEIGHTED_ROUND_ROBIN"),
					TargetsType:                         types.StringValue("IP"),
					Targets: []TargetModel{
						{
							Port:      types.Int64Value(65535), // Max port
							IPAddress: types.StringValue("255.255.255.255"),
						},
						{
							Port:      types.Int64Value(1), // Min port
							IPAddress: types.StringValue("0.0.0.0"),
						},
					},
				},
			},
			expected: []lbSDK.CreateNetworkBackendRequest{
				{
					Name:                                "extreme-backend",
					PanicThreshold:                      float64Ptr(1.0),
					CloseConnectionsOnHostHealthFailure: boolPtr(false),
					BalanceAlgorithm:                    lbSDK.BackendBalanceAlgorithm("WEIGHTED_ROUND_ROBIN"),
					TargetsType:                         lbSDK.BackendType("IP"),
					Targets: &[]lbSDK.NetworkBackendInstanceTargetRequest{
						{
							Port:      65535,
							IPAddress: stringPtr("255.255.255.255"),
						},
						{
							Port:      1,
							IPAddress: stringPtr("0.0.0.0"),
						},
					},
				},
			},
		},
		{
			name: "backend with zero panic threshold",
			input: []BackendModel{
				{
					Name:             types.StringValue("zero-panic"),
					PanicThreshold:   types.Float64Value(0.0),
					BalanceAlgorithm: types.StringValue("ROUND_ROBIN"),
					TargetsType:      types.StringValue("INSTANCE"),
					Targets:          []TargetModel{},
				},
			},
			expected: []lbSDK.CreateNetworkBackendRequest{
				{
					Name:             "zero-panic",
					PanicThreshold:   float64Ptr(0.0),
					BalanceAlgorithm: lbSDK.BackendBalanceAlgorithm("ROUND_ROBIN"),
					TargetsType:      lbSDK.BackendType("INSTANCE"),
					Targets:          &[]lbSDK.NetworkBackendInstanceTargetRequest{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &LoadBalancerModel{Backends: tt.input}
			result := model.ConvertBackendsToSDK()

			// Use the same comparison logic as the main test
			assert.Equal(t, len(tt.expected), len(result))
			for i, expectedBackend := range tt.expected {
				if i < len(result) {
					actualBackend := result[i]
					assert.Equal(t, expectedBackend.Name, actualBackend.Name)
					assert.Equal(t, expectedBackend.Description, actualBackend.Description)
					assert.Equal(t, expectedBackend.HealthCheckName, actualBackend.HealthCheckName)
					assert.Equal(t, expectedBackend.PanicThreshold, actualBackend.PanicThreshold)
					assert.Equal(t, expectedBackend.CloseConnectionsOnHostHealthFailure, actualBackend.CloseConnectionsOnHostHealthFailure)
					assert.Equal(t, expectedBackend.BalanceAlgorithm, actualBackend.BalanceAlgorithm)
					assert.Equal(t, expectedBackend.TargetsType, actualBackend.TargetsType)

					if expectedBackend.Targets != nil && actualBackend.Targets != nil {
						assert.Equal(t, len(*expectedBackend.Targets), len(*actualBackend.Targets))
						for j, expectedTarget := range *expectedBackend.Targets {
							if j < len(*actualBackend.Targets) {
								actualTarget := (*actualBackend.Targets)[j]
								assert.Equal(t, expectedTarget, actualTarget)
							}
						}
					} else {
						assert.Equal(t, expectedBackend.Targets == nil, actualBackend.Targets == nil)
					}
				}
			}
		})
	}
}

func TestLoadBalancerModel_ConvertHealthChecksToSDK_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *[]HealthCheckModel
		expected []lbSDK.CreateNetworkHealthCheckRequest
	}{
		{
			name: "health check with maximum values",
			input: &[]HealthCheckModel{
				{
					Name:                    types.StringValue("max-values-hc"),
					Protocol:                types.StringValue("HTTPS"),
					Port:                    types.Int64Value(65535),
					Path:                    types.StringValue("/very/long/path/to/health/check/endpoint/that/might/be/used/in/some/applications"),
					HealthyStatusCode:       types.Int64Value(999),
					IntervalSeconds:         types.Int64Value(3600), // 1 hour
					TimeoutSeconds:          types.Int64Value(300),  // 5 minutes
					InitialDelaySeconds:     types.Int64Value(3600), // 1 hour
					HealthyThresholdCount:   types.Int64Value(100),
					UnhealthyThresholdCount: types.Int64Value(100),
				},
			},
			expected: []lbSDK.CreateNetworkHealthCheckRequest{
				{
					Name:                    "max-values-hc",
					Protocol:                lbSDK.HealthCheckProtocol("HTTPS"),
					Port:                    65535,
					Path:                    stringPtr("/very/long/path/to/health/check/endpoint/that/might/be/used/in/some/applications"),
					HealthyStatusCode:       intPtr(999),
					IntervalSeconds:         intPtr(3600),
					TimeoutSeconds:          intPtr(300),
					InitialDelaySeconds:     intPtr(3600),
					HealthyThresholdCount:   intPtr(100),
					UnhealthyThresholdCount: intPtr(100),
				},
			},
		},
		{
			name: "health check with negative values (edge case)",
			input: &[]HealthCheckModel{
				{
					Name:                    types.StringValue("negative-values"),
					Protocol:                types.StringValue("TCP"),
					Port:                    types.Int64Value(-1), // Invalid but testing conversion
					HealthyStatusCode:       types.Int64Value(-1),
					IntervalSeconds:         types.Int64Value(-1),
					TimeoutSeconds:          types.Int64Value(-1),
					InitialDelaySeconds:     types.Int64Value(-1),
					HealthyThresholdCount:   types.Int64Value(-1),
					UnhealthyThresholdCount: types.Int64Value(-1),
				},
			},
			expected: []lbSDK.CreateNetworkHealthCheckRequest{
				{
					Name:                    "negative-values",
					Protocol:                lbSDK.HealthCheckProtocol("TCP"),
					Port:                    -1,
					HealthyStatusCode:       intPtr(-1),
					IntervalSeconds:         intPtr(-1),
					TimeoutSeconds:          intPtr(-1),
					InitialDelaySeconds:     intPtr(-1),
					HealthyThresholdCount:   intPtr(-1),
					UnhealthyThresholdCount: intPtr(-1),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &LoadBalancerModel{HealthChecks: tt.input}
			result := model.ConvertHealthChecksToSDK()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadBalancerModel_ToTerraformNetworkResource_EdgeCases(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name     string
		input    lbSDK.NetworkLoadBalancerResponse
		expected LoadBalancerModel
	}{
		{
			name: "load balancer with empty string fields",
			input: lbSDK.NetworkLoadBalancerResponse{
				ID:              "",
				Name:            "",
				Type:            "",
				Visibility:      lbSDK.LoadBalancerVisibility(""),
				VPCID:           "",
				HealthChecks:    []lbSDK.NetworkHealthCheckResponse{},
				ACLs:            []lbSDK.NetworkAclResponse{},
				Backends:        []lbSDK.NetworkBackendResponse{},
				TLSCertificates: []lbSDK.NetworkTLSCertificateResponse{},
				Listeners:       []lbSDK.NetworkListenerResponse{},
				PublicIP:        &lbSDK.NetworkPublicIPResponse{},
			},
			expected: LoadBalancerModel{
				ID:              types.StringValue(""),
				Name:            types.StringValue(""),
				Description:     types.StringNull(),
				PublicIPID:      types.StringNull(),
				SubnetpoolID:    types.StringNull(),
				Type:            types.StringValue(""),
				Visibility:      types.StringValue(""),
				VPCID:           types.StringValue(""),
				ACLs:            &[]ACLModel{},
				Backends:        []BackendModel{},
				HealthChecks:    &[]HealthCheckModel{},
				Listeners:       []ListenerModel{},
				TLSCertificates: &[]TLSCertificateModel{},
			},
		},
		{
			name: "load balancer with backend without health check",
			input: lbSDK.NetworkLoadBalancerResponse{
				ID:         "lb-no-hc",
				Name:       "no-health-check-lb",
				Type:       "APPLICATION",
				Visibility: lbSDK.LoadBalancerVisibility("PUBLIC"),
				VPCID:      "vpc-123",
				Backends: []lbSDK.NetworkBackendResponse{
					{
						ID:                                  "backend-no-hc",
						Name:                                "backend-without-health-check",
						HealthCheckID:                       nil, // No health check
						BalanceAlgorithm:                    lbSDK.BackendBalanceAlgorithm("ROUND_ROBIN"),
						CloseConnectionsOnHostHealthFailure: nil,
						TargetsType:                         lbSDK.BackendType("INSTANCE"),
						Targets:                             []lbSDK.NetworkBackedTarget{},
					},
				},
				HealthChecks:    []lbSDK.NetworkHealthCheckResponse{},
				ACLs:            []lbSDK.NetworkAclResponse{},
				TLSCertificates: []lbSDK.NetworkTLSCertificateResponse{},
				Listeners:       []lbSDK.NetworkListenerResponse{},
				PublicIP:        &lbSDK.NetworkPublicIPResponse{},
			},
			expected: LoadBalancerModel{
				ID:           types.StringValue("lb-no-hc"),
				Name:         types.StringValue("no-health-check-lb"),
				Description:  types.StringNull(),
				PublicIPID:   types.StringNull(),
				SubnetpoolID: types.StringNull(),
				Type:         types.StringValue("APPLICATION"),
				Visibility:   types.StringValue("PUBLIC"),
				VPCID:        types.StringValue("vpc-123"),
				Backends: []BackendModel{
					{
						ID:                                  types.StringValue("backend-no-hc"),
						Name:                                types.StringValue("backend-without-health-check"),
						Description:                         types.StringNull(),
						BalanceAlgorithm:                    types.StringValue("ROUND_ROBIN"),
						HealthCheckName:                     types.StringNull(), // Null when no health check
						PanicThreshold:                      types.Float64Null(),
						CloseConnectionsOnHostHealthFailure: types.BoolNull(),
						TargetsType:                         types.StringValue("INSTANCE"),
						Targets:                             nil,
					},
				},
				ACLs:            &[]ACLModel{},
				HealthChecks:    &[]HealthCheckModel{},
				Listeners:       []ListenerModel{},
				TLSCertificates: &[]TLSCertificateModel{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &LoadBalancerModel{}
			result := model.ToTerraformNetworkResource(ctx, tt.input)

			// Compare main fields
			assert.Equal(t, tt.expected.ID, result.ID)
			assert.Equal(t, tt.expected.Name, result.Name)
			assert.Equal(t, tt.expected.Description, result.Description)
			assert.Equal(t, tt.expected.PublicIPID, result.PublicIPID)
			assert.Equal(t, tt.expected.SubnetpoolID, result.SubnetpoolID)
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.Visibility, result.Visibility)
			assert.Equal(t, tt.expected.VPCID, result.VPCID)

			// Compare slice lengths and content
			require.NotNil(t, result.ACLs)
			assert.Equal(t, len(*tt.expected.ACLs), len(*result.ACLs))

			assert.Equal(t, len(tt.expected.Backends), len(result.Backends))
			for i, expectedBackend := range tt.expected.Backends {
				if i < len(result.Backends) {
					actualBackend := result.Backends[i]
					assert.Equal(t, expectedBackend, actualBackend)
				}
			}

			require.NotNil(t, result.HealthChecks)
			assert.Equal(t, len(*tt.expected.HealthChecks), len(*result.HealthChecks))

			assert.Equal(t, len(tt.expected.Listeners), len(result.Listeners))

			require.NotNil(t, result.TLSCertificates)
			assert.Equal(t, len(*tt.expected.TLSCertificates), len(*result.TLSCertificates))
		})
	}
}

func TestACLModel_hasACLChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		plan     ACLModel
		state    ACLModel
		expected bool
	}{
		{
			name: "no changes - identical ACLs",
			plan: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			state: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			expected: false,
		},
		{
			name: "action changed",
			plan: ACLModel{
				Action:         types.StringValue("DENY"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			state: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			expected: true,
		},
		{
			name: "ethertype changed",
			plan: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv6"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			state: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			expected: true,
		},
		{
			name: "protocol changed",
			plan: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("UDP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			state: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			expected: true,
		},
		{
			name: "remote IP prefix changed",
			plan: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("10.0.0.0/8"),
			},
			state: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			expected: true,
		},
		{
			name: "name changed from value to different value",
			plan: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("new-acl-name"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			state: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			expected: true,
		},
		{
			name: "name changed from null to value",
			plan: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			state: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringNull(),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			expected: true,
		},
		{
			name: "name changed from value to null",
			plan: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringNull(),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			state: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			expected: true,
		},
		{
			name: "name - both null, no change",
			plan: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringNull(),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			state: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringNull(),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			expected: false,
		},
		{
			name: "name - empty string vs null, no change",
			plan: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue(""),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			state: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringNull(),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			expected: false,
		},
		{
			name: "name - both empty strings, no change",
			plan: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue(""),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			state: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue(""),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			expected: false,
		},
		{
			name: "unknown action - should return false",
			plan: ACLModel{
				Action:         types.StringUnknown(),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			state: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			expected: false,
		},
		{
			name: "unknown ethertype - should return false",
			plan: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringUnknown(),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			state: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			expected: false,
		},
		{
			name: "unknown name - should return false",
			plan: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringUnknown(),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			state: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			expected: false,
		},
		{
			name: "unknown protocol - should return false",
			plan: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringUnknown(),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			state: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			expected: false,
		},
		{
			name: "unknown remote IP prefix - should return false",
			plan: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringUnknown(),
			},
			state: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			expected: false,
		},
		{
			name: "multiple changes",
			plan: ACLModel{
				Action:         types.StringValue("DENY"),
				Ethertype:      types.StringValue("IPv6"),
				Name:           types.StringValue("new-acl"),
				Protocol:       types.StringValue("UDP"),
				RemoteIPPrefix: types.StringValue("10.0.0.0/8"),
			},
			state: ACLModel{
				Action:         types.StringValue("ALLOW"),
				Ethertype:      types.StringValue("IPv4"),
				Name:           types.StringValue("test-acl"),
				Protocol:       types.StringValue("TCP"),
				RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.plan.hasACLChanges(tt.state)
			assert.Equal(t, tt.expected, result, "hasACLChanges returned unexpected result")
		})
	}
}

func TestLoadBalancerModel_hasACLChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		plan     LoadBalancerModel
		state    LoadBalancerModel
		expected bool
	}{
		{
			name: "no ACLs in either plan or state",
			plan: LoadBalancerModel{
				ACLs: nil,
			},
			state: LoadBalancerModel{
				ACLs: nil,
			},
			expected: false,
		},
		{
			name: "no ACLs in plan, ACLs in state",
			plan: LoadBalancerModel{
				ACLs: nil,
			},
			state: LoadBalancerModel{
				ACLs: &[]ACLModel{
					{
						Action:         types.StringValue("ALLOW"),
						Ethertype:      types.StringValue("IPv4"),
						Name:           types.StringValue("test-acl"),
						Protocol:       types.StringValue("TCP"),
						RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
					},
				},
			},
			expected: true,
		},
		{
			name: "ACLs in plan, no ACLs in state",
			plan: LoadBalancerModel{
				ACLs: &[]ACLModel{
					{
						Action:         types.StringValue("ALLOW"),
						Ethertype:      types.StringValue("IPv4"),
						Name:           types.StringValue("test-acl"),
						Protocol:       types.StringValue("TCP"),
						RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
					},
				},
			},
			state: LoadBalancerModel{
				ACLs: nil,
			},
			expected: true,
		},
		{
			name: "identical ACL lists",
			plan: LoadBalancerModel{
				ACLs: &[]ACLModel{
					{
						Action:         types.StringValue("ALLOW"),
						Ethertype:      types.StringValue("IPv4"),
						Name:           types.StringValue("test-acl"),
						Protocol:       types.StringValue("TCP"),
						RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
					},
				},
			},
			state: LoadBalancerModel{
				ACLs: &[]ACLModel{
					{
						Action:         types.StringValue("ALLOW"),
						Ethertype:      types.StringValue("IPv4"),
						Name:           types.StringValue("test-acl"),
						Protocol:       types.StringValue("TCP"),
						RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
					},
				},
			},
			expected: false,
		},
		{
			name: "different number of ACLs",
			plan: LoadBalancerModel{
				ACLs: &[]ACLModel{
					{
						Action:         types.StringValue("ALLOW"),
						Ethertype:      types.StringValue("IPv4"),
						Name:           types.StringValue("test-acl-1"),
						Protocol:       types.StringValue("TCP"),
						RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
					},
					{
						Action:         types.StringValue("ALLOW"),
						Ethertype:      types.StringValue("IPv4"),
						Name:           types.StringValue("test-acl-2"),
						Protocol:       types.StringValue("TCP"),
						RemoteIPPrefix: types.StringValue("192.168.2.0/24"),
					},
				},
			},
			state: LoadBalancerModel{
				ACLs: &[]ACLModel{
					{
						Action:         types.StringValue("ALLOW"),
						Ethertype:      types.StringValue("IPv4"),
						Name:           types.StringValue("test-acl-1"),
						Protocol:       types.StringValue("TCP"),
						RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
					},
				},
			},
			expected: true,
		},
		{
			name: "ACL content changed",
			plan: LoadBalancerModel{
				ACLs: &[]ACLModel{
					{
						Action:         types.StringValue("DENY"),
						Ethertype:      types.StringValue("IPv4"),
						Name:           types.StringValue("test-acl"),
						Protocol:       types.StringValue("TCP"),
						RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
					},
				},
			},
			state: LoadBalancerModel{
				ACLs: &[]ACLModel{
					{
						Action:         types.StringValue("ALLOW"),
						Ethertype:      types.StringValue("IPv4"),
						Name:           types.StringValue("test-acl"),
						Protocol:       types.StringValue("TCP"),
						RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
					},
				},
			},
			expected: true,
		},
		{
			name: "new ACL added",
			plan: LoadBalancerModel{
				ACLs: &[]ACLModel{
					{
						Action:         types.StringValue("ALLOW"),
						Ethertype:      types.StringValue("IPv4"),
						Name:           types.StringValue("existing-acl"),
						Protocol:       types.StringValue("TCP"),
						RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
					},
					{
						Action:         types.StringValue("ALLOW"),
						Ethertype:      types.StringValue("IPv4"),
						Name:           types.StringValue("new-acl"),
						Protocol:       types.StringValue("UDP"),
						RemoteIPPrefix: types.StringValue("10.0.0.0/8"),
					},
				},
			},
			state: LoadBalancerModel{
				ACLs: &[]ACLModel{
					{
						Action:         types.StringValue("ALLOW"),
						Ethertype:      types.StringValue("IPv4"),
						Name:           types.StringValue("existing-acl"),
						Protocol:       types.StringValue("TCP"),
						RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
					},
				},
			},
			expected: true,
		},
		{
			name: "ACL removed",
			plan: LoadBalancerModel{
				ACLs: &[]ACLModel{
					{
						Action:         types.StringValue("ALLOW"),
						Ethertype:      types.StringValue("IPv4"),
						Name:           types.StringValue("remaining-acl"),
						Protocol:       types.StringValue("TCP"),
						RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
					},
				},
			},
			state: LoadBalancerModel{
				ACLs: &[]ACLModel{
					{
						Action:         types.StringValue("ALLOW"),
						Ethertype:      types.StringValue("IPv4"),
						Name:           types.StringValue("remaining-acl"),
						Protocol:       types.StringValue("TCP"),
						RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
					},
					{
						Action:         types.StringValue("ALLOW"),
						Ethertype:      types.StringValue("IPv4"),
						Name:           types.StringValue("removed-acl"),
						Protocol:       types.StringValue("UDP"),
						RemoteIPPrefix: types.StringValue("10.0.0.0/8"),
					},
				},
			},
			expected: true,
		},
		{
			name: "multiple ACLs with some changes",
			plan: LoadBalancerModel{
				ACLs: &[]ACLModel{
					{
						Action:         types.StringValue("ALLOW"),
						Ethertype:      types.StringValue("IPv4"),
						Name:           types.StringValue("unchanged-acl"),
						Protocol:       types.StringValue("TCP"),
						RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
					},
					{
						Action:         types.StringValue("DENY"),
						Ethertype:      types.StringValue("IPv4"),
						Name:           types.StringValue("changed-acl"),
						Protocol:       types.StringValue("UDP"),
						RemoteIPPrefix: types.StringValue("10.0.0.0/8"),
					},
				},
			},
			state: LoadBalancerModel{
				ACLs: &[]ACLModel{
					{
						Action:         types.StringValue("ALLOW"),
						Ethertype:      types.StringValue("IPv4"),
						Name:           types.StringValue("unchanged-acl"),
						Protocol:       types.StringValue("TCP"),
						RemoteIPPrefix: types.StringValue("192.168.1.0/24"),
					},
					{
						Action:         types.StringValue("ALLOW"),
						Ethertype:      types.StringValue("IPv4"),
						Name:           types.StringValue("changed-acl"),
						Protocol:       types.StringValue("TCP"),
						RemoteIPPrefix: types.StringValue("192.168.2.0/24"),
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.plan.hasACLChanges(tt.state)
			assert.Equal(t, tt.expected, result, "hasACLChanges returned unexpected result")
		})
	}
}

func TestHealthCheckModel_hasHealthCheckChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		plan     HealthCheckModel
		state    HealthCheckModel
		expected bool
	}{
		{
			name: "no changes - identical health checks",
			plan: HealthCheckModel{
				Protocol:                types.StringValue("http"),
				Port:                    types.Int64Value(80),
				Path:                    types.StringValue("/health"),
				HealthyStatusCode:       types.Int64Value(200),
				IntervalSeconds:         types.Int64Value(30),
				TimeoutSeconds:          types.Int64Value(5),
				InitialDelaySeconds:     types.Int64Value(10),
				HealthyThresholdCount:   types.Int64Value(3),
				UnhealthyThresholdCount: types.Int64Value(2),
			},
			state: HealthCheckModel{
				Protocol:                types.StringValue("http"),
				Port:                    types.Int64Value(80),
				Path:                    types.StringValue("/health"),
				HealthyStatusCode:       types.Int64Value(200),
				IntervalSeconds:         types.Int64Value(30),
				TimeoutSeconds:          types.Int64Value(5),
				InitialDelaySeconds:     types.Int64Value(10),
				HealthyThresholdCount:   types.Int64Value(3),
				UnhealthyThresholdCount: types.Int64Value(2),
			},
			expected: false,
		},
		{
			name: "protocol changed",
			plan: HealthCheckModel{
				Protocol: types.StringValue("tcp"),
				Port:     types.Int64Value(80),
			},
			state: HealthCheckModel{
				Protocol: types.StringValue("http"),
				Port:     types.Int64Value(80),
			},
			expected: true,
		},
		{
			name: "path null vs empty - no change",
			plan: HealthCheckModel{
				Protocol: types.StringValue("http"),
				Port:     types.Int64Value(80),
				Path:     types.StringNull(),
			},
			state: HealthCheckModel{
				Protocol: types.StringValue("http"),
				Port:     types.Int64Value(80),
				Path:     types.StringValue(""),
			},
			expected: false,
		},
		{
			name: "unknown field in plan - returns false",
			plan: HealthCheckModel{
				Protocol:        types.StringValue("http"),
				Port:            types.Int64Value(80),
				IntervalSeconds: types.Int64Unknown(),
			},
			state: HealthCheckModel{
				Protocol:        types.StringValue("http"),
				Port:            types.Int64Value(80),
				IntervalSeconds: types.Int64Value(15),
			},
			expected: false,
		},
		{
			name: "healthy_threshold_count changed",
			plan: HealthCheckModel{
				Protocol:              types.StringValue("http"),
				Port:                  types.Int64Value(80),
				HealthyThresholdCount: types.Int64Value(5),
			},
			state: HealthCheckModel{
				Protocol:              types.StringValue("http"),
				Port:                  types.Int64Value(80),
				HealthyThresholdCount: types.Int64Value(3),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.plan.hasHealthCheckChanges(tt.state)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestLoadBalancerModel_healthChecksToUpdate(t *testing.T) {
	t.Parallel()

	// Nil cases
	{
		plan := LoadBalancerModel{HealthChecks: nil}
		state := LoadBalancerModel{HealthChecks: nil}
		has, updates := plan.healthChecksToUpdate(state)
		assert.False(t, has)
		assert.Nil(t, updates)
	}

	// Setup state
	stateHCs := []HealthCheckModel{
		{
			ID:       types.StringValue("hc-1"),
			Name:     types.StringValue("hc-1"),
			Protocol: types.StringValue("http"),
			Port:     types.Int64Value(80),
			Path:     types.StringValue("/health"),
		},
		{
			ID:       types.StringValue("hc-2"),
			Name:     types.StringValue("hc-2"),
			Protocol: types.StringValue("tcp"),
			Port:     types.Int64Value(443),
			Path:     types.StringNull(),
		},
	}
	state := LoadBalancerModel{HealthChecks: &stateHCs}

	// Plan changes: hc-1 port change; hc-2 unchanged; hc-3 is new (ignored by update)
	planHCs := []HealthCheckModel{
		{
			ID:       types.StringValue("hc-1"),
			Name:     types.StringValue("hc-1"),
			Protocol: types.StringValue("http"),
			Port:     types.Int64Value(81), // changed
			Path:     types.StringValue("/health"),
		},
		{
			ID:       types.StringValue("hc-2"),
			Name:     types.StringValue("hc-2"),
			Protocol: types.StringValue("tcp"),
			Port:     types.Int64Value(443),
			Path:     types.StringNull(),
		},
		{
			Name:     types.StringValue("hc-3"),
			Protocol: types.StringValue("http"),
			Port:     types.Int64Value(8080),
			Path:     types.StringValue("/live"),
		},
	}
	plan := LoadBalancerModel{HealthChecks: &planHCs}

	has, updates := plan.healthChecksToUpdate(state)
	assert.True(t, has)
	require.Len(t, updates, 1)
	// Ensure the returned pointer is to the plan item for hc-1
	require.Equal(t, "hc-1", updates[0].Name.ValueString())
	require.Equal(t, int64(81), updates[0].Port.ValueInt64())
}

func TestBackendModel_hasBackendFieldChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		plan     BackendModel
		state    BackendModel
		expected bool
	}{
		{
			name: "no change",
			plan: BackendModel{
				PanicThreshold:                      types.Float64Value(0.5),
				CloseConnectionsOnHostHealthFailure: types.BoolValue(false),
			},
			state: BackendModel{
				PanicThreshold:                      types.Float64Value(0.5),
				CloseConnectionsOnHostHealthFailure: types.BoolValue(false),
			},
			expected: false,
		},
		{
			name: "panic_threshold changed",
			plan: BackendModel{
				PanicThreshold:                      types.Float64Value(0.7),
				CloseConnectionsOnHostHealthFailure: types.BoolValue(false),
			},
			state: BackendModel{
				PanicThreshold:                      types.Float64Value(0.5),
				CloseConnectionsOnHostHealthFailure: types.BoolValue(false),
			},
			expected: true,
		},
		{
			name: "close_connections_on_host_health_failure changed",
			plan: BackendModel{
				PanicThreshold:                      types.Float64Value(0.5),
				CloseConnectionsOnHostHealthFailure: types.BoolValue(true),
			},
			state: BackendModel{
				PanicThreshold:                      types.Float64Value(0.5),
				CloseConnectionsOnHostHealthFailure: types.BoolValue(false),
			},
			expected: true,
		},
		{
			name: "unknown in plan returns false",
			plan: BackendModel{
				PanicThreshold:                      types.Float64Unknown(),
				CloseConnectionsOnHostHealthFailure: types.BoolValue(false),
			},
			state: BackendModel{
				PanicThreshold:                      types.Float64Value(0.5),
				CloseConnectionsOnHostHealthFailure: types.BoolValue(false),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.plan.hasBackendFieldChanges(tt.state)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestBackendModel_hasTargetChanges(t *testing.T) {
	t.Parallel()

	// Base state with two targets
	state := BackendModel{
		Targets: []TargetModel{
			{
				Port:      types.Int64Value(80),
				NICID:     types.StringNull(),
				IPAddress: types.StringValue("10.0.0.1"),
			},
			{
				Port:      types.Int64Value(81),
				NICID:     types.StringNull(),
				IPAddress: types.StringValue("10.0.0.2"),
			},
		},
	}

	tests := []struct {
		name     string
		plan     BackendModel
		expected bool
	}{
		{
			name: "no change - identical",
			plan: BackendModel{
				Targets: []TargetModel{
					{
						Port:      types.Int64Value(80),
						NICID:     types.StringNull(),
						IPAddress: types.StringValue("10.0.0.1"),
					},
					{
						Port:      types.Int64Value(81),
						NICID:     types.StringNull(),
						IPAddress: types.StringValue("10.0.0.2"),
					},
				},
			},
			expected: false,
		},
		{
			name: "length mismatch - removed target",
			plan: BackendModel{
				Targets: []TargetModel{
					{
						Port:      types.Int64Value(80),
						NICID:     types.StringNull(),
						IPAddress: types.StringValue("10.0.0.1"),
					},
				},
			},
			expected: true,
		},
		{
			name: "port changed on existing target",
			plan: BackendModel{
				Targets: []TargetModel{
					{
						Port:      types.Int64Value(8080), // changed
						NICID:     types.StringNull(),
						IPAddress: types.StringValue("10.0.0.1"),
					},
					{
						Port:      types.Int64Value(81),
						NICID:     types.StringNull(),
						IPAddress: types.StringValue("10.0.0.2"),
					},
				},
			},
			expected: true,
		},
		{
			name: "null vs empty string equivalence (nic_id/ip_address) - no change",
			plan: BackendModel{
				Targets: []TargetModel{
					{
						Port:      types.Int64Value(80),
						NICID:     types.StringValue(""),
						IPAddress: types.StringValue("10.0.0.1"),
					},
					{
						Port:      types.Int64Value(81),
						NICID:     types.StringNull(),
						IPAddress: types.StringValue("10.0.0.2"),
					},
				},
			},
			expected: false,
		},
		{
			name: "unknown in plan target - no change signaled",
			plan: BackendModel{
				Targets: []TargetModel{
					{
						Port:      types.Int64Unknown(), // unknown
						NICID:     types.StringNull(),
						IPAddress: types.StringValue("10.0.0.1"),
					},
					{
						Port:      types.Int64Value(81),
						NICID:     types.StringNull(),
						IPAddress: types.StringValue("10.0.0.2"),
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.plan.hasTargetChanges(state)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestLoadBalancerModel_backendsToUpdate(t *testing.T) {
	t.Parallel()

	// State with two backends
	state := LoadBalancerModel{
		Backends: []BackendModel{
			{
				ID:                                  types.StringValue("b1"),
				Name:                                types.StringValue("backend-a"),
				PanicThreshold:                      types.Float64Value(0.5),
				CloseConnectionsOnHostHealthFailure: types.BoolValue(false),
				Targets: []TargetModel{
					{Port: types.Int64Value(80), IPAddress: types.StringValue("10.0.0.1")},
				},
			},
			{
				ID:                                  types.StringValue("b2"),
				Name:                                types.StringValue("backend-b"),
				PanicThreshold:                      types.Float64Value(0.3),
				CloseConnectionsOnHostHealthFailure: types.BoolValue(false),
				Targets: []TargetModel{
					{Port: types.Int64Value(81), IPAddress: types.StringValue("10.0.0.2")},
				},
			},
		},
	}

	// Plan: b1 has field change; b2 has target change; b3 is new (ignored)
	plan := LoadBalancerModel{
		Backends: []BackendModel{
			{
				ID:                                  types.StringValue("b1"),
				Name:                                types.StringValue("backend-a"),
				PanicThreshold:                      types.Float64Value(0.7), // field change
				CloseConnectionsOnHostHealthFailure: types.BoolValue(false),
				Targets: []TargetModel{
					{Port: types.Int64Value(80), IPAddress: types.StringValue("10.0.0.1")},
				},
			},
			{
				ID:                                  types.StringValue("b2"),
				Name:                                types.StringValue("backend-b"),
				PanicThreshold:                      types.Float64Value(0.3),
				CloseConnectionsOnHostHealthFailure: types.BoolValue(false),
				Targets: []TargetModel{
					{Port: types.Int64Value(8081), IPAddress: types.StringValue("10.0.0.2")}, // target change
				},
			},
			{
				Name:                                types.StringValue("backend-new"),
				PanicThreshold:                      types.Float64Value(0.2),
				CloseConnectionsOnHostHealthFailure: types.BoolValue(true),
				Targets: []TargetModel{
					{Port: types.Int64Value(90), IPAddress: types.StringValue("10.0.0.9")},
				},
			},
		},
	}

	fieldUpdates, targetUpdates := plan.backendsToUpdate(state)

	require.Len(t, fieldUpdates, 1)
	require.Len(t, targetUpdates, 1)

	assert.Equal(t, "backend-a", fieldUpdates[0].Name.ValueString())
	assert.Equal(t, "backend-b", targetUpdates[0].Name.ValueString())
}

func TestTargetModel_hasUnknowns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		target   TargetModel
		expected bool
	}{
		{
			name: "no unknowns",
			target: TargetModel{
				NICID:     types.StringNull(),
				IPAddress: types.StringValue("10.0.0.1"),
				Port:      types.Int64Value(80),
			},
			expected: false,
		},
		{
			name: "unknown port",
			target: TargetModel{
				NICID:     types.StringNull(),
				IPAddress: types.StringValue("10.0.0.1"),
				Port:      types.Int64Unknown(),
			},
			expected: true,
		},
		{
			name: "unknown ip",
			target: TargetModel{
				NICID:     types.StringNull(),
				IPAddress: types.StringUnknown(),
				Port:      types.Int64Value(80),
			},
			expected: true,
		},
		{
			name: "unknown nic",
			target: TargetModel{
				NICID:     types.StringUnknown(),
				IPAddress: types.StringValue("10.0.0.1"),
				Port:      types.Int64Value(80),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.target.hasUnknowns()
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestTargetModel_equalsNormalized(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        TargetModel
		b        TargetModel
		expected bool
	}{
		{
			name: "equal with null vs empty nic",
			a: TargetModel{
				NICID:     types.StringNull(),
				IPAddress: types.StringValue("10.0.0.1"),
				Port:      types.Int64Value(80),
			},
			b: TargetModel{
				NICID:     types.StringValue(""),
				IPAddress: types.StringValue("10.0.0.1"),
				Port:      types.Int64Value(80),
			},
			expected: true,
		},
		{
			name: "different IP",
			a: TargetModel{
				NICID:     types.StringNull(),
				IPAddress: types.StringValue("10.0.0.1"),
				Port:      types.Int64Value(80),
			},
			b: TargetModel{
				NICID:     types.StringNull(),
				IPAddress: types.StringValue("10.0.0.2"),
				Port:      types.Int64Value(80),
			},
			expected: false,
		},
		{
			name: "unknown on either side yields true (tolerant)",
			a: TargetModel{
				NICID:     types.StringNull(),
				IPAddress: types.StringUnknown(),
				Port:      types.Int64Value(80),
			},
			b: TargetModel{
				NICID:     types.StringNull(),
				IPAddress: types.StringValue("10.0.0.1"),
				Port:      types.Int64Value(80),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.equalsNormalized(tt.b)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestLoadBalancerModel_backendsToUpdate_DualChangeSameBackend(t *testing.T) {
	t.Parallel()

	// State with one backend and one target
	state := LoadBalancerModel{
		Backends: []BackendModel{
			{
				ID:                                  types.StringValue("b1"),
				Name:                                types.StringValue("backend-a"),
				PanicThreshold:                      types.Float64Value(0.5),
				CloseConnectionsOnHostHealthFailure: types.BoolValue(false),
				Targets: []TargetModel{
					{Port: types.Int64Value(80), IPAddress: types.StringValue("10.0.0.1")},
				},
			},
		},
	}

	// Plan changes: same backend requires both a field change and a target change
	plan := LoadBalancerModel{
		Backends: []BackendModel{
			{
				ID:                                  types.StringValue("b1"),
				Name:                                types.StringValue("backend-a"),
				PanicThreshold:                      types.Float64Value(0.7), // field change
				CloseConnectionsOnHostHealthFailure: types.BoolValue(true),   // field change
				Targets: []TargetModel{
					{Port: types.Int64Value(8080), IPAddress: types.StringValue("10.0.0.1")}, // target change
				},
			},
		},
	}

	fieldUpdates, targetUpdates := plan.backendsToUpdate(state)

	require.Len(t, fieldUpdates, 1, "should detect field updates")
	require.Len(t, targetUpdates, 1, "should detect target updates")
	assert.Equal(t, "backend-a", fieldUpdates[0].Name.ValueString())
	assert.Equal(t, "backend-a", targetUpdates[0].Name.ValueString())
}

func TestLoadBalancerModel_healthChecksToUpdate_MatchByName(t *testing.T) {
	t.Parallel()

	// State has a HC with name but missing ID
	stateHCs := []HealthCheckModel{
		{
			ID:       types.StringNull(),
			Name:     types.StringValue("hc-by-name"),
			Protocol: types.StringValue("http"),
			Port:     types.Int64Value(80),
			Path:     types.StringValue("/health"),
		},
	}
	state := LoadBalancerModel{HealthChecks: &stateHCs}

	// Plan changes same HC by name (no ID), port changed
	planHCs := []HealthCheckModel{
		{
			Name:     types.StringValue("hc-by-name"),
			Protocol: types.StringValue("http"),
			Port:     types.Int64Value(81), // changed
			Path:     types.StringValue("/health"),
		},
	}
	plan := LoadBalancerModel{HealthChecks: &planHCs}

	has, updates := plan.healthChecksToUpdate(state)
	assert.True(t, has)
	require.Len(t, updates, 1)
	assert.Equal(t, "hc-by-name", updates[0].Name.ValueString())
	assert.Equal(t, int64(81), updates[0].Port.ValueInt64())
}

func Test_backendItemModel_fromSDKBackend(t *testing.T) {
	t.Parallel()

	in := &lbSDK.NetworkBackendResponse{
		ID:                                  "backend-123",
		Name:                                "backend-name",
		Description:                         stringPtr("desc"),
		BalanceAlgorithm:                    lbSDK.BackendBalanceAlgorithm("ROUND_ROBIN"),
		HealthCheckID:                       stringPtr("hc-1"),
		PanicThreshold:                      float64Ptr(0.9),
		CloseConnectionsOnHostHealthFailure: boolPtr(true),
		TargetsType:                         lbSDK.BackendType("INSTANCE"),
		Targets: []lbSDK.NetworkBackedTarget{
			{
				ID:        "t1",
				Port:      int64Ptr(80),
				NicID:     stringPtr("nic-1"),
				IPAddress: stringPtr("10.0.0.1"),
			},
			{
				ID:        "t2",
				Port:      int64Ptr(81),
				NicID:     nil,
				IPAddress: stringPtr("10.0.0.2"),
			},
		},
	}

	var item backendItemModel
	out := item.fromSDKBackend(in)

	assert.Equal(t, types.StringValue("backend-123"), out.ID)
	assert.Equal(t, types.StringValue("backend-name"), out.Name)
	assert.Equal(t, types.StringValue("desc"), out.Description)
	assert.Equal(t, types.StringValue("ROUND_ROBIN"), out.BalanceAlgorithm)
	assert.Equal(t, types.StringValue("hc-1"), out.HealthCheckID)
	assert.Equal(t, types.Float64Value(0.9), out.PanicThreshold)
	assert.Equal(t, types.BoolValue(true), out.CloseConnectionsOnHostHealthFailure)
	assert.Equal(t, types.StringValue("INSTANCE"), out.TargetsType)

	require.Len(t, out.Targets, 2)
	assert.Equal(t, types.Int64Value(80), out.Targets[0].Port)
	assert.Equal(t, types.StringValue("nic-1"), out.Targets[0].NICID)
	assert.Equal(t, types.StringValue("10.0.0.1"), out.Targets[0].IPAddress)

	assert.Equal(t, types.Int64Value(81), out.Targets[1].Port)
	assert.Equal(t, types.StringNull(), out.Targets[1].NICID)
	assert.Equal(t, types.StringValue("10.0.0.2"), out.Targets[1].IPAddress)
}
