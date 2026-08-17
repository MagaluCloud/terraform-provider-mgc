package objects

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	objSdk "github.com/MagaluCloud/mgc-sdk-go/objectstorage"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

type ObjectStorageObject struct {
	Bucket                    types.String `tfsdk:"bucket"`
	Key                       types.String `tfsdk:"key"`
	ContentType               types.String `tfsdk:"content_type"`
	Source                    types.String `tfsdk:"source"`
	Content                   types.String `tfsdk:"content"`
	ETag                      types.String `tfsdk:"etag"`
	Size                      types.Int64  `tfsdk:"size"`
	LastModified              types.String `tfsdk:"last_modified"`
	ObjectLockRetainUntilDate types.String `tfsdk:"object_lock_retain_until_date"`
}

func NewObjectStorageObjectsResource() resource.Resource {
	return &objectStorageObjects{}
}

type objectStorageObjects struct {
	objects  objSdk.ObjectService
	region   string
	endpoint string
}

func (r *objectStorageObjects) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object_storage_objects"
}

func (r *objectStorageObjects) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.objects = a.Objects()
	r.region = dataConfig.Region
	r.endpoint = endpoint.String()
}

func (r *objectStorageObjects) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An object storage object.",
		Attributes: map[string]schema.Attribute{
			"bucket": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Name of the bucket to store the object.",
			},
			"key": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "The name of the object (key) in the bucket.",
			},
			"content_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Content type of the object (MIME type).",
			},
			"source": schema.StringAttribute{
				Optional:    true,
				Description: "Path to a file to upload as the object content.",
			},
			"content": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Literal content to upload as the object.",
			},
			"etag": schema.StringAttribute{
				Computed:    true,
				Description: "ETag of the uploaded object.",
			},
			"size": schema.Int64Attribute{
				Computed:    true,
				Description: "Size of the object in bytes.",
			},
			"last_modified": schema.StringAttribute{
				Computed:    true,
				Description: "Last modified date of the object.",
			},
			"object_lock_retain_until_date": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "The retain-until-date for object lock in RFC3339 format (e.g., 2025-12-31T23:59:59Z). " +
					"Object locks created through Terraform and MGC CLI are applied in COMPLIANCE mode. " +
					"Therefore, once configured, they cannot be removed or reverted.",
			},
		},
	}
}

func (r *objectStorageObjects) getContent(plan *ObjectStorageObject) ([]byte, string, error) {
	srcSet := !plan.Source.IsNull() && plan.Source.ValueString() != ""
	ctSet := !plan.Content.IsNull() && plan.Content.ValueString() != ""

	if srcSet && ctSet {
		return nil, "", fmt.Errorf("source and content cannot be set simultaneously")
	}

	if srcSet {
		data, err := os.ReadFile(plan.Source.ValueString())
		if err != nil {
			return nil, "", fmt.Errorf("could not read source file %s: %w", plan.Source.ValueString(), err)
		}
		ct := plan.ContentType.ValueString()
		if ct == "" {
			ct = mime.TypeByExtension(filepath.Ext(plan.Source.ValueString()))
		}
		return data, ct, nil
	}

	if ctSet {
		ct := plan.ContentType.ValueString()
		if ct == "" {
			ct = "text/plain; charset=utf-8"
		}
		return []byte(plan.Content.ValueString()), ct, nil
	}

	return nil, "", fmt.Errorf("either source or content must be specified")
}

