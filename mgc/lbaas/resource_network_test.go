package lbaas

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	lbSDK "github.com/MagaluCloud/mgc-sdk-go/lbaas"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type mockLBService struct {
	lbSDK.NetworkLoadBalancerService
	UpdateFunc func(ctx context.Context, id string, req lbSDK.UpdateNetworkLoadBalancerRequest) (string, error)
	GetFunc    func(ctx context.Context, id string) (lbSDK.NetworkLoadBalancerResponse, error)
}

func (m *mockLBService) Update(ctx context.Context, id string, req lbSDK.UpdateNetworkLoadBalancerRequest) (string, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, req)
	}
	return "", errors.New("UpdateFunc not implemented")
}

func (m *mockLBService) Get(ctx context.Context, id string) (lbSDK.NetworkLoadBalancerResponse, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, id)
	}
	return lbSDK.NetworkLoadBalancerResponse{}, errors.New("GetFunc not implemented")
}

type mockACLService struct {
	lbSDK.NetworkACLService
	ReplaceFunc func(ctx context.Context, id string, req lbSDK.UpdateNetworkACLRequest) error
}

func (m *mockACLService) Replace(ctx context.Context, id string, req lbSDK.UpdateNetworkACLRequest) error {
	if m.ReplaceFunc != nil {
		return m.ReplaceFunc(ctx, id, req)
	}
	return errors.New("ReplaceFunc not implemented")
}

type mockHealthCheckService struct {
	lbSDK.NetworkHealthCheckService
	UpdateFunc func(ctx context.Context, lbID string, hcID string, req lbSDK.UpdateNetworkHealthCheckRequest) error
}

func (m *mockHealthCheckService) Update(ctx context.Context, lbID string, hcID string, req lbSDK.UpdateNetworkHealthCheckRequest) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, lbID, hcID, req)
	}
	return errors.New("UpdateFunc not implemented")
}

type mockBackendService struct {
	lbSDK.NetworkBackendService
	UpdateFunc func(ctx context.Context, lbID string, backendID string, req lbSDK.UpdateNetworkBackendRequest) (string, error)
}

func (m *mockBackendService) Update(ctx context.Context, lbID string, backendID string, req lbSDK.UpdateNetworkBackendRequest) (string, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, lbID, backendID, req)
	}
	return "", errors.New("UpdateFunc not implemented")
}

type mockTargetService struct {
	lbSDK.NetworkBackendTargetService
	ReplaceFunc func(ctx context.Context, lbID string, backendID string, req lbSDK.CreateNetworkBackendTargetRequest) (string, error)
}

func (m *mockTargetService) Replace(ctx context.Context, lbID string, backendID string, req lbSDK.CreateNetworkBackendTargetRequest) (string, error) {
	if m.ReplaceFunc != nil {
		return m.ReplaceFunc(ctx, lbID, backendID, req)
	}
	return "", errors.New("ReplaceFunc not implemented")
}

func canceledCtx() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func Test_updateLBNameDescription_noChange(t *testing.T) {
	r := &LoadBalancerResource{
		lbNetworkLB: &mockLBService{
			UpdateFunc: func(ctx context.Context, id string, req lbSDK.UpdateNetworkLoadBalancerRequest) (string, error) {
				t.Fatalf("unexpected call to Update")
				return "", nil
			},
			GetFunc: func(ctx context.Context, id string) (lbSDK.NetworkLoadBalancerResponse, error) {
				t.Fatalf("unexpected call to Get")
				return lbSDK.NetworkLoadBalancerResponse{}, nil
			},
		},
	}

	state := LoadBalancerModel{
		ID:          types.StringValue("lb-1"),
		Name:        types.StringValue("name"),
		Description: types.StringValue("desc"),
	}
	plan := state

	err := r.updateLBNameDescription(context.Background(), &plan, &state)
	assert.NoError(t, err)
	assert.Equal(t, "name", state.Name.ValueString())
	assert.Equal(t, "desc", state.Description.ValueString())
}

func Test_updateLBNameDescription_callsUpdate_and_waitError(t *testing.T) {
	calls := 0
	r := &LoadBalancerResource{
		lbNetworkLB: &mockLBService{
			UpdateFunc: func(ctx context.Context, id string, req lbSDK.UpdateNetworkLoadBalancerRequest) (string, error) {
				calls++
				assert.Equal(t, "lb-1", id)
				assert.NotNil(t, req.Name)
				assert.NotNil(t, req.Description)
				assert.Equal(t, "new", *req.Name)
				assert.Equal(t, "new-desc", *req.Description)
				return "ok", nil
			},
		},
	}

	state := LoadBalancerModel{
		ID:          types.StringValue("lb-1"),
		Name:        types.StringValue("old"),
		Description: types.StringValue("old-desc"),
	}
	plan := LoadBalancerModel{
		ID:          types.StringValue("lb-1"),
		Name:        types.StringValue("new"),
		Description: types.StringValue("new-desc"),
	}

	err := r.updateLBNameDescription(canceledCtx(), &plan, &state)
	assert.Error(t, err)
	assert.Equal(t, 1, calls)
	assert.Equal(t, "new", state.Name.ValueString())
	assert.Equal(t, "new-desc", state.Description.ValueString())
}

