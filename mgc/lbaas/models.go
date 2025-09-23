package lbaas

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"

	lbSDK "github.com/MagaluCloud/mgc-sdk-go/lbaas"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

type LoadBalancerModel struct {
	ID              types.String           `tfsdk:"id"`
	Name            types.String           `tfsdk:"name"`
	Description     types.String           `tfsdk:"description"`
	PublicIPID      types.String           `tfsdk:"public_ip_id"`
	SubnetpoolID    types.String           `tfsdk:"subnetpool_id"`
	Type            types.String           `tfsdk:"type"`
	Visibility      types.String           `tfsdk:"visibility"`
	VPCID           types.String           `tfsdk:"vpc_id"`
	ACLs            *[]ACLModel            `tfsdk:"acls"`
	Backends        []BackendModel         `tfsdk:"backends"`
	HealthChecks    *[]HealthCheckModel    `tfsdk:"health_checks"`
	Listeners       []ListenerModel        `tfsdk:"listeners"`
	TLSCertificates *[]TLSCertificateModel `tfsdk:"tls_certificates"`
}

type ACLModel struct {
	Action         types.String `tfsdk:"action"`
	Ethertype      types.String `tfsdk:"ethertype"`
	Name           types.String `tfsdk:"name"`
	Protocol       types.String `tfsdk:"protocol"`
	RemoteIPPrefix types.String `tfsdk:"remote_ip_prefix"`
}

type BackendModel struct {
	ID                                  types.String  `tfsdk:"id"`
	BalanceAlgorithm                    types.String  `tfsdk:"balance_algorithm"`
	Description                         types.String  `tfsdk:"description"`
	HealthCheckName                     types.String  `tfsdk:"health_check_name"`
	Name                                types.String  `tfsdk:"name"`
	Targets                             []TargetModel `tfsdk:"targets"`
	TargetsType                         types.String  `tfsdk:"targets_type"`
	PanicThreshold                      types.Float64 `tfsdk:"panic_threshold"`
	CloseConnectionsOnHostHealthFailure types.Bool    `tfsdk:"close_connections_on_host_health_failure"`
}

type TargetModel struct {
	NICID     types.String `tfsdk:"nic_id"`
	IPAddress types.String `tfsdk:"ip_address"`
	Port      types.Int64  `tfsdk:"port"`
}

type HealthCheckModel struct {
	ID                      types.String `tfsdk:"id"`
	Description             types.String `tfsdk:"description"`
	HealthyStatusCode       types.Int64  `tfsdk:"healthy_status_code"`
	HealthyThresholdCount   types.Int64  `tfsdk:"healthy_threshold_count"`
	InitialDelaySeconds     types.Int64  `tfsdk:"initial_delay_seconds"`
	IntervalSeconds         types.Int64  `tfsdk:"interval_seconds"`
	Name                    types.String `tfsdk:"name"`
	Path                    types.String `tfsdk:"path"`
	Port                    types.Int64  `tfsdk:"port"`
	Protocol                types.String `tfsdk:"protocol"`
	TimeoutSeconds          types.Int64  `tfsdk:"timeout_seconds"`
	UnhealthyThresholdCount types.Int64  `tfsdk:"unhealthy_threshold_count"`
}

type ListenerModel struct {
	ID                 types.String `tfsdk:"id"`
	BackendName        types.String `tfsdk:"backend_name"`
	Description        types.String `tfsdk:"description"`
	Name               types.String `tfsdk:"name"`
	Port               types.Int64  `tfsdk:"port"`
	Protocol           types.String `tfsdk:"protocol"`
	TLSCertificateName types.String `tfsdk:"tls_certificate_name"`
}

