package project

import (
	"context"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/importid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type IdentityModel struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
}

func IdentitySchema() identityschema.Schema {
	return identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "ID of the resource.",
				RequiredForImport: true,
			},
			"project_id": IdentityAttribute(),
		},
	}
}

func ResolveImport(ctx context.Context, req resource.ImportStateRequest) (IdentityModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	if req.ID == "" {
		if req.Identity == nil {
			diags.AddError("Invalid import ID", "An import ID or an identity must be provided.")
			return IdentityModel{}, diags
		}
		var model IdentityModel
		diags.Append(req.Identity.Get(ctx, &model)...)
		return model, diags
	}

	parsed, err := importid.Parse(req.ID, Namespace)
	if err != nil {
		diags.AddError("Invalid import ID", err.Error())
		return IdentityModel{}, diags
	}

	return IdentityModel{
		ID:        types.StringValue(parsed.ID),
		ProjectID: parsed.Value(Namespace),
	}, diags
}

func SetIdentity(ctx context.Context, identity *tfsdk.ResourceIdentity, id, projectID types.String) diag.Diagnostics {
	if identity == nil {
		return nil
	}
	return identity.Set(ctx, IdentityModel{ID: id, ProjectID: projectID})
}
