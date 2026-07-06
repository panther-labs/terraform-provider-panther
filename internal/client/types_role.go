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

// RoleInput is the request body for creating or updating a Role.
type RoleInput struct {
	Name              string   `json:"name"`
	Permissions       []string `json:"permissions"`
	LogTypeAccess     []string `json:"logTypeAccess,omitempty"`
	LogTypeAccessKind string   `json:"logTypeAccessKind,omitempty"`
}

// Role is the API response for a Role (includes server-managed fields).
type Role struct {
	ID        string `json:"id"`
	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	RoleInput
}
