package tags

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ planmodifier.String = idFromAttributes{}

// idFromAttributes plans the id from the names it is built from. Tags and values
// are renamed in place, so UseStateForUnknown would promise the old id and the
// apply would fail with an inconsistent result. While any of the names is still
// unknown, the id stays unknown too.
type idFromAttributes struct {
	attributes []string
	build      func(names []string) string
}

func (m idFromAttributes) Description(_ context.Context) string {
	return "The id follows the names it is built from."
}

func (m idFromAttributes) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m idFromAttributes) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// A destroy plans no id.
	if req.Plan.Raw.IsNull() {
		return
	}

	names := make([]string, 0, len(m.attributes))
	for _, attribute := range m.attributes {
		var value types.String
		resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root(attribute), &value)...)
		if resp.Diagnostics.HasError() || value.IsUnknown() || value.IsNull() {
			return
		}
		names = append(names, value.ValueString())
	}

	resp.PlanValue = types.StringValue(m.build(names))
}

func tagIDFromName() planmodifier.String {
	return idFromAttributes{
		attributes: []string{"name"},
		build:      func(names []string) string { return names[0] },
	}
}

func tagValueIDFromNames() planmodifier.String {
	return idFromAttributes{
		attributes: []string{"tag_name", "name"},
		build:      func(names []string) string { return tagValueID(names[0], names[1]) },
	}
}
