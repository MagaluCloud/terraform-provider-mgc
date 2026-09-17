package lbaas

import (
	"context"
	"testing"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	lbSDK "github.com/MagaluCloud/mgc-sdk-go/lbaas"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/project"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/testutils"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

const lbTestProjectID = "550e8400-e29b-41d4-a716-446655440000"

type recordingLBService struct {
	lbSDK.NetworkLoadBalancerService
	gotCallOpts clientSDK.CallOptions
}

func (m *recordingLBService) Get(_ context.Context, callOpts clientSDK.CallOptions, id string) (lbSDK.NetworkLoadBalancerResponse, error) {
	m.gotCallOpts = callOpts
	return lbSDK.NetworkLoadBalancerResponse{ID: id, Name: "lb"}, nil
}

func TestLoadBalancerModelConversionKeepsProject(t *testing.T) {
	current := LoadBalancerModel{ProjectID: types.StringValue(lbTestProjectID)}

	converted := current.ToTerraformNetworkResource(context.Background(), lbSDK.NetworkLoadBalancerResponse{ID: "lb-1"})

	assert.Equal(t, types.StringValue(lbTestProjectID), converted.ProjectID)
}

func TestLoadBalancerResourceDeclaresProject(t *testing.T) {
	ctx := context.Background()
	s := testutils.GetResourceTestSchema(t, &LoadBalancerResource{})

	state := tfsdk.State{Schema: s.Schema, Raw: testutils.ObjectWith(ctx, s.Schema.Type(), map[string]tftypes.Value{
		"project_id": tftypes.NewValue(tftypes.String, lbTestProjectID),
	})}
	assert.False(t, state.Get(ctx, &LoadBalancerModel{}).HasError(), "model must match the schema")
}

func TestLoadBalancerReadKeepsProject(t *testing.T) {
	ctx := context.Background()

	for _, projectID := range []string{lbTestProjectID, ""} {
		expected := types.StringNull()
		if projectID != "" {
			expected = types.StringValue(projectID)
		}

		t.Run(projectID, func(t *testing.T) {
			mock := &recordingLBService{}
			r := &LoadBalancerResource{lbNetworkLB: mock}
			s := testutils.GetResourceTestSchema(t, r)
			current := tfsdk.State{Schema: s.Schema, Raw: testutils.ObjectWith(ctx, s.Schema.Type(), map[string]tftypes.Value{
				"id":         tftypes.NewValue(tftypes.String, "lb-1"),
				"project_id": testutils.OptionalString(projectID),
			})}

			resp := &resource.ReadResponse{State: current}
			r.Read(ctx, resource.ReadRequest{State: current}, resp)
			assert.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

			assert.Equal(t, clientSDK.CallOptions{ProjectID: projectID}, mock.gotCallOpts)
			var got types.String
			assert.False(t, resp.State.GetAttribute(ctx, path.Root("project_id"), &got).HasError())
			assert.Equal(t, expected, got)
		})
	}
}

func TestLoadBalancerImportStateWithProject(t *testing.T) {
	ctx := context.Background()
	r := &LoadBalancerResource{}

	s := testutils.GetResourceTestSchema(t, r)
	i := testutils.GetResourceTestIdentitySchema(t, r)

	resp := &resource.ImportStateResponse{
		State:    tfsdk.State{Schema: s.Schema, Raw: tftypes.NewValue(s.Schema.Type().TerraformType(ctx), nil)},
		Identity: &tfsdk.ResourceIdentity{Schema: i.IdentitySchema, Raw: tftypes.NewValue(i.IdentitySchema.Type().TerraformType(ctx), nil)},
	}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "project/" + lbTestProjectID + "/lb-1"}, resp)
	assert.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var id, projectID types.String
	assert.False(t, resp.State.GetAttribute(ctx, path.Root("id"), &id).HasError())
	assert.False(t, resp.State.GetAttribute(ctx, path.Root("project_id"), &projectID).HasError())
	assert.Equal(t, types.StringValue("lb-1"), id)
	assert.Equal(t, types.StringValue(lbTestProjectID), projectID)

	var identity project.IdentityModel
	assert.False(t, resp.Identity.Get(ctx, &identity).HasError())
	assert.Equal(t, types.StringValue(lbTestProjectID), identity.ProjectID)
}
