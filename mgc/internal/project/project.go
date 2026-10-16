package project

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/importid"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var idRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

const description = "ID of the project the resource belongs to. When omitted, the tenant's default project is used."

var Namespace = importid.Namespace{Name: "project", Validate: validateID}

func validateID(v string) error {
	if !idRegex.MatchString(v) {
		return fmt.Errorf("%q is not a valid project ID (UUID)", v)
	}
	return nil
}

func idValidator() validator.String {
	return stringvalidator.RegexMatches(idRegex, "must be a valid project ID (UUID)")
}

func ResourceAttribute() rschema.StringAttribute {
	return rschema.StringAttribute{
		Description:   description + " Changing it forces a new resource; removing it keeps the current project.",
		Optional:      true,
		Computed:      true,
		PlanModifiers: []planmodifier.String{keepWhenUnset{}},
		Validators:    []validator.String{idValidator()},
	}
}

func DataSourceAttribute() dschema.StringAttribute {
	return dschema.StringAttribute{
		Description: description,
		Optional:    true,
		Validators:  []validator.String{idValidator()},
	}
}

func IdentityAttribute() identityschema.StringAttribute {
	return identityschema.StringAttribute{
		Description:       description,
		OptionalForImport: true,
	}
}

func CallOptions(projectID types.String) clientSDK.CallOptions {
	if projectID.IsNull() || projectID.IsUnknown() {
		return clientSDK.CallOptions{}
	}
	return clientSDK.CallOptions{ProjectID: strings.TrimSpace(projectID.ValueString())}
}

func ParseSDKError(err error, projectID types.String) (msg, detail string) {
	msg, detail = utils.ParseSDKError(err)

	var httpErr *clientSDK.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusNotFound {
		return msg, detail
	}

	if projectID.IsNull() || projectID.IsUnknown() || strings.TrimSpace(projectID.ValueString()) == "" {
		return msg, detail + "\n\nThe resource was not found in the tenant's default project. " +
			"If it belongs to a project, set project_id, or import it as project/{project_id}/resource/{id}."
	}
	return msg, detail + fmt.Sprintf("\n\nThe resource was not found in project %q. "+
		"Check that project_id is the project the resource belongs to.", projectID.ValueString())
}
