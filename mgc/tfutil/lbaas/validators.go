package tfutil

// This package contains all validators specific to Load Balancer as a Service (LBaaS) resources.
// Each validator is implemented in its own file for better organization and maintainability.

// Available validators:
// - VisibilityValidator: Validates public_ip_id configuration based on visibility (external/internal)
// - TargetValidator: Validates target configuration based on targets_type (raw/instance)

/*
Usage Examples:

1. TargetValidator - Applied to targets list in backend schema:
   ```go
   "targets": schema.ListNestedAttribute{
       Description: "The targets for this backend.",
       Required:    true,
       Validators: []validator.List{
           lbaasvalidators.TargetValidator{},
       },
       // ... rest of schema
   }
   ```

2. VisibilityValidator - Applied to visibility field in load balancer schema:
   ```go
   "visibility": schema.StringAttribute{
       Description: "The visibility of the load balancer.",
       Required:    true,
       Validators: []validator.String{
           stringvalidator.OneOf("internal", "external"),
           lbaasvalidators.VisibilityValidator{},
       },
       // ... rest of schema
   }
   ```

Validation Rules:

TargetValidator:
- For targets_type = "raw":
  * ip_address is required and must not be empty
  * nic_id must be null or empty
  * port is always required (validated by schema)

- For targets_type = "instance":
  * nic_id is required and must not be empty
  * ip_address must be null or empty
  * port is always required (validated by schema)

VisibilityValidator:
- For visibility = "external":
  * public_ip_id is required and must not be empty

- For visibility = "internal":
  * public_ip_id must be null or empty

Implementation Details:

Both validators access individual attributes directly from the Terraform configuration
rather than converting entire objects to minimal models. This approach:
- Avoids value conversion errors when the parent object has more fields than needed
- Provides better performance by accessing only required attributes
- Maintains clean separation between validation logic and data models

Both validators provide clear error messages when validation fails,
helping users understand exactly what configuration is required.
*/
