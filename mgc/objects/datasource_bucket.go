package objects

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	objSdk "github.com/MagaluCloud/mgc-sdk-go/objectstorage"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

var _ datasource.DataSource = &objectStorageBucketDataSource{}

type objectStorageBucketDataSource struct {
	buckets  objSdk.BucketService
	region   string
	endpoint string
}

type ObjectStorageBucketDataSourceModel struct {
	Bucket     types.String `tfsdk:"bucket"`
	Versioning types.Bool   `tfsdk:"versioning"`
	Lock       types.Bool   `tfsdk:"lock"`
	Policy     types.String `tfsdk:"policy"`
	CORS       types.Object `tfsdk:"cors"`
	Region     types.String `tfsdk:"region"`
	URL        types.String `tfsdk:"url"`
}

func NewObjectStorageBucketDataSource() datasource.DataSource {
	return &objectStorageBucketDataSource{}
}

func (d *objectStorageBucketDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object_storage_bucket"
}

func (d *objectStorageBucketDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	a, err := objSdk.New(&dataConfig.CoreConfig, dataConfig.AccessKey, dataConfig.SecretKey, objSdk.WithEndpoint(endpoint))
	if err != nil {
		resp.Diagnostics.AddError("Failed to configure object storage", "Invalid credentials data")
		return
	}
	d.buckets = a.Buckets()
	d.region = dataConfig.Region
	d.endpoint = endpoint.String()
}

func (d *objectStorageBucketDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about an object storage bucket.",
		Attributes: map[string]schema.Attribute{
			"bucket": schema.StringAttribute{
				Required:    true,
				Description: "Name of the bucket.",
			},
			"versioning": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether versioning is enabled for this bucket.",
			},
			"lock": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether object lock is enabled for this bucket.",
			},
			"policy": schema.StringAttribute{
				Computed:    true,
				Description: "Bucket policy document as a JSON string.",
			},
			"cors": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "CORS configuration for the bucket.",
				Attributes: map[string]schema.Attribute{
					"allowed_headers": schema.ListAttribute{
						ElementType: types.StringType,
						Computed:    true,
						Description: "Allowed headers for CORS requests.",
					},
					"allowed_methods": schema.ListAttribute{
						ElementType: types.StringType,
						Computed:    true,
						Description: "Allowed HTTP methods for CORS requests.",
					},
					"allowed_origins": schema.ListAttribute{
						ElementType: types.StringType,
						Computed:    true,
						Description: "Allowed origins for CORS requests.",
					},
					"expose_headers": schema.ListAttribute{
						ElementType: types.StringType,
						Computed:    true,
						Description: "Headers exposed to the browser for CORS requests.",
					},
					"max_age_seconds": schema.Int64Attribute{
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

func (d *objectStorageBucketDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ObjectStorageBucketDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := data.Bucket.ValueString()

	exists, err := d.buckets.Exists(ctx, bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error checking bucket existence",
			fmt.Sprintf("Could not check if bucket %s exists: %s", bucketName, err.Error()),
		)
		return
	}

	if !exists {
		resp.Diagnostics.AddError(
			"Bucket not found",
			fmt.Sprintf("Bucket %s does not exist", bucketName),
		)
		return
	}

	versioningStatus, err := d.buckets.GetVersioningStatus(ctx, bucketName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading versioning status",
			fmt.Sprintf("Could not read versioning status for bucket %s: %s", bucketName, err.Error()),
		)
		return
	}
	if versioningStatus != nil && versioningStatus.Status == "Enabled" {
		data.Versioning = types.BoolValue(true)
	} else {
		data.Versioning = types.BoolValue(false)
	}

	lockStatus, err := d.buckets.GetBucketLockStatus(ctx, bucketName)
	if err != nil {
		data.Lock = types.BoolValue(false)
	} else {
		data.Lock = types.BoolValue(lockStatus)
	}

	policy, err := d.buckets.GetPolicy(ctx, bucketName)
	if err != nil {
		data.Policy = types.StringNull()
	} else if policy != nil {
		policyJSON, err := json.Marshal(policy)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error serializing bucket policy",
				fmt.Sprintf("Could not serialize policy for bucket %s: %s", bucketName, err.Error()),
			)
			return
		}
		data.Policy = types.StringValue(string(policyJSON))
	} else {
		data.Policy = types.StringNull()
	}

	corsConfig, err := d.buckets.GetCORS(ctx, bucketName)
	if err != nil || corsConfig == nil || len(corsConfig.CORSRules) == 0 {
		data.CORS = types.ObjectNull(map[string]attr.Type{
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
		data.CORS = corsObj
	}

	data.Region = types.StringValue(d.region)
	data.URL = types.StringValue(fmt.Sprintf("%s/%s", d.endpoint, bucketName))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
