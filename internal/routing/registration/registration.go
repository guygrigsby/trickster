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

package registration

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/Comcast/trickster/internal/cache"
	"github.com/Comcast/trickster/internal/cache/registration"
	"github.com/Comcast/trickster/internal/config"
	"github.com/Comcast/trickster/internal/proxy/methods"
	"github.com/Comcast/trickster/internal/proxy/model"
	"github.com/Comcast/trickster/internal/proxy/origins/clickhouse"
	"github.com/Comcast/trickster/internal/proxy/origins/influxdb"
	"github.com/Comcast/trickster/internal/proxy/origins/irondb"
	"github.com/Comcast/trickster/internal/proxy/origins/prometheus"
	"github.com/Comcast/trickster/internal/proxy/origins/reverseproxycache"
	"github.com/Comcast/trickster/internal/routing"
	"github.com/Comcast/trickster/internal/util/log"
	"github.com/Comcast/trickster/internal/util/middleware"
)

// ProxyClients maintains a list of proxy clients configured for use by Trickster
var ProxyClients = make(map[string]model.Client)

// RegisterProxyRoutes iterates the Trickster Configuration and registers the routes for the configured origins
func RegisterProxyRoutes() error {

	defaultOrigin := ""
	var ndo *config.OriginConfig // points to the origin config named "default"
	var cdo *config.OriginConfig // points to the origin config with IsDefault set to true

	// This iteration will ensure default origins are handled properly
	for k, o := range config.Origins {

		if !config.IsValidOriginType(o.OriginType) {
			return fmt.Errorf(`unknown origin type in origin config. originName: %s, originType: %s`, k, o.OriginType)
		}

		// Ensure only one default origin exists
		if o.IsDefault {
			if cdo != nil {
				return fmt.Errorf("only one origin can be marked as default. Found both %s and %s", defaultOrigin, k)
			}
			log.Debug("default origin identified", log.Pairs{"name": k})
			defaultOrigin = k
			cdo = o
			continue
		}

		// handle origin named "default" last as it needs special handling based on a full pass over the range
		if k == "default" {
			ndo = o
			continue
		}

		err := registerOriginRoutes(k, o)
		if err != nil {
			return err
		}
	}

	if ndo != nil {
		if cdo == nil {
			ndo.IsDefault = true
			cdo = ndo
			defaultOrigin = "default"
		} else {
			err := registerOriginRoutes("default", ndo)
			if err != nil {
				return err
			}
		}
	}

	if cdo != nil {
		return registerOriginRoutes(defaultOrigin, cdo)
	}

	return nil
}

func registerOriginRoutes(k string, o *config.OriginConfig) error {

	var client model.Client
	var c cache.Cache
	var err error

	c, err = registration.GetCache(o.CacheName)
	if err != nil {
		return err
	}

	log.Info("registering route paths", log.Pairs{"originName": k, "originType": o.OriginType, "upstreamHost": o.Host})

	switch strings.ToLower(o.OriginType) {
	case "prometheus", "":
		client, err = prometheus.NewClient(k, o, c)
	case "influxdb":
		client, err = influxdb.NewClient(k, o, c)
	case "irondb":
		client, err = irondb.NewClient(k, o, c)
	case "clickhouse":
		client, err = clickhouse.NewClient(k, o, c)
	case "rpc", "reverseproxycache":
		client, err = reverseproxycache.NewClient(k, o, c)
	}
	if err != nil {
		return err
	}
	if client != nil {
		ProxyClients[k] = client
		defaultPaths := client.DefaultPathConfigs(o)
		registerPathRoutes(client.Handlers(), o, c, defaultPaths)
	}
	return nil
}

