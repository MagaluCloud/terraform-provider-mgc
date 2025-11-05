package objects

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	objSdk "github.com/MagaluCloud/mgc-sdk-go/objectstorage"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

var _ datasource.DataSource = &objectStorageBucketsDataSource{}

type objectStorageBucketsDataSource struct {
	buckets objSdk.BucketService
}

type ObjectStorageBucketsDataSourceModel struct {
	Buckets []types.String `tfsdk:"buckets"`
}

func NewObjectStorageBucketsDataSource() datasource.DataSource {
	return &objectStorageBucketsDataSource{}
}

func (d *objectStorageBucketsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object_storage_buckets"
}

func (d *objectStorageBucketsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
}

func (d *objectStorageBucketsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a list of all object storage bucket names.",
		Attributes: map[string]schema.Attribute{
			"buckets": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of bucket names.",
			},
		},
	}
}

func (d *objectStorageBucketsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ObjectStorageBucketsDataSourceModel

	buckets, err := d.buckets.List(ctx, objSdk.BucketListOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing buckets",
			"Could not list buckets: "+err.Error(),
		)
		return
	}

	data.Buckets = make([]types.String, len(buckets))
	for i, bucket := range buckets {
		data.Buckets[i] = types.StringValue(bucket.Name)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
