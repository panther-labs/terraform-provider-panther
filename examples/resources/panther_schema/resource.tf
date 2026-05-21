# Basic custom log schema with field discovery
resource "panther_schema" "web_access_logs" {
  name                       = "Custom.WebAccessLogs"
  description                = "Schema for web server access logs"
  spec                       = <<EOF
fields:
  - name: timestamp
    type: timestamp
    timeFormat: unix
    isEventTime: true
  - name: client_ip
    type: string
  - name: method
    type: string
  - name: path
    type: string
  - name: status_code
    type: int
  - name: response_time
    type: float
EOF
  is_field_discovery_enabled = true
}

# Advanced schema with nested objects and arrays
resource "panther_schema" "application_logs" {
  name                       = "Custom.ApplicationLogs"
  description                = "Schema for application logs with structured data"
  spec                       = <<EOF
fields:
  - name: timestamp
    type: timestamp
    timeFormat: rfc3339
    isEventTime: true
  - name: level
    type: string
  - name: message
    type: string
  - name: service
    type: string
  - name: user
    type: object
    fields:
      - name: id
        type: string
      - name: email
        type: string
      - name: role
        type: string
  - name: request
    type: object
    fields:
      - name: method
        type: string
      - name: url
        type: string
      - name: headers
        type: object
  - name: tags
    type: array
    element:
      type: string
EOF
  is_field_discovery_enabled = false
}