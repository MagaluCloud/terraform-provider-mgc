package ssh

import (
	"context"
	"errors"
	"testing"

	sdkSSHKeys "github.com/MagaluCloud/mgc-sdk-go/sshkeys"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/mocks"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/testutils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func TestDataSourceSSH_Metadata(t *testing.T) {
	t.Parallel()
	d := NewDataSourceSSH()
	req := datasource.MetadataRequest{
		ProviderTypeName: "mgc",
	}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.Background(), req, resp)

	assert.Equal(t, "mgc_ssh_keys", resp.TypeName)
}

func TestDataSourceSSH_Schema(t *testing.T) {
	t.Parallel()
	d := NewDataSourceSSH()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.Background(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.NotEmpty(t, resp.Schema.Description)
	assert.Contains(t, resp.Schema.Attributes, "ssh_keys")
}

func TestDataSourceSSH_Read(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.KeyService)
	mockSvc.On("List", ctx, sdkSSHKeys.ListOptions{}).Return([]sdkSSHKeys.SSHKey{
		{
			ID:      "key-1",
			Name:    "my-key-1",
			KeyType: "ssh-rsa",
		},
		{
			ID:      "key-2",
			Name:    "my-key-2",
			KeyType: "ssh-ed25519",
		},
	}, nil)

	d := &DataSourceSSH{
		sshKeys: mockSvc,
	}

	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	sshKeysType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":       tftypes.String,
			"key_type": tftypes.String,
			"name":     tftypes.String,
		},
	}
	listType := tftypes.List{ElementType: sshKeysType}

	configRaw := tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"ssh_keys": listType,
			},
		},
		map[string]tftypes.Value{
			"ssh_keys": tftypes.NewValue(listType, []tftypes.Value{}),
		},
	)

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    configRaw,
	}

	req := datasource.ReadRequest{
		Config: config,
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	d.Read(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "Read operation returned diagnostics errors: %v", resp.Diagnostics)

	var finalState SshKeysModel
	resp.State.Get(ctx, &finalState)

	assert.Len(t, finalState.SSHKeys, 2)
	assert.Equal(t, "key-1", finalState.SSHKeys[0].ID.ValueString())
	assert.Equal(t, "my-key-1", finalState.SSHKeys[0].Name.ValueString())
	assert.Equal(t, "ssh-rsa", finalState.SSHKeys[0].Key_Type.ValueString())
	assert.Equal(t, "key-2", finalState.SSHKeys[1].ID.ValueString())
	assert.Equal(t, "my-key-2", finalState.SSHKeys[1].Name.ValueString())
	assert.Equal(t, "ssh-ed25519", finalState.SSHKeys[1].Key_Type.ValueString())
	mockSvc.AssertExpectations(t)
}

func TestDataSourceSSH_Read_EmptyList(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.KeyService)
	mockSvc.On("List", ctx, sdkSSHKeys.ListOptions{}).Return([]sdkSSHKeys.SSHKey{}, nil)

	d := &DataSourceSSH{
		sshKeys: mockSvc,
	}

	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	sshKeysType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":       tftypes.String,
			"key_type": tftypes.String,
			"name":     tftypes.String,
		},
	}
	listType := tftypes.List{ElementType: sshKeysType}

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(
			tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"ssh_keys": listType,
				},
			},
			map[string]tftypes.Value{
				"ssh_keys": tftypes.NewValue(listType, []tftypes.Value{}),
			},
		),
	}

	req := datasource.ReadRequest{
		Config: config,
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	d.Read(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError())

	var finalState SshKeysModel
	resp.State.Get(ctx, &finalState)

	assert.Len(t, finalState.SSHKeys, 0)
	mockSvc.AssertExpectations(t)
}

func TestDataSourceSSH_Read_APIError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.KeyService)
	mockSvc.On("List", ctx, sdkSSHKeys.ListOptions{}).Return([]sdkSSHKeys.SSHKey(nil), errors.New("api connection error"))

	d := &DataSourceSSH{
		sshKeys: mockSvc,
	}

	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	sshKeysType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":       tftypes.String,
			"key_type": tftypes.String,
			"name":     tftypes.String,
		},
	}
	listType := tftypes.List{ElementType: sshKeysType}

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(
			tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"ssh_keys": listType,
				},
			},
			map[string]tftypes.Value{
				"ssh_keys": tftypes.NewValue(listType, []tftypes.Value{}),
			},
		),
	}

	req := datasource.ReadRequest{
		Config: config,
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	d.Read(ctx, req, resp)

	assert.True(t, resp.Diagnostics.HasError(), "Read operation should return diagnostics errors on API failure")
	mockSvc.AssertExpectations(t)
}
