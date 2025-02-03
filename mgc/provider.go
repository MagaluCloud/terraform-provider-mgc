package mgc

import (
	"context"
	"os"
	"time"

	datasources "github.com/MagaluCloud/terraform-provider-mgc/mgc/datasources"
	resources "github.com/MagaluCloud/terraform-provider-mgc/mgc/resources"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	mgcSdk "github.com/MagaluCloud/magalu/mgc/sdk"
	"github.com/MagaluCloud/mgc-sdk-go/client"
)

const providerTypeName = "mgc"

type mgcProvider struct {
	version string
	sdk     *mgcSdk.Sdk
	mgcSdk  *client.CoreClient
}

func (p *mgcProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = providerTypeName
	resp.Version = p.version
}

func (p *mgcProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform Provider for Magalu Cloud",
		Attributes: map[string]schema.Attribute{
			"region": schema.StringAttribute{
				Description: "The region to use for resources. Options: br-ne1 / br-se1. Default is br-se1.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("br-ne1", "br-se1", "br-mgl1"),
				},
			},
			"env": schema.StringAttribute{
				Description: "The environment to use. Options: prod / pre-prod. Default is prod.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("prod", "pre-prod"),
				},
			},
			"api_key": schema.StringAttribute{
				Description: "The Magalu API Key for authentication.",
				Optional:    true,
			},
			"object_storage": schema.SingleNestedAttribute{
				Description: "Configuration settings for Object Storage",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"key_pair": schema.SingleNestedAttribute{
						Description: "Bucket Key Pair configuration",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"key_id": schema.StringAttribute{
								Description: "The API Key ID.",
								Required:    true,
							},
							"key_secret": schema.StringAttribute{
								Description: "The API Key Secret.",
								Required:    true,
							},
						},
					},
				},
			},
		},
	}
}

func (p *mgcProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data tfutil.ProviderConfig
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "fail to get configs from provider")
	}

	if data.ApiKey.ValueString() == "" {
		if apiKeyFromOS := os.Getenv("MGC_API_KEY"); apiKeyFromOS != "" {
			resp.Diagnostics.AddWarning("The ´MGC_API_KEY´ environment variable is deprecated. Please use the ´api_key´ provider configuration instead.", "The environment variable ´MGC_API_KEY´ is deprecated. Please use the ´api_key´ provider configuration instead. This environment variable will be removed in a future release.")
			data.ApiKey = types.StringValue(apiKeyFromOS)
		} else {
			data.ApiKey = types.StringValue("")
		}
	}

	if data.Env.ValueString() == "" {
		if envFromOS := os.Getenv("MGC_ENV"); envFromOS != "" {
			resp.Diagnostics.AddWarning("The ´MGC_ENV´ environment variable is deprecated. Please use the ´env´ provider configuration instead.", "The environment variable ´MGC_ENV´ is deprecated. Please use the ´env´ provider configuration instead. This environment variable will be removed in a future release.")
			data.Env = types.StringValue(envFromOS)
		} else {
			data.Env = types.StringValue("prod")
		}
	}

	if data.Region.ValueString() == "" {
		if regionFromOS := os.Getenv("MGC_REGION"); regionFromOS != "" {
			resp.Diagnostics.AddWarning("The ´MGC_REGION´ environment variable is deprecated. Please use the ´region´ provider configuration instead.", "The environment variable ´MGC_REGION´ is deprecated. Please use the ´region´ provider configuration instead. This environment variable will be removed in a future release.")
			data.Region = types.StringValue(regionFromOS)
		} else {
			data.Region = types.StringValue("br-se1")
		}
	}

	if data.ObjectStorage == nil || (os.Getenv("MGC_OBJ_KEY_ID") != "" && os.Getenv("MGC_OBJ_KEY_SECRET") != "") {
		resp.Diagnostics.AddWarning("The ´MGC_OBJ_KEY_ID´ and ´MGC_OBJ_KEY_SECRET´ environment variables are deprecated. Please use the ´object_storage´ provider configuration instead.", "The environment variables ´MGC_OBJ_KEY_ID´ and ´MGC_OBJ_KEY_SECRET´ are deprecated. Please use the ´object_storage´ provider configuration instead. These environment variables will be removed in a future release.")
		data.ObjectStorage = &tfutil.ObjectStorageConfig{
			ObjectKeyPair: &tfutil.KeyPair{
				KeyID:     types.StringValue(os.Getenv("MGC_OBJ_KEY_ID")),
				KeySecret: types.StringValue(os.Getenv("MGC_OBJ_KEY_SECRET")),
			},
		}
	}

	// Remove this comment - New SDK
	if data.ApiKey.ValueString() == "" {
		resp.Diagnostics.AddError("ApiKey is required", "ApiKey is required")
	}
	regionUrl, err := tfutil.RegionToUrl(data.Region.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid region", err.Error())
	}
	mgcSdkClient := client.NewMgcClient(
		data.ApiKey.ValueString(),
		client.WithBaseURL(client.MgcUrl(regionUrl)),
		client.WithRetryConfig(10, 5*time.Second, 60*time.Second, 2.0),
	)
	data.Sdk = mgcSdkClient

	resp.DataSourceData = data
	resp.ResourceData = data
}

