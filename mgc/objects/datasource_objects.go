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

var _ datasource.DataSource = &ObjectStorageObjectsDataSource{}

type ObjectStorageObjectsDataSource struct {
	objects objSdk.ObjectService
}

type ObjectStorageObjectsDataSourceItemModel struct {
	Key          types.String `tfsdk:"key"`
	Size         types.Int64  `tfsdk:"size"`
	ETag         types.String `tfsdk:"etag"`
	LastModified types.String `tfsdk:"last_modified"`
}

type ObjectStorageObjectsDataSourceModel struct {
	Bucket  types.String                              `tfsdk:"bucket"`
	Objects []ObjectStorageObjectsDataSourceItemModel `tfsdk:"objects"`
}

func NewObjectStorageObjectsDataSource() datasource.DataSource {
	return &ObjectStorageObjectsDataSource{}
}

func (d *ObjectStorageObjectsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object_storage_objects"
}

func (d *ObjectStorageObjectsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.objects = a.Objects()
}

func (d *ObjectStorageObjectsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a list of objects in a bucket.",
		Attributes: map[string]schema.Attribute{
			"bucket": schema.StringAttribute{
				Required:    true,
				Description: "Name of the bucket.",
			},
			"objects": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of objects in the bucket.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the object (key).",
						},
						"size": schema.Int64Attribute{
							Computed:    true,
							Description: "Size of the object in bytes.",
						},
						"etag": schema.StringAttribute{
							Computed:    true,
							Description: "ETag of the object.",
						},
						"last_modified": schema.StringAttribute{
							Computed:    true,
							Description: "Last modified date of the object.",
						},
					},
				},
			},
		},
	}
}

func (d *ObjectStorageObjectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ObjectStorageObjectsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := data.Bucket.ValueString()

	objects, err := d.objects.List(ctx, bucketName, objSdk.ObjectListOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing objects",
			fmt.Sprintf("Could not list objects in bucket %s: %s", bucketName, err.Error()),
		)
		return
	}

	for _, obj := range objects {
		data.Objects = append(data.Objects, ObjectStorageObjectsDataSourceItemModel{
			Key:          types.StringValue(obj.Key),
			Size:         types.Int64Value(obj.Size),
			ETag:         types.StringValue(strings.Trim(obj.ETag, "\"")),
			LastModified: types.StringValue(obj.LastModified.Format(time.RFC3339)),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
