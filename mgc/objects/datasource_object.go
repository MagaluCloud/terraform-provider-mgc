package objects

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	objSdk "github.com/MagaluCloud/mgc-sdk-go/objectstorage"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

var _ datasource.DataSource = &ObjectStorageObjectDataSource{}

type ObjectStorageObjectDataSource struct {
	objects  objSdk.ObjectService
	region   string
	endpoint string
}

type ObjectStorageObjectDataSourceModel struct {
	Bucket       types.String `tfsdk:"bucket"`
	Key          types.String `tfsdk:"key"`
	ContentType  types.String `tfsdk:"content_type"`
	ETag         types.String `tfsdk:"etag"`
	Size         types.Int64  `tfsdk:"size"`
	LastModified types.String `tfsdk:"last_modified"`
	LockStatus   types.Bool   `tfsdk:"lock_status"`
}

func NewObjectStorageObjectDataSource() datasource.DataSource {
	return &ObjectStorageObjectDataSource{}
}

func (d *ObjectStorageObjectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object_storage_object"
}

func (d *ObjectStorageObjectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to configure provider", "Invalid provider data")
		return
	}

	endpoint, err := resolveS3Endpoint(dataConfig)
	if err != nil {
		resp.Diagnostics.AddError("Invalid region/env for object storage", err.Error())
		return
	}

	a, err := objSdk.New(&dataConfig.CoreConfig, dataConfig.KeyPairID, dataConfig.KeyPairSecret, objSdk.WithEndpoint(endpoint))
	if err != nil {
		resp.Diagnostics.AddError("Failed to configure object storage", "Invalid credentials data")
		return
	}
	d.objects = a.Objects()
	d.region = dataConfig.Region
	d.endpoint = endpoint.String()
}

func (d *ObjectStorageObjectDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about an object storage object.",
		Attributes: map[string]schema.Attribute{
			"bucket": schema.StringAttribute{
				Required:    true,
				Description: "Name of the bucket.",
			},
			"key": schema.StringAttribute{
				Required:    true,
				Description: "The name of the object (key) in the bucket.",
			},
			"content_type": schema.StringAttribute{
				Computed:    true,
				Description: "Content type of the object (MIME type).",
			},
			"etag": schema.StringAttribute{
				Computed:    true,
				Description: "ETag of the object.",
			},
			"size": schema.Int64Attribute{
				Computed:    true,
				Description: "Size of the object in bytes.",
			},
			"last_modified": schema.StringAttribute{
				Computed:    true,
				Description: "Last modified date of the object.",
			},
			"lock_status": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether object lock is enabled for this object.",
			},
		},
	}
}

func (d *ObjectStorageObjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ObjectStorageObjectDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := data.Bucket.ValueString()
	objectKey := data.Key.ValueString()

	objMeta, err := d.objects.Metadata(ctx, bucketName, objectKey, nil)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "NoSuchKey") || strings.Contains(errStr, "not found") || strings.Contains(errStr, "does not exist") {
			resp.Diagnostics.AddError(
				"Object not found",
				fmt.Sprintf("Object %s does not exist in bucket %s", objectKey, bucketName),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading object metadata",
			fmt.Sprintf("Could not read metadata for object %s: %s", objectKey, err.Error()),
		)
		return
	}

	data.ETag = types.StringValue(strings.Trim(objMeta.ETag, "\""))
	data.Size = types.Int64Value(objMeta.Size)
	data.LastModified = types.StringValue(objMeta.LastModified.Format(time.RFC3339))
	data.ContentType = types.StringValue(objMeta.ContentType)

	lockInfo, err := d.objects.GetObjectLockInfo(ctx, bucketName, objectKey)
	if err != nil {
		data.LockStatus = types.BoolValue(false)
	} else {
		data.LockStatus = types.BoolValue(lockInfo.Locked)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