type TLSCertificateModel struct {
	Certificate    types.String `tfsdk:"certificate"`
	Description    types.String `tfsdk:"description"`
	Name           types.String `tfsdk:"name"`
	PrivateKey     types.String `tfsdk:"private_key"`
	ID             types.String `tfsdk:"id"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
}

func (lb *LoadBalancerModel) ConvertACLsToSDK() []lbSDK.CreateNetworkACLRequest {
	if lb.ACLs == nil {
		return nil
	}

	var aclRequests []lbSDK.CreateNetworkACLRequest
	for _, acl := range *lb.ACLs {
		aclRequests = append(aclRequests, lbSDK.CreateNetworkACLRequest{
			Action:         lbSDK.AclActionType(acl.Action.ValueString()),
			Name:           acl.Name.ValueStringPointer(),
			Ethertype:      lbSDK.AclEtherType(acl.Ethertype.ValueString()),
			Protocol:       lbSDK.AclProtocol(acl.Protocol.ValueString()),
			RemoteIPPrefix: acl.RemoteIPPrefix.ValueString(),
		})
	}
	return aclRequests
}

func (lb *LoadBalancerModel) ConvertBackendsToSDK() []lbSDK.CreateNetworkBackendRequest {
	var backendRequests []lbSDK.CreateNetworkBackendRequest
	for _, backend := range lb.Backends {

		var targets []lbSDK.NetworkBackendInstanceTargetRequest
		for _, target := range backend.Targets {
			targets = append(targets, lbSDK.NetworkBackendInstanceTargetRequest{
				NicID:     target.NICID.ValueStringPointer(),
				Port:      target.Port.ValueInt64(),
				IPAddress: target.IPAddress.ValueStringPointer(),
			})
		}

		backendRequests = append(backendRequests, lbSDK.CreateNetworkBackendRequest{
			HealthCheckName:                     backend.HealthCheckName.ValueStringPointer(),
			Name:                                backend.Name.ValueString(),
			Description:                         backend.Description.ValueStringPointer(),
			PanicThreshold:                      backend.PanicThreshold.ValueFloat64Pointer(),
			CloseConnectionsOnHostHealthFailure: backend.CloseConnectionsOnHostHealthFailure.ValueBoolPointer(),
			BalanceAlgorithm:                    lbSDK.BackendBalanceAlgorithm(backend.BalanceAlgorithm.ValueString()),
			TargetsType:                         lbSDK.BackendType(backend.TargetsType.ValueString()),
			Targets:                             &targets,
		})
	}
	return backendRequests
}

func (lb *LoadBalancerModel) ConvertHealthChecksToSDK() []lbSDK.CreateNetworkHealthCheckRequest {
	if lb.HealthChecks == nil {
		return nil
	}

	var healthCheckRequests []lbSDK.CreateNetworkHealthCheckRequest
	for _, healthCheck := range *lb.HealthChecks {
		healthCheckRequests = append(healthCheckRequests, lbSDK.CreateNetworkHealthCheckRequest{
			Name:                    healthCheck.Name.ValueString(),
			Description:             healthCheck.Description.ValueStringPointer(),
			Protocol:                lbSDK.HealthCheckProtocol(healthCheck.Protocol.ValueString()),
			Port:                    int(healthCheck.Port.ValueInt64()),
			Path:                    healthCheck.Path.ValueStringPointer(),
			HealthyStatusCode:       utils.ConvertInt64PointerToIntPointer(healthCheck.HealthyStatusCode.ValueInt64Pointer()),
			IntervalSeconds:         utils.ConvertInt64PointerToIntPointer(healthCheck.IntervalSeconds.ValueInt64Pointer()),
			TimeoutSeconds:          utils.ConvertInt64PointerToIntPointer(healthCheck.TimeoutSeconds.ValueInt64Pointer()),
			InitialDelaySeconds:     utils.ConvertInt64PointerToIntPointer(healthCheck.InitialDelaySeconds.ValueInt64Pointer()),
			HealthyThresholdCount:   utils.ConvertInt64PointerToIntPointer(healthCheck.HealthyThresholdCount.ValueInt64Pointer()),
			UnhealthyThresholdCount: utils.ConvertInt64PointerToIntPointer(healthCheck.UnhealthyThresholdCount.ValueInt64Pointer()),
		})
	}
	return healthCheckRequests
}

func (lb *LoadBalancerModel) ConvertListenersToSDK() []lbSDK.NetworkListenerRequest {
	var listenerRequests []lbSDK.NetworkListenerRequest
	for _, listener := range lb.Listeners {
		listenerRequests = append(listenerRequests, lbSDK.NetworkListenerRequest{
			Name:               listener.Name.ValueString(),
			Description:        listener.Description.ValueStringPointer(),
			Port:               int(listener.Port.ValueInt64()),
			Protocol:           lbSDK.ListenerProtocol(listener.Protocol.ValueString()),
			TLSCertificateName: listener.TLSCertificateName.ValueStringPointer(),
			BackendName:        listener.BackendName.ValueString(),
		})
	}
	return listenerRequests
}

func (lb *LoadBalancerModel) ConvertTLSCertificatesToSDK() []lbSDK.CreateNetworkCertificateRequest {
	if lb.TLSCertificates == nil {
		return nil
	}

	var certificateRequests []lbSDK.CreateNetworkCertificateRequest
	for _, certificate := range *lb.TLSCertificates {
		certificateRequests = append(certificateRequests, lbSDK.CreateNetworkCertificateRequest{
			Name:        certificate.Name.ValueString(),
			Description: certificate.Description.ValueStringPointer(),
			Certificate: certificate.Certificate.ValueString(),
			PrivateKey:  certificate.PrivateKey.ValueString(),
		})
	}
	return certificateRequests
}

func (lb *LoadBalancerModel) ToTerraformNetworkResource(ctx context.Context, lbResponse lbSDK.NetworkLoadBalancerResponse) LoadBalancerModel {
	healthCheckIDsNames := make(map[string]string)
	var healthCheckModels []HealthCheckModel
	for _, healthCheck := range lbResponse.HealthChecks {
		healthCheckIDsNames[healthCheck.ID] = healthCheck.Name
		healthCheckModels = append(healthCheckModels, HealthCheckModel{
			ID:                      types.StringValue(healthCheck.ID),
			Name:                    types.StringValue(healthCheck.Name),
			Description:             types.StringPointerValue(healthCheck.Description),
			Protocol:                types.StringValue(string(healthCheck.Protocol)),
			Port:                    types.Int64Value(int64(healthCheck.Port)),
			Path:                    types.StringPointerValue(healthCheck.Path),
			IntervalSeconds:         types.Int64Value(int64(healthCheck.IntervalSeconds)),
			TimeoutSeconds:          types.Int64Value(int64(healthCheck.TimeoutSeconds)),
			HealthyStatusCode:       types.Int64Value(int64(healthCheck.HealthyStatusCode)),
			HealthyThresholdCount:   types.Int64Value(int64(healthCheck.HealthyThresholdCount)),
			InitialDelaySeconds:     types.Int64Value(int64(healthCheck.InitialDelaySeconds)),
			UnhealthyThresholdCount: types.Int64Value(int64(healthCheck.UnhealthyThresholdCount)),
		})
	}

	aclModels := make([]ACLModel, len(lbResponse.ACLs))
	for i, acl := range lbResponse.ACLs {
		aclModels[i] = ACLModel{
			Action:         types.StringValue(acl.Action),
			Ethertype:      types.StringValue(string(acl.Ethertype)),
			Protocol:       types.StringValue(string(acl.Protocol)),
			Name:           types.StringPointerValue(acl.Name),
			RemoteIPPrefix: types.StringValue(acl.RemoteIPPrefix),
		}
	}

	backendIDsNames := make(map[string]string)
	var backendModels []BackendModel
	for _, backend := range lbResponse.Backends {
		backendIDsNames[backend.ID] = backend.Name
		var targets []TargetModel
		for _, target := range backend.Targets {
			targets = append(targets, TargetModel{
				Port:      types.Int64PointerValue(target.Port),
				NICID:     types.StringPointerValue(target.NicID),
				IPAddress: types.StringPointerValue(target.IPAddress),
			})
		}

		var healthCheckName *string
		if backend.HealthCheckID != nil {
			name := healthCheckIDsNames[*backend.HealthCheckID]
			healthCheckName = &name
		}

		backendModels = append(backendModels, BackendModel{
			ID:                                  types.StringValue(backend.ID),
			Name:                                types.StringValue(backend.Name),
			Description:                         types.StringPointerValue(backend.Description),
			BalanceAlgorithm:                    types.StringValue(string(backend.BalanceAlgorithm)),
			HealthCheckName:                     types.StringPointerValue(healthCheckName),
			PanicThreshold:                      types.Float64PointerValue(backend.PanicThreshold),
			CloseConnectionsOnHostHealthFailure: types.BoolPointerValue(backend.CloseConnectionsOnHostHealthFailure),
			TargetsType:                         types.StringValue(string(backend.TargetsType)),
			Targets:                             targets,
		})
	}

	tlsCertificatesIDsNames := make(map[string]string)
	var tlsCertificates []TLSCertificateModel
	for _, certificate := range lbResponse.TLSCertificates {
		tlsCertificatesIDsNames[certificate.ID] = certificate.Name
		t := TLSCertificateModel{
			Description: types.StringPointerValue(certificate.Description),
			Name:        types.StringValue(certificate.Name),
			Certificate: types.StringNull(),
			PrivateKey:  types.StringNull(),
			ID:          types.StringValue(certificate.ID),
		}

		if certificate.ExpirationDate != nil {
			t.ExpirationDate = types.StringValue(certificate.ExpirationDate.String())
		}

		tlsCertificates = append(tlsCertificates, t)
	}

	var listenerModels []ListenerModel
	for _, listener := range lbResponse.Listeners {
		lm := ListenerModel{
			ID:          types.StringValue(listener.ID),
			Name:        types.StringValue(listener.Name),
			Description: types.StringPointerValue(listener.Description),
			Port:        types.Int64Value(int64(listener.Port)),
			Protocol:    types.StringValue(string(listener.Protocol)),
		}

		if listener.TLSCertificateID != nil {
			if tlsName, exists := tlsCertificatesIDsNames[*listener.TLSCertificateID]; exists {
				lm.TLSCertificateName = types.StringValue(tlsName)
			}
		}

		if backendName, exists := backendIDsNames[listener.BackendID]; exists {
			lm.BackendName = types.StringValue(backendName)
		}

		listenerModels = append(listenerModels, lm)
	}

	loadBalancer := LoadBalancerModel{
		ID:              types.StringValue(lbResponse.ID),
		Name:            types.StringValue(lbResponse.Name),
		Description:     types.StringPointerValue(lbResponse.Description),
		SubnetpoolID:    types.StringPointerValue(lbResponse.SubnetPoolID),
		Type:            types.StringValue(lbResponse.Type),
		Visibility:      types.StringValue(string(lbResponse.Visibility)),
		VPCID:           types.StringValue(lbResponse.VPCID),
		ACLs:            &aclModels,
		Backends:        backendModels,
		HealthChecks:    &healthCheckModels,
		Listeners:       listenerModels,
		TLSCertificates: &tlsCertificates,
	}

	if lbResponse.PublicIP != nil && lbResponse.PublicIP.ExternalID != "" {
		loadBalancer.PublicIPID = types.StringValue(lbResponse.PublicIP.ExternalID)
	} else {
		loadBalancer.PublicIPID = types.StringNull()
	}

	return loadBalancer
}

func (plan ACLModel) hasACLChanges(state ACLModel) bool {
	if plan.Action.IsUnknown() || plan.Ethertype.IsUnknown() ||
		plan.Name.IsUnknown() || plan.Protocol.IsUnknown() ||
		plan.RemoteIPPrefix.IsUnknown() {
		return false
	}

	if !plan.Action.Equal(state.Action) {
		return true
	}
	if !plan.Ethertype.Equal(state.Ethertype) {
		return true
	}
	if !plan.Protocol.Equal(state.Protocol) {
		return true
	}
	if !plan.RemoteIPPrefix.Equal(state.RemoteIPPrefix) {
		return true
	}

	if !plan.Name.Equal(state.Name) {
		if (plan.Name.IsNull() || plan.Name.ValueString() == "") &&
			(state.Name.IsNull() || state.Name.ValueString() == "") {
		} else {
			return true
		}
	}

	return false
}

func (plan *LoadBalancerModel) hasACLChanges(state LoadBalancerModel) bool {
	if plan.ACLs == nil && state.ACLs == nil {
		return false
	}
	if plan.ACLs == nil || state.ACLs == nil {
		return true
	}
	if len(*plan.ACLs) != len(*state.ACLs) {
		return true
	}

	for i, planACL := range *plan.ACLs {
		if planACL.hasACLChanges((*state.ACLs)[i]) {
			return true
		}
	}

	return false
}

type LbaasNetworksListModel struct {
	LoadBalancers []lbNetworkItemModel `tfsdk:"load_balancers"`
}

type lbNetworkItemModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	PublicIPID      types.String `tfsdk:"public_ip_id"`
	SubnetpoolID    types.String `tfsdk:"subnetpool_id"`
	Type            types.String `tfsdk:"type"`
	Visibility      types.String `tfsdk:"visibility"`
	VPCID           types.String `tfsdk:"vpc_id"`
	ACLs            []ACLModel   `tfsdk:"acls"`
	Backends        types.List   `tfsdk:"backends"`
	HealthChecks    types.List   `tfsdk:"health_checks"`
	Listeners       types.List   `tfsdk:"listeners"`
	TLSCertificates types.List   `tfsdk:"tls_certificates"`
}

type backendItemModel struct {
	ID                                  types.String  `tfsdk:"id"`
	BalanceAlgorithm                    types.String  `tfsdk:"balance_algorithm"`
	Description                         types.String  `tfsdk:"description"`
	HealthCheckID                       types.String  `tfsdk:"health_check_id"`
	Name                                types.String  `tfsdk:"name"`
	Targets                             []TargetModel `tfsdk:"targets"`
	TargetsType                         types.String  `tfsdk:"targets_type"`
	PanicThreshold                      types.Float64 `tfsdk:"panic_threshold"`
	CloseConnectionsOnHostHealthFailure types.Bool    `tfsdk:"close_connections_on_host_health_failure"`
}

type networkBackendItemModel struct {
	LBID types.String `tfsdk:"lb_id"`
	backendItemModel
}

func (b backendItemModel) fromSDKBackend(lb *lbSDK.NetworkBackendResponse) backendItemModel {
	targets := make([]TargetModel, 0, len(lb.Targets))
	for _, t := range lb.Targets {
		targets = append(targets, TargetModel{
			NICID:     types.StringPointerValue(t.NicID),
			IPAddress: types.StringPointerValue(t.IPAddress),
			Port:      types.Int64PointerValue(t.Port),
		})
	}

	item := backendItemModel{
		ID:                                  types.StringValue(lb.ID),
		Name:                                types.StringValue(lb.Name),
		Description:                         types.StringPointerValue(lb.Description),
		BalanceAlgorithm:                    types.StringValue(string(lb.BalanceAlgorithm)),
		HealthCheckID:                       types.StringPointerValue(lb.HealthCheckID),
		PanicThreshold:                      types.Float64PointerValue(lb.PanicThreshold),
		CloseConnectionsOnHostHealthFailure: types.BoolPointerValue(lb.CloseConnectionsOnHostHealthFailure),
		TargetsType:                         types.StringValue(string(lb.TargetsType)),
		Targets:                             targets,
	}
	return item
}

func (plan HealthCheckModel) hasHealthCheckChanges(state HealthCheckModel) bool {
	if plan.Protocol.IsUnknown() ||
		plan.Port.IsUnknown() ||
		plan.Path.IsUnknown() ||
		plan.HealthyStatusCode.IsUnknown() ||
		plan.IntervalSeconds.IsUnknown() ||
		plan.TimeoutSeconds.IsUnknown() ||
		plan.InitialDelaySeconds.IsUnknown() ||
		plan.HealthyThresholdCount.IsUnknown() ||
		plan.UnhealthyThresholdCount.IsUnknown() {
		return false
	}

	if !plan.Protocol.Equal(state.Protocol) {
		return true
	}
	if !plan.Port.Equal(state.Port) {
		return true
	}
	if !plan.HealthyStatusCode.Equal(state.HealthyStatusCode) {
		return true
	}
	if !plan.IntervalSeconds.Equal(state.IntervalSeconds) {
		return true
	}
	if !plan.TimeoutSeconds.Equal(state.TimeoutSeconds) {
		return true
	}
	if !plan.InitialDelaySeconds.Equal(state.InitialDelaySeconds) {
		return true
	}
	if !plan.HealthyThresholdCount.Equal(state.HealthyThresholdCount) {
		return true
	}
	if !plan.UnhealthyThresholdCount.Equal(state.UnhealthyThresholdCount) {
		return true
	}

	if !plan.Path.Equal(state.Path) {
		if (plan.Path.IsNull() || plan.Path.ValueString() == "") &&
			(state.Path.IsNull() || state.Path.ValueString() == "") {
		} else {
			return true
		}
	}

	return false
}

// healthChecksToUpdate compares the health checks in the plan with those in the current state
// It returns a boolean indicating whether there are any health checks to update,
// and a slice of pointers to the health checks that need updating.
func (plan *LoadBalancerModel) healthChecksToUpdate(state LoadBalancerModel) (bool, []*HealthCheckModel) {
	if plan.HealthChecks == nil || state.HealthChecks == nil {
		return false, nil
	}

	byID := make(map[string]HealthCheckModel)
	byName := make(map[string]HealthCheckModel)
	for _, s := range *state.HealthChecks {
		if !s.ID.IsNull() && !s.ID.IsUnknown() && s.ID.ValueString() != "" {
			byID[s.ID.ValueString()] = s
		}
		if !s.Name.IsNull() && !s.Name.IsUnknown() && s.Name.ValueString() != "" {
			byName[s.Name.ValueString()] = s
		}
	}

	var updates []*HealthCheckModel
	for i := range *plan.HealthChecks {
		p := &(*plan.HealthChecks)[i]

		var st HealthCheckModel
		var ok bool

		if !p.ID.IsNull() && !p.ID.IsUnknown() && p.ID.ValueString() != "" {
			st, ok = byID[p.ID.ValueString()]
		}

		if !ok && !p.Name.IsNull() && !p.Name.IsUnknown() && p.Name.ValueString() != "" {
			st, ok = byName[p.Name.ValueString()]
		}

		if ok && p.hasHealthCheckChanges(st) {
			updates = append(updates, p)
		}
	}

	return true, updates
}

func (plan BackendModel) hasBackendFieldChanges(state BackendModel) bool {
	if plan.PanicThreshold.IsUnknown() || plan.CloseConnectionsOnHostHealthFailure.IsUnknown() {
		return false
	}
	if !plan.PanicThreshold.Equal(state.PanicThreshold) {
		return true
	}
	if !plan.CloseConnectionsOnHostHealthFailure.Equal(state.CloseConnectionsOnHostHealthFailure) {
		return true
	}
	return false
}

func (t TargetModel) hasUnknowns() bool {
	return t.NICID.IsUnknown() || t.IPAddress.IsUnknown() || t.Port.IsUnknown()
}

func (t TargetModel) equalsNormalized(o TargetModel) bool {
	if t.hasUnknowns() || o.hasUnknowns() {
		return true
	}
	if !t.Port.Equal(o.Port) {
		return false
	}
	nicA := ""
	if !t.NICID.IsNull() {
		nicA = t.NICID.ValueString()
	}
	nicB := ""
	if !o.NICID.IsNull() {
		nicB = o.NICID.ValueString()
	}
	if nicA != nicB {
		return false
	}
	ipA := ""
	if !t.IPAddress.IsNull() {
		ipA = t.IPAddress.ValueString()
	}
	ipB := ""
	if !o.IPAddress.IsNull() {
		ipB = o.IPAddress.ValueString()
	}
	if ipA != ipB {
		return false
	}
	return true
}

func (plan BackendModel) hasTargetChanges(state BackendModel) bool {
	if len(plan.Targets) != len(state.Targets) {
		return true
	}

	used := make([]bool, len(state.Targets))

	for _, pt := range plan.Targets {
		if pt.hasUnknowns() {
			continue
		}

		matched := false
		for i, st := range state.Targets {
			if used[i] {
				continue
			}
			if pt.equalsNormalized(st) {
				used[i] = true
				matched = true
				break
			}
		}
		if !matched {
			return true
		}
	}

	return false
}

// backendsToUpdate compares plan vs state and returns which backends need field
// updates and which need target updates. The same backend may appear in both slices.
func (plan *LoadBalancerModel) backendsToUpdate(state LoadBalancerModel) (fieldUpdates []*BackendModel, targetUpdates []*BackendModel) {
	if plan == nil {
		return nil, nil
	}
	byID := make(map[string]BackendModel)
	byName := make(map[string]BackendModel)
	for _, s := range state.Backends {
		if !s.ID.IsNull() && !s.ID.IsUnknown() && s.ID.ValueString() != "" {
			byID[s.ID.ValueString()] = s
		}
		if !s.Name.IsNull() && !s.Name.IsUnknown() && s.Name.ValueString() != "" {
			byName[s.Name.ValueString()] = s
		}
	}
	for i := range plan.Backends {
		pb := &plan.Backends[i]
		var sb BackendModel
		var ok bool
		if !pb.ID.IsNull() && !pb.ID.IsUnknown() && pb.ID.ValueString() != "" {
			sb, ok = byID[pb.ID.ValueString()]
		}
		if !ok && !pb.Name.IsNull() && !pb.Name.IsUnknown() && pb.Name.ValueString() != "" {
			sb, ok = byName[pb.Name.ValueString()]
		}
		if !ok {
			// New backend (create) – not an update
			continue
		}
		if pb.hasBackendFieldChanges(sb) {
			fieldUpdates = append(fieldUpdates, pb)
		}
		if pb.hasTargetChanges(sb) {
			targetUpdates = append(targetUpdates, pb)
		}
	}
	return
}

type healthCheckNotFoundError struct {
	backendName string
	hcName      string
}

func (e healthCheckNotFoundError) Error() string {
	return fmt.Sprintf("Health check with name %s not found for backend %s", e.hcName, e.backendName)
}
