package provider

import "github.com/technofish-au/terraform-provider-nsxt-intervlan-routing/client"

type SegmentPort struct {
	AddressBindings client.PortAddressBindingEntry `tfsdk:"address_bindings"`
	AdminState      string                         `tfsdk:"admin_state"`
	Attachment      client.PortAttachment          `tfsdk:"attachment"`
	Description     string                         `tfsdk:"description"`
	DisplayName     string                         `tfsdk:"display_name"`
	Id              string                         `tfsdk:"id"`
	ResourceType    string                         `tfsdk:"resource_type"`
}
