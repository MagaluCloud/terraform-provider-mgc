package virtualmachines

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// nullPublicIPv4WhenNotAllocated plans "ipv4" as null on creation when the instance
// is not going to receive a public IPv4, instead of leaving it unknown.
// "allocate_public_ipv4" is write-only and never shows up in the plan, so this is
// the only way for plan consumers (e.g. policy checks over "terraform show -json")
// to tell a private instance from a public one before apply.
type nullPublicIPv4WhenNotAllocated struct{}

func NullPublicIPv4WhenNotAllocated() planmodifier.String {
	return nullPublicIPv4WhenNotAllocated{}
}

func (m nullPublicIPv4WhenNotAllocated) Description(context.Context) string {
	return "On creation, plans the value as null when allocate_public_ipv4 is not true and network_interface_id is not set."
}

func (m nullPublicIPv4WhenNotAllocated) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m nullPublicIPv4WhenNotAllocated) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.State.Raw.IsNull() || !req.PlanValue.IsUnknown() || req.Config.Raw.IsNull() {
		return
	}

	// An existing interface may already have a public IPv4 associated.
	var networkInterfaceID types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("network_interface_id"), &networkInterfaceID)...)
	if resp.Diagnostics.HasError() || !networkInterfaceID.IsNull() {
		return
	}

	var allocatePublicIPv4 types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("allocate_public_ipv4"), &allocatePublicIPv4)...)
	if resp.Diagnostics.HasError() || allocatePublicIPv4.IsUnknown() || allocatePublicIPv4.ValueBool() {
		return
	}

	resp.PlanValue = types.StringNull()
}
