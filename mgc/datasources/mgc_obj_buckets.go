package datasources

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DatasourceBuckets{}

type DatasourceBuckets struct {
	s3Client *minio.Client
}

type BucketModel struct {
	Name         types.String `tfsdk:"name"`
	CreationDate types.String `tfsdk:"creation_date"`
}

type BucketsModel struct {
	Buckets []BucketModel `tfsdk:"buckets"`
}

func NewDatasourceBuckets() datasource.DataSource {
	return &DatasourceBuckets{}
}

func (r *DatasourceBuckets) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object_storage_buckets"
}

func (r *DatasourceBuckets) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerConfig := req.ProviderData.(tfutil.DataConfig)

	s3Url, err := tfutil.RegionToS3Url(providerConfig.Region, providerConfig.Env)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create s3 client", err.Error())
		return
	}

	minioClient, err := minio.New(
		s3Url,
		&minio.Options{
			Creds: credentials.NewStaticV4(
				providerConfig.Keypair.KeyID,
				providerConfig.Keypair.KeySecret,
				""),
			Secure: true,
		})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create s3 client", err.Error())
		return
	}

	r.s3Client = minioClient

}

func (r *DatasourceBuckets) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"buckets": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of ssh-keys.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Bucket name",
						},
						"creation_date": schema.StringAttribute{
							Computed:    true,
							Description: "Bucket creation date",
						},
					},
				},
			},
		},
	}
	resp.Schema.Description = "Get all buckets."
}

func (r *DatasourceBuckets) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BucketsModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	buckets, err := r.s3Client.ListBuckets(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get buckets", err.Error())
		return
	}

	for _, bucket := range buckets {
		data.Buckets = append(data.Buckets, BucketModel{
			Name:         types.StringValue(bucket.Name),
			CreationDate: types.StringValue(bucket.CreationDate.Format(time.RFC3339)),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