func Test_replaceACLsIfChanged_noChange(t *testing.T) {
	acls := []ACLModel{
		{
			Action:         types.StringValue("ALLOW"),
			Ethertype:      types.StringValue("IPv4"),
			Name:           types.StringValue("n"),
			Protocol:       types.StringValue("tcp"),
			RemoteIPPrefix: types.StringValue("0.0.0.0/0"),
		},
	}
	r := &LoadBalancerResource{
		lbNetworkACL: &mockACLService{
			ReplaceFunc: func(ctx context.Context, id string, req lbSDK.UpdateNetworkACLRequest) error {
				t.Fatalf("unexpected call to Replace")
				return nil
			},
		},
	}
	state := LoadBalancerModel{ID: types.StringValue("lb-1"), ACLs: &acls}
	plan := LoadBalancerModel{ID: types.StringValue("lb-1"), ACLs: &acls}

	err := r.replaceACLsIfChanged(context.Background(), &plan, &state)
	assert.NoError(t, err)
	assert.Equal(t, &acls, state.ACLs)
}

func Test_replaceACLsIfChanged_callsReplace_and_waitError(t *testing.T) {
	replaceCalls := 0
	r := &LoadBalancerResource{
		lbNetworkACL: &mockACLService{
			ReplaceFunc: func(ctx context.Context, id string, req lbSDK.UpdateNetworkACLRequest) error {
				replaceCalls++
				assert.Equal(t, "lb-1", id)
				assert.Len(t, req.Acls, 1)
				assert.Equal(t, "ALLOW", string(req.Acls[0].Action))
				return nil
			},
		},
	}

	state := LoadBalancerModel{ID: types.StringValue("lb-1")}
	planACLs := []ACLModel{
		{
			Action:         types.StringValue("ALLOW"),
			Ethertype:      types.StringValue("IPv4"),
			Name:           types.StringNull(),
			Protocol:       types.StringValue("tcp"),
			RemoteIPPrefix: types.StringValue("0.0.0.0/0"),
		},
	}
	plan := LoadBalancerModel{ID: types.StringValue("lb-1"), ACLs: &planACLs}

	err := r.replaceACLsIfChanged(canceledCtx(), &plan, &state)
	assert.Error(t, err)
	assert.Equal(t, 1, replaceCalls)
	assert.Equal(t, &planACLs, state.ACLs)
}

func Test_updateHealthChecks_nil_noop(t *testing.T) {
	updateCalls := 0
	r := &LoadBalancerResource{
		lbNetworkHealthCheck: &mockHealthCheckService{
			UpdateFunc: func(ctx context.Context, lbID string, hcID string, req lbSDK.UpdateNetworkHealthCheckRequest) error {
				updateCalls++
				return nil
			},
		},
	}

	state := LoadBalancerModel{ID: types.StringValue("lb-1"), HealthChecks: nil}
	plan := LoadBalancerModel{ID: types.StringValue("lb-1"), HealthChecks: nil}

	err := r.updateHealthChecks(context.Background(), &plan, &state)
	assert.NoError(t, err)
	assert.Equal(t, 0, updateCalls)
}

func Test_updateHealthChecks_equal_stillWaits_and_errors(t *testing.T) {
	updateCalls := 0
	r := &LoadBalancerResource{
		lbNetworkHealthCheck: &mockHealthCheckService{
			UpdateFunc: func(ctx context.Context, lbID string, hcID string, req lbSDK.UpdateNetworkHealthCheckRequest) error {
				updateCalls++
				return nil
			},
		},
	}

	hc := HealthCheckModel{
		ID:                      types.StringValue("hc-1"),
		Name:                    types.StringValue("hc-a"),
		Protocol:                types.StringValue("http"),
		Port:                    types.Int64Value(80),
		Path:                    types.StringValue("/"),
		HealthyStatusCode:       types.Int64Value(200),
		HealthyThresholdCount:   types.Int64Value(3),
		InitialDelaySeconds:     types.Int64Value(0),
		IntervalSeconds:         types.Int64Value(10),
		TimeoutSeconds:          types.Int64Value(5),
		UnhealthyThresholdCount: types.Int64Value(3),
	}
	stateHCs := []HealthCheckModel{hc}
	planHCs := []HealthCheckModel{hc}

	state := LoadBalancerModel{ID: types.StringValue("lb-1"), HealthChecks: &stateHCs}
	plan := LoadBalancerModel{ID: types.StringValue("lb-1"), HealthChecks: &planHCs}

	err := r.updateHealthChecks(canceledCtx(), &plan, &state)
	assert.Error(t, err)
	assert.Equal(t, 0, updateCalls)
}

