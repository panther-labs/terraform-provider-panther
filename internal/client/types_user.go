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

// UserRoleRef is the role reference object on a User.
type UserRoleRef struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// UserInput is the request body for creating or updating a User.
type UserInput struct {
	Email      string      `json:"email"`
	FamilyName string      `json:"familyName"`
	GivenName  string      `json:"givenName"`
	Role       UserRoleRef `json:"role"`
}

// User is the API response for a User (includes server-managed fields).
type User struct {
	ID             string      `json:"id"`
	CreatedAt      string      `json:"createdAt,omitempty"`
	Email          string      `json:"email"`
	Enabled        bool        `json:"enabled"`
	FamilyName     string      `json:"familyName"`
	GivenName      string      `json:"givenName"`
	LastLoggedInAt string      `json:"lastLoggedInAt,omitempty"`
	Role           UserRoleRef `json:"role"`
	Status         string      `json:"status,omitempty"`
}