func (r *objectStorageObjects) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ObjectStorageObject
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data, contentType, err := r.getContent(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Error preparing object content", err.Error())
		return
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if err := r.objects.UploadStream(ctx, plan.Bucket.ValueString(), plan.Key.ValueString(), bytes.NewReader(data), int64(len(data)), contentType); err != nil {
		resp.Diagnostics.AddError("Error uploading object",
			fmt.Sprintf("Could not upload object %s to bucket %s: %s", plan.Key.ValueString(), plan.Bucket.ValueString(), err.Error()))
		return
	}

	if !plan.ObjectLockRetainUntilDate.IsNull() && plan.ObjectLockRetainUntilDate.ValueString() != "" {
		retainUntil, err := time.Parse(time.RFC3339, plan.ObjectLockRetainUntilDate.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid object_lock_retain_until_date",
				fmt.Sprintf("Could not parse retain until date: %s", err.Error()))
			return
		}
		if err := r.objects.LockObject(ctx, plan.Bucket.ValueString(), plan.Key.ValueString(), retainUntil); err != nil {
			resp.Diagnostics.AddError("Error locking object",
				fmt.Sprintf("Could not lock object %s: %s", plan.Key.ValueString(), err.Error()))
			return
		}
	}

	objMeta, err := r.objects.Metadata(ctx, plan.Bucket.ValueString(), plan.Key.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Error reading object metadata",
			fmt.Sprintf("Could not read metadata for object %s: %s", plan.Key.ValueString(), err.Error()))
		return
	}

	plan.ETag = types.StringValue(strings.Trim(objMeta.ETag, "\""))
	plan.Size = types.Int64Value(objMeta.Size)
	plan.LastModified = types.StringValue(objMeta.LastModified.Format(time.RFC3339))
	plan.ContentType = types.StringValue(objMeta.ContentType)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *objectStorageObjects) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ObjectStorageObject
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.Bucket.ValueString()
	objectKey := state.Key.ValueString()

	objMeta, err := r.objects.Metadata(ctx, bucketName, objectKey, nil)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "NoSuchKey") || strings.Contains(errStr, "not found") || strings.Contains(errStr, "does not exist") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading object metadata",
			fmt.Sprintf("Could not read metadata for object %s: %s", objectKey, err.Error()))
		return
	}

	state.ETag = types.StringValue(strings.Trim(objMeta.ETag, "\""))
	state.Size = types.Int64Value(objMeta.Size)
	state.LastModified = types.StringValue(objMeta.LastModified.Format(time.RFC3339))
	state.ContentType = types.StringValue(objMeta.ContentType)

	lockInfo, err := r.objects.GetObjectLockInfo(ctx, bucketName, objectKey)
	if err != nil {
		state.ObjectLockRetainUntilDate = types.StringNull()
	} else if lockInfo.Locked && lockInfo.RetainUntilDate != nil {
		state.ObjectLockRetainUntilDate = types.StringValue(lockInfo.RetainUntilDate.Format(time.RFC3339))
	} else {
		state.ObjectLockRetainUntilDate = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *objectStorageObjects) contentChanged(ctx context.Context, plan, state *ObjectStorageObject) (bool, error) {
	sourceOrContentSet := (!plan.Source.IsNull() && plan.Source.ValueString() != "") ||
		(!plan.Content.IsNull() && plan.Content.ValueString() != "")
	contentTypeChanged := !plan.ContentType.Equal(state.ContentType)

	if !sourceOrContentSet && !contentTypeChanged {
		return false, nil
	}

	if !sourceOrContentSet {
		return false, fmt.Errorf("content_type changed but no source or content specified to upload")
	}

	data, contentType, err := r.getContent(plan)
	if err != nil {
		return false, err
	}

	meta, err := r.objects.Metadata(ctx, plan.Bucket.ValueString(), plan.Key.ValueString(), nil)
	if err != nil {
		return true, nil
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if contentType == meta.ContentType && int64(len(data)) == meta.Size {
		return false, nil
	}

	return true, nil
}

func (r *objectStorageObjects) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ObjectStorageObject
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ObjectStorageObject
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	changed, err := r.contentChanged(ctx, &plan, &state)
	if err != nil {
		resp.Diagnostics.AddError("Error preparing object content", err.Error())
		return
	}

	if changed {
		data, contentType, err := r.getContent(&plan)
		if err != nil {
			resp.Diagnostics.AddError("Error preparing object content", err.Error())
			return
		}
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		if err := r.objects.UploadStream(ctx, plan.Bucket.ValueString(), plan.Key.ValueString(), bytes.NewReader(data), int64(len(data)), contentType); err != nil {
			resp.Diagnostics.AddError("Error uploading object",
				fmt.Sprintf("Could not upload object %s to bucket %s: %s", plan.Key.ValueString(), plan.Bucket.ValueString(), err.Error()))
			return
		}
	}

	if !plan.ObjectLockRetainUntilDate.Equal(state.ObjectLockRetainUntilDate) {
		if plan.ObjectLockRetainUntilDate.IsNull() || plan.ObjectLockRetainUntilDate.ValueString() == "" {
			if err := r.objects.UnlockObject(ctx, plan.Bucket.ValueString(), plan.Key.ValueString()); err != nil {
				resp.Diagnostics.AddError("Error unlocking object",
					fmt.Sprintf("Could not unlock object %s: %s", plan.Key.ValueString(), err.Error()))
				return
			}
		} else {
			retainUntil, err := time.Parse(time.RFC3339, plan.ObjectLockRetainUntilDate.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Invalid object_lock_retain_until_date",
					fmt.Sprintf("Could not parse retain until date: %s", err.Error()))
				return
			}
			if err := r.objects.LockObject(ctx, plan.Bucket.ValueString(), plan.Key.ValueString(), retainUntil); err != nil {
				resp.Diagnostics.AddError("Error locking object",
					fmt.Sprintf("Could not lock object %s: %s", plan.Key.ValueString(), err.Error()))
				return
			}
		}
	}

	objMeta, err := r.objects.Metadata(ctx, plan.Bucket.ValueString(), plan.Key.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Error reading object metadata",
			fmt.Sprintf("Could not read metadata for object %s: %s", plan.Key.ValueString(), err.Error()))
		return
	}

	plan.ETag = types.StringValue(strings.Trim(objMeta.ETag, "\""))
	plan.Size = types.Int64Value(objMeta.Size)
	plan.LastModified = types.StringValue(objMeta.LastModified.Format(time.RFC3339))
	plan.ContentType = types.StringValue(objMeta.ContentType)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *objectStorageObjects) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ObjectStorageObject
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.Bucket.ValueString()
	objectKey := state.Key.ValueString()

	if !state.ObjectLockRetainUntilDate.IsNull() && state.ObjectLockRetainUntilDate.ValueString() != "" {
		if err := r.objects.UnlockObject(ctx, bucketName, objectKey); err != nil {
			resp.Diagnostics.AddError("Error unlocking object",
				fmt.Sprintf("Could not unlock object %s: %s", objectKey, err.Error()))
			return
		}
	}

	if err := r.objects.Delete(ctx, bucketName, objectKey, nil); err != nil {
		resp.Diagnostics.AddError("Error deleting object",
			fmt.Sprintf("Could not delete object %s from bucket %s: %s", objectKey, bucketName, err.Error()))
	}
}

func (r *objectStorageObjects) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID",
			"Import ID must be in the format bucket/key")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), parts[1])...)
}
