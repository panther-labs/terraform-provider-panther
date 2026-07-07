resource "panther_logforwardersource" "example" {
  integration_label = var.integration_label
  log_stream_type   = var.log_stream_type
  log_types         = var.log_types
  log_stream_type_options = {
    json_array_envelope_field = var.json_array_envelope_field
    xml_root_element          = var.xml_root_element
  }
}
