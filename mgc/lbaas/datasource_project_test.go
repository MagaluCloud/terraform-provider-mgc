package lbaas

import (
	"context"
	"testing"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	lbSDK "github.com/MagaluCloud/mgc-sdk-go/lbaas"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/testutils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

type recordingCertificateService struct {
	lbSDK.NetworkCertificateService
	gotCallOpts clientSDK.CallOptions
}

func (m *recordingCertificateService) Get(_ context.Context, callOpts clientSDK.CallOptions, _, certID string) (*lbSDK.NetworkTLSCertificateResponse, error) {
	m.gotCallOpts = callOpts
	return &lbSDK.NetworkTLSCertificateResponse{ID: certID, Name: "cert"}, nil
}

func TestLbaasDataSourcesDeclareProject(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name  string
		ds    datasource.DataSource
		model any
	}{
		{"network", &DataSourceLbaasNetwork{}, &lbNetworkState{}},
		{"networks", &DataSourceLbaasNetworks{}, &LbaasNetworksListModel{}},
		{"backend", &DataSourceLbaasNetworkBackend{}, &networkBackendItemModel{}},
		{"backends", &DataSourceLbaasNetworkBackends{}, &backendListState{}},
		{"certificate", &DataSourceLbaasNetworkCertificate{}, &certificateItemState{}},
		{"certificates", &DataSourceLbaasNetworkCertificates{}, &certificateListState{}},
		{"healthcheck", &DataSourceLbaasNetworkHealthCheck{}, &healthCheckItemModel{}},
		{"healthchecks", &DataSourceLbaasNetworkHealthChecks{}, &healthCheckListState{}},
		{"listener", &DataSourceLbaasNetworkListener{}, &networkListenerItemModel{}},
		{"listeners", &DataSourceLbaasNetworkListeners{}, &listenerListState{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := testutils.GetDataSourceTestSchema(t, tc.ds)

			config := tfsdk.Config{Schema: s.Schema, Raw: testutils.ObjectWith(ctx, s.Schema.Type(), map[string]tftypes.Value{
				"project_id": tftypes.NewValue(tftypes.String, lbTestProjectID),
			})}
			assert.False(t, config.Get(ctx, tc.model).HasError(), "model must match the schema")
		})
	}
}

func TestLbaasDataSourcesKeepProject(t *testing.T) {
	ctx := context.Background()

	for _, projectID := range []string{lbTestProjectID, ""} {
		expected := types.StringNull()
		if projectID != "" {
			expected = types.StringValue(projectID)
		}

		read := func(t *testing.T, d datasource.DataSource, values map[string]tftypes.Value) *datasource.ReadResponse {
			s := testutils.GetDataSourceTestSchema(t, d)
			values["project_id"] = testutils.OptionalString(projectID)
			raw := testutils.ObjectWith(ctx, s.Schema.Type(), values)

			resp := &datasource.ReadResponse{State: tfsdk.State{Schema: s.Schema, Raw: raw}}
			d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: s.Schema, Raw: raw}}, resp)
			assert.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
			return resp
		}
		projectOf := func(resp *datasource.ReadResponse) types.String {
			var got types.String
			assert.False(t, resp.State.GetAttribute(ctx, path.Root("project_id"), &got).HasError())
			return got
		}

		t.Run("network/"+projectID, func(t *testing.T) {
			mock := &recordingLBService{}
			resp := read(t, &DataSourceLbaasNetwork{lbNetworkLB: mock}, map[string]tftypes.Value{
				"id": tftypes.NewValue(tftypes.String, "lb-1"),
			})
			assert.Equal(t, clientSDK.CallOptions{ProjectID: projectID}, mock.gotCallOpts)
			assert.Equal(t, expected, projectOf(resp))
		})

		t.Run("certificate/"+projectID, func(t *testing.T) {
			mock := &recordingCertificateService{}
			resp := read(t, &DataSourceLbaasNetworkCertificate{lbNetworkTLS: mock}, map[string]tftypes.Value{
				"lb_id": tftypes.NewValue(tftypes.String, "lb-1"),
				"id":    tftypes.NewValue(tftypes.String, "cert-1"),
			})
			assert.Equal(t, clientSDK.CallOptions{ProjectID: projectID}, mock.gotCallOpts)
			assert.Equal(t, expected, projectOf(resp))
		})
	}
}
