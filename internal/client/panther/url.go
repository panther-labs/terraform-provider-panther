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

package panther

import "strings"

const GraphqlPath = "/v1/public/graphql"
const RestHttpSourcePath = "/v1/log-sources/http"

func TrimUrl(url string) string {
	if strings.HasSuffix(url, GraphqlPath) {
		return strings.TrimSuffix(url, GraphqlPath)
	}
	return url
}
