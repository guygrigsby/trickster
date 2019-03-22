/**
* Copyright 2018 Comcast Cable Communications Management, LLC
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
* http://www.apache.org/licenses/LICENSE-2.0
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package prometheus

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/Comcast/trickster/internal/proxy"
	"github.com/Comcast/trickster/internal/timeseries"
)

// BaseURL returns a URL in the form of schme://host/path based on the proxy configuration
func (c Client) BaseURL() *url.URL {
	u := &url.URL{}
	u.Scheme = c.Config.Scheme
	u.Host = c.Config.Host
	u.Path = c.Config.PathPrefix
	return u
}

// BuildUpstreamURL will merge the downstream request with the BaseURL to construct the full upstream URL
func (c Client) BuildUpstreamURL(r *http.Request) *url.URL {
	u := c.BaseURL()
	u.Path += r.URL.Path
	u.RawQuery = r.URL.RawQuery
	u.Fragment = r.URL.Fragment
	u.User = r.URL.User
	return u
}

// SetExtent will change the upstream request query to use the provided Extent
func (c Client) SetExtent(r *proxy.Request, extent *timeseries.Extent) {
	params := r.URL.Query()
	params.Set(upStart, strconv.FormatInt(extent.Start.Unix(), 10))
	params.Set(upEnd, strconv.FormatInt(extent.End.Unix(), 10))
	r.URL.RawQuery = params.Encode()
}