resource "panther_pubsubsource" "example_pubsub_source" {
  integration_label = var.integration_label
  subscription_id   = var.subscription_id
  project_id        = var.project_id
  credentials       = file(var.credentials_file)
  log_types         = var.log_types
  log_stream_type   = var.log_stream_type
  log_stream_type_options = {
    json_array_envelope_field = var.json_array_envelope_field
    xml_root_element          = var.xml_root_element
  }
}
