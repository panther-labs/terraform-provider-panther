output "web_access_logs_schema" {
  description = "Web access logs schema details"
  value = {
    id      = panther_schema.web_access_logs.id
    name    = panther_schema.web_access_logs.name
    version = panther_schema.web_access_logs.version
  }
}

output "application_logs_schema" {
  description = "Application logs schema details"
  value = {
    id      = panther_schema.application_logs.id
    name    = panther_schema.application_logs.name
    version = panther_schema.application_logs.version
  }
}

output "security_events_schema" {
  description = "Security events schema details"
  value = {
    id      = panther_schema.security_events.id
    name    = panther_schema.security_events.name
    version = panther_schema.security_events.version
  }
}

output "database_audit_logs_schema" {
  description = "Database audit logs schema details"
  value = {
    id      = panther_schema.database_audit_logs.id
    name    = panther_schema.database_audit_logs.name
    version = panther_schema.database_audit_logs.version
  }
}

output "all_schemas" {
  description = "All created schemas"
  value = {
    web_access_logs     = panther_schema.web_access_logs
    application_logs    = panther_schema.application_logs
    security_events     = panther_schema.security_events
    database_audit_logs = panther_schema.database_audit_logs
  }
}