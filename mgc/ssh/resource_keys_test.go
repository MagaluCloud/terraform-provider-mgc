package ssh

import (
	"context"
	"errors"
	"testing"

	sdkSSHKeys "github.com/MagaluCloud/mgc-sdk-go/sshkeys"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/mocks"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/testutils"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestSshKeysResource_Metadata(t *testing.T) {
	r := NewSshKeysResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "mgc",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	assert.Equal(t, "mgc_ssh_keys", resp.TypeName)
}

func TestSshKeysResource_Schema(t *testing.T) {
	r := NewSshKeysResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.NotEmpty(t, resp.Schema.Description)
	assert.Contains(t, resp.Schema.Attributes, "name")
	assert.Contains(t, resp.Schema.Attributes, "key")
	assert.Contains(t, resp.Schema.Attributes, "id")
}

func TestSshKeysResource_Create(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.KeyService)
	mockSvc.On("Create", ctx, sdkSSHKeys.CreateSSHKeyRequest{
		Name: "my-test-key",
		Key:  "ssh-rsa AAAAB3Nza...",
	}).Return(&sdkSSHKeys.SSHKey{
		ID:   "12345",
		Name: "my-test-key",
		Key:  "ssh-rsa AAAAB3Nza...",
	}, nil)

	r := &sshKeys{
		sshKeys: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	inputData := sshKeyModel{
		ID:   types.StringUnknown(),
		Name: types.StringValue("my-test-key"),
		Key:  types.StringValue("ssh-rsa AAAAB3Nza..."),
	}
	diags := plan.Set(ctx, &inputData)
	assert.False(t, diags.HasError())

	req := resource.CreateRequest{
		Plan: plan,
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "Create operation returned diagnostics errors")

	var finalState sshKeyModel
	resp.State.Get(ctx, &finalState)

	assert.Equal(t, "12345", finalState.ID.ValueString())
	assert.Equal(t, "my-test-key", finalState.Name.ValueString())
	mockSvc.AssertExpectations(t)
}

func TestSshKeysResource_Read(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.KeyService)
	mockSvc.On("Get", ctx, "12345").Return(&sdkSSHKeys.SSHKey{
		ID:   "12345",
		Name: "my-read-key",
		Key:  "ssh-rsa AAAAB3Nza...",
	}, nil)

	r := &sshKeys{
		sshKeys: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	state := tfsdk.State{Schema: schemaResp.Schema}
	inputData := sshKeyModel{
		ID:   types.StringValue("12345"),
		Name: types.StringValue("my-read-key"),
		Key:  types.StringValue("ssh-rsa AAAAB3Nza..."),
	}
	diags := state.Set(ctx, &inputData)
	assert.False(t, diags.HasError())

	req := resource.ReadRequest{
		State: state,
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "Read operation returned diagnostics errors")

	var finalState sshKeyModel
	resp.State.Get(ctx, &finalState)

	assert.Equal(t, "12345", finalState.ID.ValueString())
	assert.Equal(t, "my-read-key", finalState.Name.ValueString())
	mockSvc.AssertExpectations(t)
}

func TestSshKeysResource_Delete(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.KeyService)
	mockSvc.On("Delete", ctx, "12345").Return((*sdkSSHKeys.SSHKey)(nil), nil)

	r := &sshKeys{
		sshKeys: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	state := tfsdk.State{Schema: schemaResp.Schema}
	inputData := sshKeyModel{
		ID:   types.StringValue("12345"),
		Name: types.StringValue("my-del-key"),
		Key:  types.StringValue("ssh-rsa AAAAB3Nza..."),
	}
	diags := state.Set(ctx, &inputData)
	assert.False(t, diags.HasError())

	req := resource.DeleteRequest{
		State: state,
	}

	resp := &resource.DeleteResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Delete(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "Delete operation returned diagnostics errors")
	mockSvc.AssertExpectations(t)
}

func TestSshKeysResource_Create_APIError(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.KeyService)
	mockSvc.On("Create", ctx, sdkSSHKeys.CreateSSHKeyRequest{
		Name: "my-error-key",
		Key:  "ssh-rsa AAAAB3Nza...",
	}).Return((*sdkSSHKeys.SSHKey)(nil), errors.New("internal server error"))

	r := &sshKeys{
		sshKeys: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	inputData := sshKeyModel{
		ID:   types.StringUnknown(),
		Name: types.StringValue("my-error-key"),
		Key:  types.StringValue("ssh-rsa AAAAB3Nza..."),
	}
	_ = plan.Set(ctx, &inputData)

	req := resource.CreateRequest{
		Plan: plan,
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(ctx, req, resp)

	assert.True(t, resp.Diagnostics.HasError(), "Create operation should return diagnostics errors on API failure")
	mockSvc.AssertExpectations(t)
}

func TestSshKeysResource_Read_APIError(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.KeyService)
	mockSvc.On("Get", ctx, "12345").Return((*sdkSSHKeys.SSHKey)(nil), errors.New("not found"))

	r := &sshKeys{
		sshKeys: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	state := tfsdk.State{Schema: schemaResp.Schema}
	inputData := sshKeyModel{
		ID:   types.StringValue("12345"),
		Name: types.StringValue("my-read-key"),
		Key:  types.StringValue("ssh-rsa AAAAB3Nza..."),
	}
	_ = state.Set(ctx, &inputData)

	req := resource.ReadRequest{
		State: state,
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(ctx, req, resp)

	assert.True(t, resp.Diagnostics.HasError(), "Read operation should return diagnostics errors on API failure")
	mockSvc.AssertExpectations(t)
}

func TestSshKeysResource_Update_NotSupported(t *testing.T) {
	ctx := context.Background()
	r := &sshKeys{}

	req := resource.UpdateRequest{}
	resp := &resource.UpdateResponse{}

	r.Update(ctx, req, resp)

	assert.True(t, resp.Diagnostics.HasError(), "Update operation should return error since it is not supported")
}

func TestSshKeysResource_ImportState(t *testing.T) {
	ctx := context.Background()
	r := &sshKeys{}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	state := tfsdk.State{Schema: schemaResp.Schema}
	_ = state.Set(ctx, &sshKeyModel{
		ID:   types.StringUnknown(),
		Name: types.StringUnknown(),
		Key:  types.StringUnknown(),
	})

	req := resource.ImportStateRequest{
		ID: "import-id-123",
	}
	resp := &resource.ImportStateResponse{
		State: state,
	}

	r.ImportState(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "ImportState should not return errors: %v", resp.Diagnostics)

	var finalState sshKeyModel
	resp.State.Get(ctx, &finalState)
	assert.Equal(t, "import-id-123", finalState.ID.ValueString())
}

func TestSshKeysResource_Delete_APIError(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.KeyService)
	mockSvc.On("Delete", ctx, "12345").Return((*sdkSSHKeys.SSHKey)(nil), errors.New("delete error"))

	r := &sshKeys{
		sshKeys: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	state := tfsdk.State{Schema: schemaResp.Schema}
	inputData := sshKeyModel{
		ID:   types.StringValue("12345"),
		Name: types.StringValue("my-del-key"),
		Key:  types.StringValue("ssh-rsa AAAAB3Nza..."),
	}
	_ = state.Set(ctx, &inputData)

	req := resource.DeleteRequest{
		State: state,
	}

	resp := &resource.DeleteResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Delete(ctx, req, resp)

	assert.True(t, resp.Diagnostics.HasError(), "Delete operation should return diagnostics errors on API failure")
	mockSvc.AssertExpectations(t)
}
