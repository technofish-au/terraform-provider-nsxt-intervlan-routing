package provider

import (
	"context"
	"encoding/json"

	"github.com/technofish-au/terraform-provider-nsxt-intervlan-routing/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &segmentPortsDataSource{}
	_ datasource.DataSourceWithConfigure = &segmentPortsDataSource{}
)

func NewSegmentPortsDataSource() datasource.DataSource {
	return &segmentPortsDataSource{}
}

type segmentPortsDataSource struct {
	client *client.Client
}

type segmentPortsDataSourceModel struct {
	SegmentId    string        `tfsdk:"segment_id"`
	SegmentPorts []SegmentPort `tfsdk:"segment_ports"`
}

func (d segmentPortsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		tflog.Error(ctx, "Unable to prepare client")
		return
	}
	d.client = client
}

// Metadata returns the data source type name.
func (d *segmentPortsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_segment_ports"
}

// Schema defines the schema for the data source.
func (d *segmentPortsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all Segment Ports.",
		Attributes: map[string]schema.Attribute{
			"segment_id": schema.StringAttribute{
				Description: "Identifier for this segment.",
				Required:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *segmentPortsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read item data source")
	var state segmentPortsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	portsResponse, err := d.client.ListSegmentPorts(ctx, state.SegmentId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read segment ports for ",
			err.Error(),
		)
		return
	}

	var segmentPorts client.ListSegmentPortsResponse
	if portsResponse.StatusCode != 200 {
		resp.Diagnostics.AddError(
			"Unexpected HTTP error code received for Item",
			portsResponse.Status,
		)
		return
	}

	if err := json.NewDecoder(portsResponse.Body).Decode(&segmentPorts); err != nil {
		resp.Diagnostics.AddError(
			"Invalid format received for segment ports",
			err.Error(),
		)
		return
	}

	// Map response body to model
	state = segmentPortsDataSourceModel{}
	state.SegmentId = state.SegmentId
	for _, segment := range segmentPorts.Results {
		state.SegmentPorts = append(
			state.SegmentPorts,
			SegmentPort{
				AddressBindings: segment.AddressBindings,
				AdminState:      segment.AdminState,
				Attachment:      segment.Attachment,
				Description:     segment.Description,
				DisplayName:     segment.DisplayName,
				Id:              segment.Id,
			})
	}

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading segment ports data source", map[string]any{"success": true})
}