func (p *mgcProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewNewNodePoolResource,
		resources.NewK8sClusterResource,
		resources.NewObjectStorageBucketsResource,
		resources.NewVirtualMachineInstancesResource,
		resources.NewVirtualMachineSnapshotsResource,
		resources.NewVolumeAttachResource,
		resources.NewBlockStorageSnapshotsResource,
		resources.NewBlockStorageVolumesResource,
		resources.NewSshKeysResource,
		resources.NewNetworkPublicIPResource,
		resources.NewNetworkVPCResource,
		resources.NewNetworkSecurityGroupsResource,
		resources.NewNetworkSecurityGroupsRulesResource,
		resources.NewNetworkVPCInterfaceResource,
		resources.NewNetworkSecurityGroupsAttachResource,
		resources.NewNetworkPublicIPAttachResource,
		resources.NewNetworkVpcsSubnetsResource,
		resources.NewNetworkSubnetpoolsResource,
		resources.NewDBaaSInstanceResource,
		resources.NewDBaaSInstanceSnapshotResource,
		resources.NewContainerRegistryRegistriesResource,
		resources.NewVirtualMachineInterfaceAttachResource,
	}
}

func (p *mgcProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewDataSourceKubernetesClusterKubeConfig,
		datasources.NewDataSourceKubernetesCluster,
		datasources.NewDataSourceKubernetesFlavor,
		datasources.NewDataSourceKubernetesVersion,
		datasources.NewDataSourceKubernetesNodepool,
		datasources.NewDataSourceKubernetesNode,
		datasources.NewDataSourceVmMachineType,
		datasources.NewDataSourceVMIMages,
		datasources.NewDataSourceVmInstance,
		datasources.NewDataSourceVmInstances,
		datasources.NewDataSourceNetworkVPC,
		datasources.NewDataSourceNetworkSecurityGroup,
		datasources.NewDataSourceNetworkVPCInterface,
		datasources.NewDataSourceNetworkVpcsSubnet,
		datasources.NewDataSourceNetworkSubnetpool,
		datasources.NewDataSourceNetworkPublicIP,
		datasources.NewDataSourceBSSnapshots,
		datasources.NewDataSourceBsVolumes,
		datasources.NewDataSourceBSSnapshot,
		datasources.NewDataSourceBsVolume,
		datasources.NewDataSourceBsVolumeTypes,
		datasources.NewDataSourceDbaasEngines,
		datasources.NewDataSourceDbaasInstanceTypes,
		datasources.NewDataSourceDbaasInstances,
		datasources.NewDataSourceDbaasInstance,
		datasources.NewDataSourceDbaasInstancesSnapshots,
		datasources.NewDataSourceDbaasInstancesSnapshot,
		datasources.NewDataSourceAvailabilityZones,
		datasources.NewDataSourceCRImages,
		datasources.NewDataSourceCRRepositories,
		datasources.NewDataSourceCRRegistries,
		datasources.NewDataSourceCRCredentials,
	}
}

func New(version string) func() provider.Provider {
	//deprecated
	sdk := mgcSdk.NewSdk()
	mgcSdk.SetUserAgent("MgcTF/" + version)

	return func() provider.Provider {
		return &mgcProvider{
			sdk:     sdk,
			version: version,
			mgcSdk:  nil,
		}
	}
}
