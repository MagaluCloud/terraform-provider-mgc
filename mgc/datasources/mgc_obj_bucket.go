package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	tfutil "github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var _ datasource.DataSource = &DatasourceBucket{}

type DatasourceBucket struct {
	s3Client *minio.Client
}
type BucketDetailModel struct {
	Name       types.String `tfsdk:"name"`
	Versioning types.String `tfsdk:"versioning"`
	MFADelete  types.String `tfsdk:"mfadelete"`
	Policy     types.String `tfsdk:"policy"`
}

func NewDatasourceBucket() datasource.DataSource {
	return &DatasourceBucket{}
}

func (r *DatasourceBucket) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object_storage_bucket"
}

func (r *DatasourceBucket) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *DatasourceBucket) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Bucket name",
			},
			"versioning": schema.StringAttribute{
				Computed:    true,
				Description: "Versioning status",
			},
			"mfadelete": schema.StringAttribute{
				Computed:    true,
				Description: "MFA Delete",
			},
			"policy": schema.StringAttribute{
				Computed:    true,
				Description: "Access policy",
			},
		},
	}
	resp.Schema.Description = "Get details of bucket."
}

func (r *DatasourceBucket) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BucketDetailModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	versioning, err := r.s3Client.GetBucketVersioning(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get versioning", err.Error())
		return
	}
	data.MFADelete = types.StringValue(versioning.MFADelete)
	data.Versioning = types.StringValue(versioning.Status)

	policy, err := r.s3Client.GetBucketPolicy(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get access policy", err.Error())
		return
	}

	data.Policy = types.StringValue(policy)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