func Test_updateHealthChecks_updates_and_waitError(t *testing.T) {
	var gotLBID, gotHCID string
	var gotReq lbSDK.UpdateNetworkHealthCheckRequest
	updateCalls := 0

	r := &LoadBalancerResource{
		lbNetworkHealthCheck: &mockHealthCheckService{
			UpdateFunc: func(ctx context.Context, lbID string, hcID string, req lbSDK.UpdateNetworkHealthCheckRequest) error {
				updateCalls++
				gotLBID, gotHCID, gotReq = lbID, hcID, req
				return nil
			},
		},
	}

	stateHC := HealthCheckModel{
		ID:                      types.StringValue("hc-1"),
		Name:                    types.StringValue("hc-a"),
		Protocol:                types.StringValue("http"),
		Port:                    types.Int64Value(80),
		Path:                    types.StringValue("/"),
		HealthyStatusCode:       types.Int64Value(200),
		HealthyThresholdCount:   types.Int64Value(3),
		InitialDelaySeconds:     types.Int64Value(0),
		IntervalSeconds:         types.Int64Value(10),
		TimeoutSeconds:          types.Int64Value(5),
		UnhealthyThresholdCount: types.Int64Value(3),
	}
	planHC := stateHC
	planHC.Port = types.Int64Value(81)

	stateHCs := []HealthCheckModel{stateHC}
	planHCs := []HealthCheckModel{planHC}

	state := LoadBalancerModel{ID: types.StringValue("lb-1"), HealthChecks: &stateHCs}
	plan := LoadBalancerModel{ID: types.StringValue("lb-1"), HealthChecks: &planHCs}

	err := r.updateHealthChecks(canceledCtx(), &plan, &state)
	assert.Error(t, err)
	assert.Equal(t, 1, updateCalls)
	assert.Equal(t, "lb-1", gotLBID)
	assert.Equal(t, "hc-1", gotHCID)
	assert.Equal(t, 81, gotReq.Port)
	assert.NotNil(t, gotReq.HealthyStatusCode)
	assert.Equal(t, 200, *gotReq.HealthyStatusCode)
}

func Test_updateBackendsFields_noop(t *testing.T) {
	updateCalls := 0
	r := &LoadBalancerResource{
		lbNetworkBackend: &mockBackendService{
			UpdateFunc: func(ctx context.Context, lbID string, backendID string, req lbSDK.UpdateNetworkBackendRequest) (string, error) {
				updateCalls++
				return "", nil
			},
		},
	}

	state := LoadBalancerModel{
		ID: types.StringValue("lb-1"),
		Backends: []BackendModel{
			{
				ID:                                  types.StringValue("b-1"),
				Name:                                types.StringValue("backend-a"),
				PanicThreshold:                      types.Float64Value(50),
				CloseConnectionsOnHostHealthFailure: types.BoolValue(false),
			},
		},
	}
	plan := state

	err := r.updateBackendsFields(context.Background(), &plan, &state)
	assert.NoError(t, err)
	assert.Equal(t, 0, updateCalls)
}

func Test_updateBackendsFields_updates_and_waitError(t *testing.T) {
	updateCalls := 0
	var gotReq lbSDK.UpdateNetworkBackendRequest
	r := &LoadBalancerResource{
		lbNetworkBackend: &mockBackendService{
			UpdateFunc: func(ctx context.Context, lbID string, backendID string, req lbSDK.UpdateNetworkBackendRequest) (string, error) {
				updateCalls++
				assert.Equal(t, "lb-1", lbID)
				assert.Equal(t, "b-1", backendID)
				gotReq = req
				return "ok", nil
			},
		},
	}

	state := LoadBalancerModel{
		ID: types.StringValue("lb-1"),
		Backends: []BackendModel{
			{
				ID:                                  types.StringValue("b-1"),
				Name:                                types.StringValue("backend-a"),
				PanicThreshold:                      types.Float64Value(50),
				CloseConnectionsOnHostHealthFailure: types.BoolValue(false),
			},
		},
	}
	plan := LoadBalancerModel{
		ID: types.StringValue("lb-1"),
		Backends: []BackendModel{
			{
				ID:                                  types.StringValue("b-1"),
				Name:                                types.StringValue("backend-a"),
				PanicThreshold:                      types.Float64Value(60),
				CloseConnectionsOnHostHealthFailure: types.BoolValue(true),
			},
		},
	}

	err := r.updateBackendsFields(canceledCtx(), &plan, &state)
	assert.Error(t, err)
	assert.Equal(t, 1, updateCalls)
	assert.NotNil(t, gotReq.PanicThreshold)
	assert.NotNil(t, gotReq.CloseConnectionsOnHostHealthFailure)
	assert.Equal(t, 60.0, *gotReq.PanicThreshold)
	assert.Equal(t, true, *gotReq.CloseConnectionsOnHostHealthFailure)

	assert.Equal(t, 60.0, state.Backends[0].PanicThreshold.ValueFloat64())
	assert.Equal(t, true, state.Backends[0].CloseConnectionsOnHostHealthFailure.ValueBool())
}

