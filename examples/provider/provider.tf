terraform {
  required_providers {
    nsxt_intervlan_routing = {
      source = "technofish-au/nsxt-intervlan-routing"
    }
  }
}

provider "nsxt-intervlan-routing" {
  nsxt_host     = "127.0.0.1"
  nsxt_insecure = true
  nsxt_username = "admin"
  nsxt_password = "password"
}

data "nsxt_intervlan_routing_segment_ports" "example" {
  segment_id    = ""
  segment_ports = {}
}

resource "nsxt_intervlan_routing_segment_port" "parent_example" {
  segment_id = "4d4c0f0a-6c5 0-420b-90f1-68fb7585cda4"
  port_id    = "a274ac51-88f5-491f-a46f-840d409ce82f"
  segment_port = {
    admin_state = "UP"
    attachment = {
      id          = "9765bf41-9725-4714-977e-7f7395920de2"
      traffic_tag = "1000"
      type        = "PARENT"
    }
    description   = "GCVE-PA-VM-ESX-2 Parent Port"
    display_name  = "GCVE-PA-VM-ESX-2.vmx@060af2c2-e9ff-4686-866c-c0daab1748d6"
    id            = "060af2c2-e9ff-4686-866c-c0daab1748d6"
    resource_type = "SegmentPort"
  }
}

resource "nsxt_intervlan_routing_segment_port" "child_example" {
  segment_id = "2bfe8abf-4161-4788-9cbe-c444e9bf7454"
  port_id    = "a274ac51-88f5-491f-a46f-840d409ce82f"
  segment_port = {
    address_bindings = [
      {
        ip_address  = "169.254.254.169"
        mac_address = "00:50:56:ad:5e:64"
        vlan_id     = "1001"
      },
    ]
    admin_state = "UP"
    attachment = {
      context_id  = "9765bf41-9725-4714-977e-7f7395920de2"
      traffic_tag = "1001"
      app_id      = "Segment1001"
      type        = "CHILD"
    }
    description   = "GCVE-PA-VM-ESX-2 Child Port 1001"
    display_name  = "GCVE-PA-VM-ESX-2.vmx@a274ac51-88f5-491f-a46f-840d409ce82f"
    id            = "a274ac51-88f5-491f-a46f-840d409ce82f"
    resource_type = "SegmentPort"
  }
}
