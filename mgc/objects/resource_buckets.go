package objects

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	objSdk "github.com/MagaluCloud/mgc-sdk-go/objectstorage"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

type ObjectStorageBucket struct {
	Bucket     types.String `tfsdk:"bucket"`
	Versioning types.Bool   `tfsdk:"versioning"`
	Lock       types.Bool   `tfsdk:"lock"`
	Policy     types.String `tfsdk:"policy"`
	CORS       types.Object `tfsdk:"cors"`
	Region     types.String `tfsdk:"region"`
	URL        types.String `tfsdk:"url"`
}

type CORS struct {
	AllowedHeaders types.List  `tfsdk:"allowed_headers"`
	AllowedMethods types.List  `tfsdk:"allowed_methods"`
	AllowedOrigins types.List  `tfsdk:"allowed_origins"`
	ExposeHeaders  types.List  `tfsdk:"expose_headers"`
	MaxAgeSeconds  types.Int64 `tfsdk:"max_age_seconds"`
}

func NewObjectStorageBucketsResource() resource.Resource {
	return &objectStorageBuckets{}
}

type objectStorageBuckets struct {
	buckets  objSdk.BucketService
	region   string
	endpoint string
}

func (r *objectStorageBuckets) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object_storage_buckets"
}

func (r *objectStorageBuckets) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to configure provider", "Invalid provider data")
		return
	}

	endpoint, err := utils.RegionToS3Url(dataConfig.Region, dataConfig.Env)
	if err != nil {
		resp.Diagnostics.AddError("Invalid region/env for object storage", endpoint.String())
		return
	}

	a, err := objSdk.New(&dataConfig.CoreConfig, dataConfig.KeyPairID, dataConfig.KeyPairSecret, objSdk.WithEndpoint(endpoint))
	if err != nil {
		resp.Diagnostics.AddError("Failed to configure object storage", "Invalid credentials data")
		return
	}
	r.buckets = a.Buckets()
	r.region = dataConfig.Region
	r.endpoint = endpoint.String()
}

func (r *objectStorageBuckets) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An object storage bucket.",
		Attributes: map[string]schema.Attribute{
			"bucket": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Name of the bucket to be created.",
			},
			"versioning": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable versioning for this bucket.",
				Default:     booldefault.StaticBool(false),
			},
			"lock": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable object lock for this bucket.",
				Default:     booldefault.StaticBool(false),
			},
			"policy": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Bucket policy document as a JSON string.",
			},
			"cors": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "CORS configuration for the bucket.",
				Attributes: map[string]schema.Attribute{
					"allowed_headers": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						Description: "Allowed headers for CORS requests.",
					},
					"allowed_methods": schema.ListAttribute{
						ElementType: types.StringType,
						Required:    true,
						Description: "Allowed HTTP methods for CORS requests.",
					},
					"allowed_origins": schema.ListAttribute{
						ElementType: types.StringType,
						Required:    true,
						Description: "Allowed origins for CORS requests.",
					},
					"expose_headers": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						Description: "Headers exposed to the browser for CORS requests.",
					},
					"max_age_seconds": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "Maximum age in seconds for CORS preflight cache.",
					},
				},
			},
			"region": schema.StringAttribute{
				Computed:    true,
				Description: "The region where the bucket is located.",
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "The URL endpoint of the bucket.",
			},
		},
	}
}

