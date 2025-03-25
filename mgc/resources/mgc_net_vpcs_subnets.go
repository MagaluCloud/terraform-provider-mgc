package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
)

type mgcNetworkVpcsSubnetsModel struct {
	ID               types.String   `tfsdk:"id"`
	CidrBlock        types.String   `tfsdk:"cidr_block"`
	Description      types.String   `tfsdk:"description"`
	DnsNameservers   []types.String `tfsdk:"dns_nameservers"`
	IpVersion        types.String   `tfsdk:"ip_version"`
	Name             types.String   `tfsdk:"name"`
	SubnetpoolId     types.String   `tfsdk:"subnetpool_id"`
	VpcId            types.String   `tfsdk:"vpc_id"`
	AvailabilityZone types.String   `tfsdk:"availability_zone"`
}

type mgcNetworkVpcsSubnetsResource struct {
	networkVpcsSubnets netSDK.VPCService
	networkSubnets     netSDK.SubnetService
	region             string
}

func NewNetworkVpcsSubnetsResource() resource.Resource {
	return &mgcNetworkVpcsSubnetsResource{}
}

func (r *mgcNetworkVpcsSubnetsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_vpcs_subnets"
}

func (r *mgcNetworkVpcsSubnetsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network VPC Subnet",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the VPC subnet",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cidr_block": schema.StringAttribute{
				Description: "The CIDR block of the VPC subnet. Example: '192.168.1.0/24', '0.0.0.0/0', '::/0' or '2001:db8::/32'",
				Required:    true,
				Validators: []validator.String{
					tfutil.CidrValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the VPC subnet",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"dns_nameservers": schema.ListAttribute{
				Description: "The DNS nameservers of the VPC subnet",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"ip_version": schema.StringAttribute{
				Description: "Network protocol version. Allowed values: 'IPv4' or 'IPv6'. Example: 'IPv4'",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("IPv4", "IPv6"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the VPC subnet",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subnetpool_id": schema.StringAttribute{
				Description: "The ID of the subnet pool",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Description: "The ID of the VPC",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"availability_zone": schema.StringAttribute{
				Description: "The availability zone of the VPC subnet",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					tfutil.ReplaceIfChangeAndNotIsNotSetOnPlan{},
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *mgcNetworkVpcsSubnetsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkVpcsSubnets = netSDK.New(&dataConfig.CoreConfig).VPCs()
	r.networkSubnets = netSDK.New(&dataConfig.CoreConfig).Subnets()
	r.region = dataConfig.Region
}

func (r *mgcNetworkVpcsSubnetsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	data := mgcNetworkVpcsSubnetsModel{}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dnsCreateParam := []string{}
	for _, dns := range data.DnsNameservers {
		dnsCreateParam = append(dnsCreateParam, dns.ValueString())
	}

	createParam := netSDK.SubnetCreateRequest{
		CIDRBlock:      data.CidrBlock.ValueString(),
		Description:    data.Description.ValueStringPointer(),
		DNSNameservers: &dnsCreateParam,
		IPVersion:      convertIPStringToIPVersion(data.IpVersion.ValueString()),
		Name:           data.Name.ValueString(),
		SubnetPoolID:   data.SubnetpoolId.ValueStringPointer(),
	}

	var azparsed *string
	if !data.AvailabilityZone.IsUnknown() {
		az, err := tfutil.ConvertAvailabilityZoneToXZone(data.AvailabilityZone.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid Availability Zone", err.Error())
			return
		}
		azparsed = &az
	}
	subnetID, err := r.networkVpcsSubnets.CreateSubnet(ctx, data.VpcId.ValueString(), createParam, netSDK.SubnetCreateOptions{
		Zone: azparsed,
	})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	createdSubnet, err := r.networkSubnets.Get(ctx, subnetID)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.AvailabilityZone = types.StringValue(tfutil.ConvertXZoneToAvailabilityZone(r.region, createdSubnet.Zone))
	data.ID = types.StringValue(subnetID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *mgcNetworkVpcsSubnetsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := &mgcNetworkVpcsSubnetsModel{}
	resp.Diagnostics.Append(req.State.Get(ctx, data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subnet, err := r.networkSubnets.Get(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	dnsNameServers := []types.String{}
	for _, dns := range subnet.DNSNameservers {
		dnsNameServers = append(dnsNameServers, types.StringPointerValue(&dns))
	}

	data.DnsNameservers = dnsNameServers
	data.CidrBlock = types.StringPointerValue(&subnet.CIDRBlock)
	data.Description = types.StringPointerValue(subnet.Description)
	data.IpVersion = types.StringValue(subnet.IPVersion)
	data.Name = types.StringPointerValue(subnet.Name)
	data.SubnetpoolId = types.StringPointerValue(&subnet.SubnetPoolID)
	data.VpcId = types.StringValue(subnet.VPCID)
	data.AvailabilityZone = types.StringValue(tfutil.ConvertXZoneToAvailabilityZone(r.region, subnet.Zone))
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *mgcNetworkVpcsSubnetsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	data := mgcNetworkVpcsSubnetsModel{}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dnsServers := []string{}
	for _, dns := range data.DnsNameservers {
		dnsServers = append(dnsServers, dns.ValueString())
	}
	subnetUpdateParams := netSDK.SubnetPatchRequest{
		DNSNameservers: &dnsServers,
	}

	_, err := r.networkSubnets.Update(ctx, data.ID.ValueString(), subnetUpdateParams)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *mgcNetworkVpcsSubnetsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	data := mgcNetworkVpcsSubnetsModel{}
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.networkSubnets.Delete(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
}

func convertIPStringToIPVersion(ipVersion string) int {
	if ipVersion == "IPv4" {
		return 4
	}
	return 6
}

func (r *mgcNetworkVpcsSubnetsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, &mgcNetworkVpcsSubnetsModel{
		ID: types.StringValue(req.ID),
	})...)
}
