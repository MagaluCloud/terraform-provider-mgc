package lbaas

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	ID             types.String `tfsdk:"id"`
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
	ID        types.String `tfsdk:"id"`
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