func (r *objectStorageBuckets) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ObjectStorageBucket
	diags := req.Plan.Get(ctx, &plan)
	if diags.HasError() {
		resp.Diagnostics = diags
		return
	}

	bucketName := plan.Bucket.ValueString()

	if err := r.buckets.Create(ctx, bucketName); err != nil {
		resp.Diagnostics.AddError(
			"Error creating bucket",
			fmt.Sprintf("Could not create bucket %s: %s", bucketName, err.Error()),
		)
		return
	}

	if !plan.Versioning.IsNull() && plan.Versioning.ValueBool() {
		if err := r.buckets.EnableVersioning(ctx, bucketName); err != nil {
			resp.Diagnostics.AddError(
				"Error enabling versioning",
				fmt.Sprintf("Could not enable versioning for bucket %s: %s", bucketName, err.Error()),
			)
			_ = r.buckets.Delete(ctx, bucketName, false)
			return
		}
	}

	if !plan.Lock.IsNull() && plan.Lock.ValueBool() {
		if err := r.buckets.LockBucket(ctx, bucketName, 1, "days"); err != nil {
			resp.Diagnostics.AddError(
				"Error locking bucket",
				fmt.Sprintf("Could not enable object lock for bucket %s: %s", bucketName, err.Error()),
			)
			_ = r.buckets.Delete(ctx, bucketName, false)
			return
		}
	}

	if !plan.Policy.IsNull() && !plan.Policy.IsUnknown() && plan.Policy.ValueString() != "" {
		var policy objSdk.Policy
		if err := json.Unmarshal([]byte(plan.Policy.ValueString()), &policy); err != nil {
			resp.Diagnostics.AddError(
				"Error parsing bucket policy",
				fmt.Sprintf("Could not parse bucket policy JSON: %s", err.Error()),
			)
			_ = r.buckets.Delete(ctx, bucketName, false)
			return
		}

		if err := r.buckets.SetPolicy(ctx, bucketName, &policy); err != nil {
			resp.Diagnostics.AddError(
				"Error setting bucket policy",
				fmt.Sprintf("Could not set bucket policy for %s: %s", bucketName, err.Error()),
			)
			_ = r.buckets.Delete(ctx, bucketName, false)
			return
		}
	}

	if !plan.CORS.IsNull() && !plan.CORS.IsUnknown() {
		var corsData CORS
		diags = plan.CORS.As(ctx, &corsData, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			_ = r.buckets.Delete(ctx, bucketName, false)
			return
		}

		corsConfig, err := convertToCORSConfiguration(ctx, corsData)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing CORS configuration",
				fmt.Sprintf("Could not parse CORS configuration: %s", err.Error()),
			)
			_ = r.buckets.Delete(ctx, bucketName, false)
			return
		}

		if err := r.buckets.SetCORS(ctx, bucketName, corsConfig); err != nil {
			resp.Diagnostics.AddError(
				"Error setting CORS configuration",
				fmt.Sprintf("Could not set CORS configuration for bucket %s: %s", bucketName, err.Error()),
			)
			_ = r.buckets.Delete(ctx, bucketName, false)
			return
		}
	} else if plan.CORS.IsUnknown() {
		plan.CORS = types.ObjectNull(map[string]attr.Type{
			"allowed_headers": types.ListType{ElemType: types.StringType},
			"allowed_methods": types.ListType{ElemType: types.StringType},
			"allowed_origins": types.ListType{ElemType: types.StringType},
			"expose_headers":  types.ListType{ElemType: types.StringType},
			"max_age_seconds": types.Int64Type,
		})
	}

	plan.Region = types.StringValue(r.region)
	plan.URL = types.StringValue(fmt.Sprintf("%s/%s", r.endpoint, bucketName))

	if plan.Policy.IsNull() || plan.Policy.IsUnknown() {
		plan.Policy = types.StringValue("")
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *objectStorageBuckets) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ObjectStorageBucket
	diags := req.State.Get(ctx, &state)
	if diags.HasError() {
		resp.Diagnostics = diags
		return
	}

	bucketName := state.Bucket.ValueString()

	exists, err := r.buckets.Exists(ctx, bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error checking bucket existence",
			fmt.Sprintf("Could not check if bucket %s exists: %s", bucketName, err.Error()),
		)
		return
	}

	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	versioningStatus, err := r.buckets.GetVersioningStatus(ctx, bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading versioning status",
			fmt.Sprintf("Could not read versioning status for bucket %s: %s", bucketName, err.Error()),
		)
		return
	}
	if versioningStatus != nil && versioningStatus.Status == "Enabled" {
		state.Versioning = types.BoolValue(true)
	} else {
		state.Versioning = types.BoolValue(false)
	}

	lockStatus, err := r.buckets.GetBucketLockStatus(ctx, bucketName)
	if err != nil {
		state.Lock = types.BoolValue(false)
	} else {
		state.Lock = types.BoolValue(lockStatus)
	}

	policy, err := r.buckets.GetPolicy(ctx, bucketName)
	if err != nil {
		state.Policy = types.StringValue("")
	} else if policy != nil {
		policyJSON, err := json.Marshal(policy)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error serializing bucket policy",
				fmt.Sprintf("Could not serialize policy for bucket %s: %s", bucketName, err.Error()),
			)
			return
		}
		state.Policy = types.StringValue(string(policyJSON))
	} else {
		state.Policy = types.StringValue("")
	}

	corsConfig, err := r.buckets.GetCORS(ctx, bucketName)
	if err != nil || corsConfig == nil || len(corsConfig.CORSRules) == 0 {
		state.CORS = types.ObjectNull(map[string]attr.Type{
			"allowed_headers": types.ListType{ElemType: types.StringType},
			"allowed_methods": types.ListType{ElemType: types.StringType},
			"allowed_origins": types.ListType{ElemType: types.StringType},
			"expose_headers":  types.ListType{ElemType: types.StringType},
			"max_age_seconds": types.Int64Type,
		})
	} else {
		tfCORS, err := convertFromCORSConfiguration(ctx, corsConfig)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error converting CORS configuration",
				fmt.Sprintf("Could not convert CORS configuration: %s", err.Error()),
			)
			return
		}
		corsObj, diag := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"allowed_headers": types.ListType{ElemType: types.StringType},
			"allowed_methods": types.ListType{ElemType: types.StringType},
			"allowed_origins": types.ListType{ElemType: types.StringType},
			"expose_headers":  types.ListType{ElemType: types.StringType},
			"max_age_seconds": types.Int64Type,
		}, tfCORS)
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
		state.CORS = corsObj
	}

	state.Region = types.StringValue(r.region)
	state.URL = types.StringValue(fmt.Sprintf("%s/%s", r.endpoint, bucketName))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *objectStorageBuckets) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ObjectStorageBucket
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ObjectStorageBucket
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := plan.Bucket.ValueString()

	if !plan.Versioning.Equal(state.Versioning) {
		if plan.Versioning.ValueBool() {
			if err := r.buckets.EnableVersioning(ctx, bucketName); err != nil {
				resp.Diagnostics.AddError(
					"Error enabling versioning",
					fmt.Sprintf("Could not enable versioning for bucket %s: %s", bucketName, err.Error()),
				)
				return
			}
		} else {
			if err := r.buckets.SuspendVersioning(ctx, bucketName); err != nil {
				resp.Diagnostics.AddError(
					"Error suspending versioning",
					fmt.Sprintf("Could not suspend versioning for bucket %s: %s", bucketName, err.Error()),
				)
				return
			}
		}
	}

	if !plan.Lock.Equal(state.Lock) {
		if plan.Lock.ValueBool() {
			if err := r.buckets.LockBucket(ctx, bucketName, 1, "days"); err != nil {
				resp.Diagnostics.AddError(
					"Error locking bucket",
					fmt.Sprintf("Could not enable object lock for bucket %s: %s", bucketName, err.Error()),
				)
				return
			}
		} else {
			if err := r.buckets.UnlockBucket(ctx, bucketName); err != nil {
				resp.Diagnostics.AddError(
					"Error unlocking bucket",
					fmt.Sprintf("Could not disable object lock for bucket %s: %s", bucketName, err.Error()),
				)
				return
			}
		}
	}

	if plan.Policy.IsUnknown() {
		plan.Policy = state.Policy
	}

	if !plan.Policy.Equal(state.Policy) {
		if plan.Policy.IsNull() || plan.Policy.ValueString() == "" {
			if err := r.buckets.DeletePolicy(ctx, bucketName); err != nil {
				resp.Diagnostics.AddError(
					"Error deleting bucket policy",
					fmt.Sprintf("Could not delete policy for bucket %s: %s", bucketName, err.Error()),
				)
				return
			}
		} else {
			var policy objSdk.Policy
			if err := json.Unmarshal([]byte(plan.Policy.ValueString()), &policy); err != nil {
				resp.Diagnostics.AddError(
					"Error parsing bucket policy",
					fmt.Sprintf("Could not parse bucket policy JSON: %s", err.Error()),
				)
				return
			}

			if err := r.buckets.SetPolicy(ctx, bucketName, &policy); err != nil {
				resp.Diagnostics.AddError(
					"Error setting bucket policy",
					fmt.Sprintf("Could not set bucket policy for %s: %s", bucketName, err.Error()),
				)
				return
			}
		}
	}

	if plan.CORS.IsUnknown() {
		plan.CORS = state.CORS
	} else if !plan.CORS.Equal(state.CORS) {
		if plan.CORS.IsNull() {
			if err := r.buckets.DeleteCORS(ctx, bucketName); err != nil {
				resp.Diagnostics.AddError(
					"Error deleting CORS configuration",
					fmt.Sprintf("Could not delete CORS configuration for bucket %s: %s", bucketName, err.Error()),
				)
				return
			}
		} else {
			var corsData CORS
			diags := plan.CORS.As(ctx, &corsData, basetypes.ObjectAsOptions{
				UnhandledNullAsEmpty:    true,
				UnhandledUnknownAsEmpty: true,
			})
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}

			corsConfig, err := convertToCORSConfiguration(ctx, corsData)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error parsing CORS configuration",
					fmt.Sprintf("Could not parse CORS configuration: %s", err.Error()),
				)
				return
			}

			if err := r.buckets.SetCORS(ctx, bucketName, corsConfig); err != nil {
				resp.Diagnostics.AddError(
					"Error setting CORS configuration",
					fmt.Sprintf("Could not set CORS configuration for bucket %s: %s", bucketName, err.Error()),
				)
				return
			}
		}
	}

	plan.Region = types.StringValue(r.region)
	plan.URL = types.StringValue(fmt.Sprintf("%s/%s", r.endpoint, bucketName))

	if plan.Policy.IsNull() || plan.Policy.IsUnknown() {
		plan.Policy = types.StringValue("")
	}

	diags := resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *objectStorageBuckets) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ObjectStorageBucket
	diags := req.State.Get(ctx, &state)
	if diags.HasError() {
		resp.Diagnostics = diags
		return
	}

	bucketName := state.Bucket.ValueString()

	exists, err := r.buckets.Exists(ctx, bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error checking bucket existence",
			fmt.Sprintf("Could not check if bucket %s exists: %s", bucketName, err.Error()),
		)
		return
	}

	if !exists {
		return
	}

	if err := r.buckets.Delete(ctx, bucketName, false); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting bucket",
			fmt.Sprintf("Could not delete bucket %s: %s", bucketName, err.Error()),
		)
		return
	}
}

func (r *objectStorageBuckets) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("bucket"), req, resp)
}