func Test_updateBackendTargets_healthCheckNotFound(t *testing.T) {
	replaceCalls := 0
	r := &LoadBalancerResource{
		lbNetworkTarget: &mockTargetService{
			ReplaceFunc: func(ctx context.Context, lbID string, backendID string, req lbSDK.CreateNetworkBackendTargetRequest) (string, error) {
				replaceCalls++
				return "", nil
			},
		},
	}

	state := LoadBalancerModel{
		ID:           types.StringValue("lb-1"),
		HealthChecks: &[]HealthCheckModel{},
		Backends: []BackendModel{
			{
				ID:          types.StringValue("b-1"),
				Name:        types.StringValue("backend-a"),
				TargetsType: types.StringValue("instance"),
				Targets: []TargetModel{
					{NICID: types.StringValue("nic-1"), Port: types.Int64Value(8080)},
				},
			},
		},
	}
	plan := LoadBalancerModel{
		ID: types.StringValue("lb-1"),
		Backends: []BackendModel{
			{
				ID:              types.StringValue("b-1"),
				Name:            types.StringValue("backend-a"),
				TargetsType:     types.StringValue("instance"),
				HealthCheckName: types.StringValue("missing"),
				Targets: []TargetModel{
					{NICID: types.StringValue("nic-1"), Port: types.Int64Value(9090)},
				},
			},
		},
	}

	err := r.replaceBackendTargets(context.Background(), &plan, &state)
	var nfErr healthCheckNotFoundError
	assert.Error(t, err)
	assert.True(t, errors.As(err, &nfErr))
	assert.Equal(t, 0, replaceCalls)
}

func Test_updateBackendTargets_replace_and_waitError(t *testing.T) {
	replaceCalls := 0
	var gotReq lbSDK.CreateNetworkBackendTargetRequest

	r := &LoadBalancerResource{
		lbNetworkTarget: &mockTargetService{
			ReplaceFunc: func(ctx context.Context, lbID string, backendID string, req lbSDK.CreateNetworkBackendTargetRequest) (string, error) {
				replaceCalls++
				assert.Equal(t, "lb-1", lbID)
				assert.Equal(t, "b-1", backendID)
				gotReq = req
				return "ok", nil
			},
		},
	}

	stateHCs := []HealthCheckModel{
		{ID: types.StringValue("hc-id"), Name: types.StringValue("hc-a")},
	}
	state := LoadBalancerModel{
		ID:           types.StringValue("lb-1"),
		HealthChecks: &stateHCs,
		Backends: []BackendModel{
			{
				ID:          types.StringValue("b-1"),
				Name:        types.StringValue("backend-a"),
				TargetsType: types.StringValue("instance"),
				Targets: []TargetModel{
					{NICID: types.StringValue("nic-1"), Port: types.Int64Value(8080)},
				},
			},
		},
	}

	plan := LoadBalancerModel{
		ID: types.StringValue("lb-1"),
		Backends: []BackendModel{
			{
				ID:              types.StringValue("b-1"),
				Name:            types.StringValue("backend-a"),
				TargetsType:     types.StringValue("instance"),
				HealthCheckName: types.StringValue("hc-a"),
				Targets: []TargetModel{
					{NICID: types.StringValue("nic-2"), Port: types.Int64Value(9090)},
				},
			},
		},
	}

	err := r.replaceBackendTargets(canceledCtx(), &plan, &state)
	assert.Error(t, err)
	assert.Equal(t, 1, replaceCalls)
	if assert.NotNil(t, gotReq.HealthCheckID) {
		assert.Equal(t, "hc-id", *gotReq.HealthCheckID)
	}
	assert.Equal(t, lbSDK.BackendType("instance"), gotReq.TargetsType)
	if assert.Len(t, gotReq.Targets, 1) {
		assert.Equal(t, "nic-2", *gotReq.Targets[0].NicID)
		assert.Equal(t, int64(9090), gotReq.Targets[0].Port)
	}
	assert.Equal(t, 1, len(state.Backends[0].Targets))
	assert.Equal(t, int64(9090), state.Backends[0].Targets[0].Port.ValueInt64())
	assert.Equal(t, "nic-2", state.Backends[0].Targets[0].NICID.ValueString())
}
