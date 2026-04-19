/*
Copyright 2023 Panther Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

// LogSourceAlarmInput is the PUT request body for creating or updating a log source alarm.
type LogSourceAlarmInput struct {
	MinutesThreshold int64 `json:"minutesThreshold"`
}

// LogSourceAlarm is the API response for GET and PUT. The GET response also includes a
// runtime `state` field (OK | ALARM | INSUFFICIENT_DATA) which this struct intentionally
// does NOT mirror — the provider scopes itself to declarative configuration and exposes
// no runtime observability. Consumers who need live alarm state should query the REST
// API directly or use a dedicated data source (future work).
type LogSourceAlarm struct {
	Type string `json:"type"`
	LogSourceAlarmInput
}
