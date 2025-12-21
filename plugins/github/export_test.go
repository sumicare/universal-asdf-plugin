//
// Copyright (c) 2025 Sumicare
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package github

import (
	"context"
)

// NewClientForTests constructs a Client with explicit internals for external-package tests.
func NewClientForTests(httpClient HTTPClient, apiURL, authToken string) *Client {
	return &Client{
		httpClient: httpClient,
		apiURL:     apiURL,
		authToken:  authToken,
	}
}

// FetchJSONForTests exposes fetchJSON for external-package tests.
func (client *Client) FetchJSONForTests(ctx context.Context, url string, out any) error {
	return client.fetchJSON(ctx, url, out)
}
