package network

import (
	"context"
	"strings"
	"time"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const NetworkPoolingTimeout = 5 * time.Minute

type NetworkVPCModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

type NetworkVPCResource struct {
	networkVPC netSDK.VPCService
}

func NewNetworkVPCResource() resource.Resource {
	return &NetworkVPCResource{}
}

func (r *NetworkVPCResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_vpcs"
}

func (r *NetworkVPCResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkVPC = netSDK.New(&dataConfig.CoreConfig).VPCs()
}

func (r *NetworkVPCResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network VPC",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the VPC",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the VPC",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the VPC",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *NetworkVPCResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NetworkVPCModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createdVPC, err := r.networkVPC.Create(ctx, netSDK.CreateVPCRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	for startTime := time.Now(); time.Since(startTime) < NetworkPoolingTimeout; {
		res, err := r.networkVPC.Get(ctx, createdVPC)
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
		if res.Status == "created" {
			break
		}
		if strings.Contains(res.Status, "error") {
			resp.Diagnostics.AddError(
				"Error in VPC creation",
				"VPC creation failed with status: ["+res.Status+"] \nVPC ID: "+createdVPC+" \nPlease check the VPC status in the Magalu Cloud CLI or contact support")
			return
		}
		tflog.Info(ctx, "VPC is not yet created, waiting for 10 seconds",
			map[string]interface{}{"status": res.Status})
		time.Sleep(10 * time.Second)
	}

	data.Id = types.StringValue(createdVPC)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NetworkVPCResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NetworkVPCModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpc, err := r.networkVPC.Get(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	if vpc.Description != nil && *vpc.Description == "" {
		vpc.Description = nil
	}

	data.Name = types.StringPointerValue(vpc.Name)
	data.Description = types.StringPointerValue(vpc.Description)
	data.Id = types.StringPointerValue(vpc.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NetworkVPCResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkVPCModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.networkVPC.Delete(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
}

func (r *NetworkVPCResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update is not supported for VPC", "")
}

func (r *NetworkVPCResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