// registerPathRoutes will take the provided default paths map,
// merge it with any path data in the provided originconfig, and then register
// the path routes to the appropriate handler from the provided handlers map
func registerPathRoutes(handlers map[string]http.Handler, o *config.OriginConfig, c cache.Cache,
	paths map[string]*config.PathConfig) {

	routing.Router.Use(
		middleware.Trace(o.Name, o.OriginType),
	)

	decorate := func(p *config.PathConfig) http.Handler {
		// Add Origin, Cache, and Path Configs to the HTTP Request's context
		p.Handler = middleware.WithConfigContext(o, c, p, p.Handler)
		if p.NoMetrics {
			return p.Handler
		}
		return middleware.Decorate(o.Name, o.OriginType, p.Path, p.Handler)
	}

	pathsWithVerbs := make(map[string]*config.PathConfig)
	for _, p := range paths {
		if len(p.Methods) == 0 {
			p.Methods = methods.CacheableHTTPMethods()
		}
		pathsWithVerbs[p.Path+"-"+strings.Join(p.Methods, "-")] = p
	}

	for k, p := range o.Paths {
		p.OriginConfig = o
		if p2, ok := pathsWithVerbs[k]; ok {
			p2.Merge(p)
			continue
		}
		p3 := config.NewPathConfig()
		p3.Merge(p)
		pathsWithVerbs[k] = p3
	}

	if h, ok := handlers["health"]; ok &&
		o.HealthCheckUpstreamPath != "" && o.HealthCheckVerb != "" {
		hp := "/trickster/health/" + o.Name
		log.Debug("registering health handler path", log.Pairs{"path": hp, "originName": o.Name, "upstreamPath": o.HealthCheckUpstreamPath, "upstreamVerb": o.HealthCheckVerb})
		routing.Router.PathPrefix(hp).Handler(middleware.WithConfigContext(o, nil, nil, h)).Methods(methods.CacheableHTTPMethods()...)
	}

	plist := make([]string, 0, len(pathsWithVerbs))
	deletes := make([]string, 0, len(pathsWithVerbs))
	for k, p := range pathsWithVerbs {
		if h, ok := handlers[p.HandlerName]; ok && h != nil {
			p.Handler = h
			plist = append(plist, k)
		} else {
			log.Info("invalid handler name for path", log.Pairs{"path": p.Path, "handlerName": p.HandlerName})
			deletes = append(deletes, p.Path)
		}
	}
	for _, p := range deletes {
		delete(pathsWithVerbs, p)
	}

	sort.Sort(ByLen(plist))
	for i := len(plist)/2 - 1; i >= 0; i-- {
		opp := len(plist) - 1 - i
		plist[i], plist[opp] = plist[opp], plist[i]
	}

	for _, v := range plist {
		p, ok := pathsWithVerbs[v]
		if !ok {
			continue
		}
		log.Debug("registering origin handler path",
			log.Pairs{"originName": o.Name, "path": v, "handlerName": p.HandlerName,
				"originHost": o.Host, "handledPath": "/" + o.Name + p.Path, "matchType": p.MatchType})
		if p.Handler != nil && len(p.Methods) > 0 {

			if p.Methods[0] == "*" {
				p.Methods = methods.AllHTTPMethods()
			}

			switch p.MatchType {
			case config.PathMatchTypePrefix:
				// Case where we path match by prefix
				// Host Header Routing
				routing.Router.PathPrefix(p.Path).Handler(decorate(p)).Methods(p.Methods...).Host(o.Name)
				// Path Routing
				routing.Router.PathPrefix("/" + o.Name + p.Path).Handler(decorate(p)).Methods(p.Methods...)
			default:
				// default to exact match
				// Host Header Routing
				routing.Router.Handle(p.Path, decorate(p)).Methods(p.Methods...).Host(o.Name)
				// Path Routing
				routing.Router.Handle("/"+o.Name+p.Path, decorate(p)).Methods(p.Methods...)
			}
		}
	}

	if o.IsDefault {
		log.Info("registering default origin handler paths", log.Pairs{"originName": o.Name})
		for _, v := range plist {
			p, ok := pathsWithVerbs[v]
			if !ok {
				continue
			}
			if p.Handler != nil && len(p.Methods) > 0 {
				log.Debug("registering default origin handler paths", log.Pairs{"originName": o.Name, "path": p.Path, "handlerName": p.HandlerName, "matchType": p.MatchType})
				switch p.MatchType {
				case config.PathMatchTypePrefix:
					// Case where we path match by prefix
					routing.Router.PathPrefix(p.Path).Handler(decorate(p)).Methods(p.Methods...)
				default:
					// default to exact match
					routing.Router.Handle(p.Path, decorate(p)).Methods(p.Methods...)
				}
				routing.Router.Handle(p.Path, decorate(p)).Methods(p.Methods...)
			}
		}
	}
	o.Paths = pathsWithVerbs
}

// ByLen allows sorting of a string slice by string length
type ByLen []string

func (a ByLen) Len() int {
	return len(a)
}

func (a ByLen) Less(i, j int) bool {
	return len(a[i]) < len(a[j])
}

func (a ByLen) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
