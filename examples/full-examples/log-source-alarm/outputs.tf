output "source_id" {
  description = "The ID of the parent GCS log source"
  value       = panther_gcssource.parent.id
}

output "alarm_id" {
  description = "The composite ID of the alarm ({source_id}/{type})"
  value       = panther_log_source_alarm.example.id
}

output "alarm_minutes_threshold" {
  description = "Configured no-data evaluation period in minutes"
  value       = panther_log_source_alarm.example.minutes_threshold
}
