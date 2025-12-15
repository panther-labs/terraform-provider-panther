# Web server access logs schema
resource "panther_schema" "web_access_logs" {
  name                       = "Custom.WebAccessLogs"
  description                = "Schema for web server access logs in Common Log Format"
  spec                       = <<EOF
fields:
  - name: timestamp
    type: timestamp
    timeFormat: unix
    isEventTime: true
    description: "Request timestamp"
  - name: client_ip
    type: string
    required: true
    description: "Client IP address"
  - name: method
    type: string
    required: true
    description: "HTTP method (GET, POST, etc.)"
  - name: path
    type: string
    required: true
    description: "Request path"
  - name: status_code
    type: int
    required: true
    description: "HTTP status code"
  - name: response_size
    type: int
    description: "Response size in bytes"
  - name: response_time
    type: float
    description: "Response time in milliseconds"
  - name: user_agent
    type: string
    description: "User agent string"
  - name: referer
    type: string
    description: "HTTP referer"
EOF
  is_field_discovery_enabled = true
}

# Application logs schema with structured data
resource "panther_schema" "application_logs" {
  name                       = "Custom.ApplicationLogs"
  description                = "Schema for structured application logs"
  spec                       = <<EOF
fields:
  - name: timestamp
    type: timestamp
    timeFormat: rfc3339
    isEventTime: true
    description: "Event timestamp"
  - name: level
    type: string
    required: true
    description: "Log level (DEBUG, INFO, WARN, ERROR, FATAL)"
  - name: message
    type: string
    required: true
    description: "Log message"
  - name: service
    type: string
    required: true
    description: "Service name"
  - name: version
    type: string
    description: "Application version"
  - name: environment
    type: string
    description: "Environment (dev, staging, prod)"
  - name: trace_id
    type: string
    description: "Distributed tracing ID"
  - name: span_id
    type: string
    description: "Span ID for tracing"
  - name: user
    type: object
    description: "User information"
    fields:
      - name: id
        type: string
        description: "User ID"
      - name: email
        type: string
        description: "User email"
      - name: role
        type: string
        description: "User role"
  - name: request
    type: object
    description: "HTTP request details"
    fields:
      - name: method
        type: string
      - name: url
        type: string
      - name: headers
        type: object
      - name: body_size
        type: int
  - name: response
    type: object
    description: "HTTP response details"
    fields:
      - name: status_code
        type: int
      - name: headers
        type: object
      - name: body_size
        type: int
      - name: duration_ms
        type: float
  - name: error
    type: object
    description: "Error details if applicable"
    fields:
      - name: type
        type: string
      - name: message
        type: string
      - name: stack_trace
        type: string
  - name: tags
    type: array
    description: "Custom tags"
    element:
      type: string
  - name: custom_fields
    type: object
    description: "Custom application-specific fields"
EOF
  is_field_discovery_enabled = false
}

# Security events schema
resource "panther_schema" "security_events" {
  name                       = "Custom.SecurityEvents"
  description                = "Schema for security monitoring and audit events"
  spec                       = <<EOF
fields:
  - name: event_time
    type: timestamp
    timeFormat: unix_ms
    isEventTime: true
    description: "When the security event occurred"
  - name: event_type
    type: string
    required: true
    description: "Type of security event (login, logout, permission_change, etc.)"
  - name: severity
    type: string
    required: true
    description: "Event severity (low, medium, high, critical)"
  - name: outcome
    type: string
    required: true
    description: "Event outcome (success, failure, unknown)"
  - name: source_ip
    type: string
    description: "Source IP address"
  - name: destination_ip
    type: string
    description: "Destination IP address"
  - name: port
    type: int
    description: "Network port"
  - name: protocol
    type: string
    description: "Network protocol"
  - name: action
    type: string
    description: "Action taken (allow, deny, log)"
  - name: user_agent
    type: string
    description: "User agent string"
  - name: actor
    type: object
    description: "Actor performing the action"
    fields:
      - name: user_id
        type: string
      - name: username
        type: string
      - name: email
        type: string
      - name: role
        type: string
      - name: session_id
        type: string
  - name: target
    type: object
    description: "Target of the action"
    fields:
      - name: resource_type
        type: string
      - name: resource_id
        type: string
      - name: resource_name
        type: string
  - name: geo_location
    type: object
    description: "Geographic location information"
    fields:
      - name: country
        type: string
      - name: region
        type: string
      - name: city
        type: string
      - name: latitude
        type: float
      - name: longitude
        type: float
      - name: asn
        type: string
      - name: isp
        type: string
  - name: device
    type: object
    description: "Device information"
    fields:
      - name: type
        type: string
      - name: os
        type: string
      - name: browser
        type: string
      - name: fingerprint
        type: string
  - name: additional_data
    type: object
    description: "Additional event-specific data"
EOF
  is_field_discovery_enabled = true
}

# Database audit logs schema
resource "panther_schema" "database_audit_logs" {
  name                       = "Custom.DatabaseAuditLogs"
  description                = "Schema for database audit and access logs"
  spec                       = <<EOF
fields:
  - name: timestamp
    type: timestamp
    timeFormat: "layout:2006-01-02 15:04:05.000"
    isEventTime: true
    description: "Database operation timestamp"
  - name: database_name
    type: string
    required: true
    description: "Name of the database"
  - name: schema_name
    type: string
    description: "Database schema name"
  - name: table_name
    type: string
    description: "Table name"
  - name: operation
    type: string
    required: true
    description: "Database operation (SELECT, INSERT, UPDATE, DELETE, etc.)"
  - name: query
    type: string
    description: "SQL query executed"
  - name: query_hash
    type: string
    description: "Hash of the SQL query for grouping"
  - name: rows_affected
    type: int
    description: "Number of rows affected"
  - name: execution_time_ms
    type: float
    description: "Query execution time in milliseconds"
  - name: user
    type: object
    required: true
    description: "Database user information"
    fields:
      - name: username
        type: string
        required: true
      - name: role
        type: string
      - name: application
        type: string
      - name: host
        type: string
  - name: connection
    type: object
    description: "Connection information"
    fields:
      - name: id
        type: string
      - name: source_ip
        type: string
      - name: source_port
        type: int
      - name: ssl_enabled
        type: boolean
  - name: result
    type: object
    description: "Operation result"
    fields:
      - name: status
        type: string
      - name: error_code
        type: string
      - name: error_message
        type: string
  - name: sensitive_data_accessed
    type: boolean
    description: "Whether sensitive data was accessed"
  - name: data_classification
    type: array
    description: "Classification of accessed data"
    element:
      type: string
EOF
  is_field_discovery_enabled = false
}