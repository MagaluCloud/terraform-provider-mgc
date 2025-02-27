package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	sshSDK "github.com/MagaluCloud/mgc-sdk-go/sshkeys"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceSSH{}

type DataSourceSSH struct {
	sshKeys sshSDK.KeyService
}

type SshKeyModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Key_Type types.String `tfsdk:"key_type"`
}

type SshKeysModel struct {
	SSHKeys []SshKeyModel `tfsdk:"ssh_keys"`
}

func NewDataSourceSSH() datasource.DataSource {
	return &DataSourceSSH{}
}

func (r *DataSourceSSH) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_keys"
}

func (r *DataSourceSSH) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)

	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	r.sshKeys = sshSDK.New(&dataConfig.CoreConfig).Keys()

}

func (r *DataSourceSSH) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"ssh_keys": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of ssh-keys.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "ID of ssh key.",
						},
						"key_type": schema.StringAttribute{
							Computed:    true,
							Description: "The type of ssh key.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of ssh key",
						},
					},
				},
			},
		},
	}
	resp.Schema.Description = "Get the available ssh keys."
}

func (r *DataSourceSSH) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SshKeysModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	sdkOutput, err := r.sshKeys.List(ctx, sshSDK.ListOptions{ /*TODO: Add options*/ })
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	for _, key := range sdkOutput {
		data.SSHKeys = append(data.SSHKeys, SshKeyModel{
			ID:       types.StringValue(key.ID),
			Name:     types.StringValue(key.Name),
			Key_Type: types.StringValue(key.KeyType),
		})

	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
