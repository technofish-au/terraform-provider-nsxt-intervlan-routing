// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: GPL-2.0-or-later

package provider

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/technofish-au/terraform-provider-nsxt-intervlan-routing/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource              = &segmentPortResource{}
	_ resource.ResourceWithConfigure = &segmentPortResource{}
)

func NewSegmentPortResource() resource.Resource {
	return &segmentPortResource{}
}

type segmentPortResource struct {
	client *client.Client
}

type segmentPortResourceModel struct {
	SegmentId   types.String       `tfsdk:"segment_id"`
	PortId      types.String       `tfsdk:"port_id"`
	SegmentPort client.SegmentPort `tfsdk:"segment_port"`
}

func (r *segmentPortResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		tflog.Error(ctx, "Unable to prepare client")
		return
	}
	r.client = client
}

// Metadata returns the resource type name.
func (r *segmentPortResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_segment_port"
}

func (r *segmentPortResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage a segment port.",
		Attributes: map[string]schema.Attribute{
			"segment_id": schema.StringAttribute{
				Description: "Identifier for this segment.",
				Required:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"port_id": schema.StringAttribute{
				Description: "Identifier for this port.",
				Required:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"segment_port": schema.SetNestedAttribute{
				Description: "The segment port definition",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"address_bindings": schema.SetNestedAttribute{
							Description: "List of IP address bindings",
							Optional:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"ip_address": schema.StringAttribute{
										Description: "IP address of segment port",
									},
									"mac_address": schema.StringAttribute{
										Description: "MAC address of segment port",
									},
									"vlan_id": schema.StringAttribute{
										Description: "VLAN ID associated with this segment port",
									},
								},
							},
						},
						"admin_state": schema.StringAttribute{
							Description: "Admin state of the segment port",
						},
						"attachment": schema.SetNestedAttribute{
							Description: "List of attachments",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Description: "Port attachment ID. VIF UUID in NSX.",
									},
									"context_id": schema.StringAttribute{
										Description: "Attachment UUID of the PARENT port. Only required when type is CHILD.",
										Computed:    true,
									},
									"traffic_tag": schema.StringAttribute{
										Description: "Traffic tag associated with this port. Only required when type is CHILD.",
									},
									"app_id": schema.StringAttribute{
										Description: "Application ID associated with this port. Can be the same as the display name. Only required when type is CHILD.",
									},
									"type": schema.StringAttribute{
										Description: "Type of attachment. Case sensitive. Can be either PARENT or CHILD.",
									},
								},
							},
						},
						"description": schema.StringAttribute{
							Description: "Description of segment port",
						},
						"display_name": schema.StringAttribute{
							Description: "Display name of segment port",
						},
						"id": schema.StringAttribute{
							Description: "Id of segment port",
						},
						"resource_type": schema.StringAttribute{
							Description: "Resource type of segment port. Can only be set to 'SegmentPort'",
						},
					},
				},
			},
		},
	}
}

// Create a new resource.
func (r *segmentPortResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Preparing to create segment port resource")
	// Retrieve values from plan
	var plan segmentPortResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	segment_id := plan.SegmentId.ValueString()
	port_id := plan.PortId.ValueString()
	segment_port := plan.SegmentPort

	patchRequest := client.PatchSegmentPortRequest{
		SegmentId:   segment_id,
		PortId:      port_id,
		SegmentPort: segment_port,
	}

	// Create new item
	spResponse, err := r.client.PatchSegmentPort(ctx, patchRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Segment Port",
			err.Error(),
		)
		return
	}

	if spResponse.StatusCode != 200 {
		resp.Diagnostics.AddError(
			"An invalid response was received. Code: "+string(spResponse.StatusCode),
			spResponse.Status,
		)
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Created segment port resource", map[string]any{"success": true})
}

// Read resource information.
func (r *segmentPortResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read item resource")
	// Get current state
	var state segmentPortResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	spResponse, err := r.client.GetSegmentPort(ctx, state.SegmentId.ValueString(), state.PortId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Segment Port configuration",
			err.Error(),
		)
		return
	}

	// Treat HTTP 404 Not Found status as a signal to remove/recreate resource
	if spResponse.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if spResponse.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected HTTP error code received for segment port",
			spResponse.Status,
		)
		return
	}

	var newSegmentPort client.SegmentPort
	if err := json.NewDecoder(spResponse.Body).Decode(&newSegmentPort); err != nil {
		resp.Diagnostics.AddError(
			"Invalid format received for Item",
			err.Error(),
		)
		return
	}

	// Map response body to model
	state = segmentPortResourceModel{
		SegmentId:   state.SegmentId,
		PortId:      state.PortId,
		SegmentPort: newSegmentPort,
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Finished reading segment port resource", map[string]any{"success": true})
}

func (r *segmentPortResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Preparing to update segment port resource")
	// Retrieve values from plan
	var plan segmentPortResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	segment_id := plan.SegmentId.ValueString()
	port_id := plan.PortId.ValueString()
	segment_port := plan.SegmentPort

	patchRequest := client.PatchSegmentPortRequest{
		SegmentId:   segment_id,
		PortId:      port_id,
		SegmentPort: segment_port,
	}

	// Create new item
	spResponse, err := r.client.PatchSegmentPort(ctx, patchRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Segment Port",
			err.Error(),
		)
		return
	}

	if spResponse.StatusCode != 200 {
		resp.Diagnostics.AddError(
			"An invalid response was received. Code: "+string(spResponse.StatusCode),
			spResponse.Status,
		)
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Updated segment port resource", map[string]any{"success": true})
}

func (r *segmentPortResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Preparing to delete segment port resource")
	// Retrieve values from state
	var state segmentPortResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// delete item
	_, err := r.client.DeleteSegmentPort(ctx, state.SegmentId.ValueString(), state.PortId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Item",
			err.Error(),
		)
		return
	}
	tflog.Debug(ctx, "Deleted segment port resource", map[string]any{"success": true})
}

func (r *segmentPortResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	// If our ID was a string then we could do this
	resource.ImportStatePassthroughID(ctx, path.Root("port_id"), req, resp)

	//id, err := strconv.ParseInt(req.ID, 10, 64)
	//
	//if err != nil {
	//	resp.Diagnostics.AddError(
	//		"Error importing item",
	//		"Could not import item, unexpected error (ID should be an integer): "+err.Error(),
	//	)
	//	return
	//}

	//resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
