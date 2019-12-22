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

package config

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/Comcast/trickster/internal/proxy/headers"
)

// Config is the Running Configuration for Trickster
var Config *TricksterConfig

// Main is the Main subsection of the Running Configuration
var Main *MainConfig

// Origins is the Origin Map subsection of the Running Configuration
var Origins map[string]*OriginConfig

// Caches is the Cache Map subsection of the Running Configuration
var Caches map[string]*CachingConfig

// Frontend is the Proxy Server subsection of the Running Configuration
var Frontend *FrontendConfig

// Logging is the Logging subsection of the Running Configuration
var Logging *LoggingConfig

// Metrics is the Metrics subsection of the Running Configuration
var Metrics *MetricsConfig

// Tracing defines destricbuted trace options for the Running Configuration
var Tracing *TracingConfig

// NegativeCacheConfigs is the NegativeCacheConfig subsection of the Running Configuration
var NegativeCacheConfigs map[string]NegativeCacheConfig

// Flags is a collection of command line flags that Trickster loads.
var Flags = TricksterFlags{}
var providedOriginURL string
var providedOriginType string

// LoaderWarnings holds warnings generated during config load (before the logger is initialized),
// so they can be logged at the end of the loading process
var LoaderWarnings = make([]string, 0, 0)

// TricksterConfig is the main configuration object
type TricksterConfig struct {
	// Main is the primary MainConfig section
	Main *MainConfig `toml:"main"`
	// Origins is a map of OriginConfigs
	Origins map[string]*OriginConfig `toml:"origins"`
	// Caches is a map of CacheConfigs
	Caches map[string]*CachingConfig `toml:"caches"`
	// ProxyServer is provides configurations about the Proxy Front End
	Frontend *FrontendConfig `toml:"frontend"`
	// Logging provides configurations that affect logging behavior
	Logging *LoggingConfig `toml:"logging"`
	// Metrics provides configurations for collecting Metrics about the application
	Metrics *MetricsConfig `toml:"metrics"`
	// Tracing provides the distributed tracing configuration
	Tracing *TracingConfig `toml:"tracing"`
	// NegativeCacheConfigs is a map of NegativeCacheConfigs
	NegativeCacheConfigs map[string]NegativeCacheConfig `toml:"negative_caches"`

	activeCaches map[string]bool
}

// MainConfig is a collection of general configuration values.
type MainConfig struct {
	// InstanceID represents a unique ID for the current instance, when multiple instances on the same host
	InstanceID int `toml:"instance_id"`
	// ConfigHandlerPath provides the path to register the Config Handler for outputting the running configuration
	ConfigHandlerPath string `toml:"config_handler_path"`
	// PingHandlerPath provides the path to register the Ping Handler for checking that Trickster is running
	PingHandlerPath string `toml:"ping_handler_path"`
}

// OriginConfig is a collection of configurations for prometheus origins proxied by Trickster
type OriginConfig struct {

	// HTTP and Proxy Configurations
	//
	// IsDefault indicates if this is the default origin for any request not matching a configured route
	IsDefault bool `toml:"is_default"`
	// OriginType describes the type of origin (e.g., 'prometheus')
	OriginType string `toml:"origin_type"`
	// OriginURL provides the base upstream URL for all proxied requests to this origin.
	// it can be as simple as http://example.com or as complex as https://example.com:8443/path/prefix
	OriginURL string `toml:"origin_url"`
	// TimeoutSecs defines how long the HTTP request will wait for a response before timing out
	TimeoutSecs int64 `toml:"timeout_secs"`
	// KeepAliveTimeoutSecs defines how long an open keep-alive HTTP connection remains idle before closing
	KeepAliveTimeoutSecs int64 `toml:"keep_alive_timeout_secs"`
	// MaxIdleConns defines maximum number of open keep-alive connections to maintain
	MaxIdleConns int `toml:"max_idle_conns"`
	// CacheName provides the name of the configured cache where the origin client will store it's cache data
	CacheName string `toml:"cache_name"`
	// CacheKeyPrefix defines the cache key prefix the origin will use when writing objects to the cache
	CacheKeyPrefix string `toml:"cache_key_prefix"`
	// HealthCheckUpstreamPath provides the URL path for the upstream health check
	HealthCheckUpstreamPath string `toml:"health_check_upstream_path"`
	// HealthCheckVerb provides the HTTP verb to use when making an upstream health check
	HealthCheckVerb string `toml:"health_check_verb"`
	// HealthCheckQuery provides the HTTP query parameters to use when making an upstream health check
	HealthCheckQuery string `toml:"health_check_query"`
	// HealthCheckHeaders provides the HTTP Headers to apply when making an upstream health check
	HealthCheckHeaders map[string]string `toml:"health_check_headers"`
	// Object Proxy Cache and Delta Proxy Cache Configurations
	// TimeseriesRetentionFactor limits the maximum the number of chronological timestamps worth of data to store in cache for each query
	TimeseriesRetentionFactor int `toml:"timeseries_retention_factor"`
	// TimeseriesEvictionMethodName specifies which methodology ("oldest", "lru") is used to identify timeseries to evict from a full cache object
	TimeseriesEvictionMethodName string `toml:"timeseries_eviction_method"`
	// FastForwardDisable indicates whether the FastForward feature should be disabled for this origin
	FastForwardDisable bool `toml:"fast_forward_disable"`
	// BackfillToleranceSecs prevents values with timestamps newer than the provided number of seconds from being cached
	// this allows propagation of upstream backfill operations that modify recently-served data
	BackfillToleranceSecs int64 `toml:"backfill_tolerance_secs"`
	// PathList is a list of PathConfigs that control the behavior of the given paths when requested
	Paths map[string]*PathConfig `toml:"paths"`
	// NegativeCacheName provides the name of the Negative Cache Config to be used by this Origin
	NegativeCacheName string `toml:"negative_cache_name"`
	// TimeseriesTTLSecs specifies the cache TTL of timeseries objects
	TimeseriesTTLSecs int `toml:"timeseries_ttl_secs"`
	// TimeseriesTTLSecs specifies the cache TTL of fast forward data
	FastForwardTTLSecs int `toml:"fastforward_ttl_secs"`
	// MaxTTLSecs specifies the maximum allowed TTL for any cache object
	MaxTTLSecs int `toml:"max_ttl_secs"`
	// RevalidationFactor specifies how many times to multiply the object freshness lifetime by to calculate an absolute cache TTL
	RevalidationFactor int `toml:"revalidation_factor"`
	// MaxObjectSizeBytes specifies the max objectsize to be accepted for any given cache object
	MaxObjectSizeBytes int `toml:"max_object_size_bytes"`

	// TLS is the TLS Configuration for the Frontend and Backend
	TLS *TLSConfig `toml:"tls"`
	// RequireTLS, when true, indicates this Origin Config's paths must only be registered with the TLS Router
	RequireTLS bool `toml:"require_tls"`

	// Synthesized Configurations
	// These configurations are parsed versions of those defined above, and are what Trickster uses internally
	//
	// Name is the Name of the origin, taken from the Key in the Origins map[string]*OriginConfig
	Name string `toml:"-"`
	// Timeout is the time.Duration representation of TimeoutSecs
	Timeout time.Duration `toml:"-"`
	// BackfillTolerance is the time.Duration representation of BackfillToleranceSecs
	BackfillTolerance time.Duration `toml:"-"`
	// ValueRetention is the time.Duration representation of ValueRetentionSecs
	ValueRetention time.Duration `toml:"-"`
	// Scheme is the layer 7 protocol indicator (e.g. 'http'), derived from OriginURL
	Scheme string `toml:"-"`
	// Host is the upstream hostname/IP[:port] the origin client will connect to when fetching uncached data, derived from OriginURL
	Host string `toml:"-"`
	// PathPrefix provides any prefix added to the front of the requested path when constructing the upstream request url, derived from OriginURL
	PathPrefix string `toml:"-"`
	// NegativeCache provides a map for the negative cache, with TTLs converted to time.Durations
	NegativeCache map[int]time.Duration `toml:"-"`
	// TimeseriesRetention when subtracted from time.Now() represents the oldest allowable timestamp in a timeseries when EvictionMethod is 'oldest'
	TimeseriesRetention time.Duration `toml:"-"`
	// TimeseriesEvictionMethod is the parsed value of TimeseriesEvictionMethodName
	TimeseriesEvictionMethod TimeseriesEvictionMethod `toml:"-"`
	// TimeseriesTTL is the parsed value of TimeseriesTTLSecs
	TimeseriesTTL time.Duration `toml:"-"`
	// FastForwardTTL is the parsed value of FastForwardTTL
	FastForwardTTL time.Duration `toml:"-"`
	// FastForwardPath is the PathConfig to use for upstream Fast Forward Requests
	FastForwardPath *PathConfig `toml:"-"`
	// MaxTTL is the parsed value of MaxTTLSecs
	MaxTTL time.Duration `toml:"-"`
}

// CachingConfig is a collection of defining the Trickster Caching Behavior
type CachingConfig struct {
	// Name is the Name of the cache, taken from the Key in the Caches map[string]*CacheConfig
	Name string `toml:"-"`
	// Type represents the type of cache that we wish to use: "boltdb", "memory", "filesystem", or "redis"
	CacheType string `toml:"cache_type"`
	// Compression determines whether objects should be compressed when writing to the cache
	Compression bool `toml:"compression"`
	// Index provides options for the Cache Index
	Index CacheIndexConfig `toml:"index"`
	// Redis provides options for Redis caching
	Redis RedisCacheConfig `toml:"redis"`
	// Filesystem provides options for Filesystem caching
	Filesystem FilesystemCacheConfig `toml:"filesystem"`
	// BBolt provides options for BBolt caching
	BBolt BBoltCacheConfig `toml:"bbolt"`
	// Badger provides options for BadgerDB caching
	Badger BadgerCacheConfig `toml:"badger"`

	//  Synthetic Values

	// CacheTypeID represents the internal constant for the provided CacheType string
	// and is automatically populated at startup
	CacheTypeID CacheType `toml:"-"`
}

// CacheIndexConfig defines the operation of the Cache Indexer
type CacheIndexConfig struct {
	// ReapIntervalSecs defines how long the Cache Index reaper sleeps between reap cycles
	ReapIntervalSecs int `toml:"reap_interval_secs"`
	// FlushIntervalSecs sets how often the Cache Index saves its metadata to the cache from application memory
	FlushIntervalSecs int `toml:"flush_interval_secs"`
	// MaxSizeBytes indicates how large the cache can grow in bytes before the Index evicts
	// least-recently-accessed items.
	MaxSizeBytes int64 `toml:"max_size_bytes"`
	// MaxSizeBackoffBytes indicates how far below max_size_bytes the cache size must be
	// to complete a byte-size-based eviction exercise.
	MaxSizeBackoffBytes int64 `toml:"max_size_backoff_bytes"`
	// MaxSizeObjects  indicates how large the cache can grow in objects before the Index
	// evicts least-recently-accessed items.
	MaxSizeObjects int64 `toml:"max_size_objects"`
	// MaxSizeBackoffObjects indicates how far under max_size_objects the cache size must
	// be to complete object-size-based eviction exercise.
	MaxSizeBackoffObjects int64 `toml:"max_size_backoff_objects"`

	ReapInterval  time.Duration `toml:"-"`
	FlushInterval time.Duration `toml:"-"`
}

// RedisCacheConfig is a collection of Configurations for Connecting to Redis
type RedisCacheConfig struct {
	// ClientType defines the type of Redis Client ("standard", "cluster", "sentinel")
	ClientType string `toml:"client_type"`
	// Protocol represents the connection method (e.g., "tcp", "unix", etc.)
	Protocol string `toml:"protocol"`
	// Endpoint represents FQDN:port or IPAddress:Port of the Redis Endpoint
	Endpoint string `toml:"endpoint"`
	// Endpoints represents FQDN:port or IPAddress:Port collection of a Redis Cluster or Sentinel Nodes
	Endpoints []string `toml:"endpoints"`
	// Password can be set when using password protected redis instance.
	Password string `toml:"password"`
	// SentinelMaster should be set when using Redis Sentinel to indicate the Master Node
	SentinelMaster string `toml:"sentinel_master"`
	// DB is the Database to be selected after connecting to the server.
	DB int `toml:"db"`
	// MaxRetries is the maximum number of retries before giving up on the command
	MaxRetries int `toml:"max_retries"`
	// MinRetryBackoffMS is the minimum backoff between each retry.
	MinRetryBackoffMS int `toml:"min_retry_backoff_ms"`
	// MaxRetryBackoffMS is the Maximum backoff between each retry.
	MaxRetryBackoffMS int `toml:"max_retry_backoff_ms"`
	// DialTimeoutMS is the timeout for establishing new connections.
	DialTimeoutMS int `toml:"dial_timeout_ms"`
	// ReadTimeoutMS is the timeout for socket reads. If reached, commands will fail with a timeout instead of blocking.
	ReadTimeoutMS int `toml:"read_timeout_ms"`
	// WriteTimeoutMS is the timeout for socket writes. If reached, commands will fail with a timeout instead of blocking.
	WriteTimeoutMS int `toml:"write_timeout_ms"`
	// PoolSize is the maximum number of socket connections.
	PoolSize int `toml:"pool_size"`
	// MinIdleConns is the minimum number of idle connections which is useful when establishing new connection is slow.
	MinIdleConns int `toml:"min_idle_conns"`
	// MaxConnAgeMS is the connection age at which client retires (closes) the connection.
	MaxConnAgeMS int `toml:"max_conn_age_ms"`
	// PoolTimeoutMS is the amount of time client waits for connection if all connections are busy before returning an error.
	PoolTimeoutMS int `toml:"pool_timeout_ms"`
	// IdleTimeoutMS is the amount of time after which client closes idle connections.
	IdleTimeoutMS int `toml:"idle_timeout_ms"`
	// IdleCheckFrequencyMS is the frequency of idle checks made by idle connections reaper.
	IdleCheckFrequencyMS int `toml:"idle_check_frequency_ms"`
}

// BadgerCacheConfig is a collection of Configurations for storing cached data on the Filesystem in a Badger key-value store
type BadgerCacheConfig struct {
	// Directory represents the path on disk where the Badger database should store data
	Directory string `toml:"directory"`
	// ValueDirectory represents the path on disk where the Badger database will store its value log.
	ValueDirectory string `toml:"value_directory"`
}

// BBoltCacheConfig is a collection of Configurations for storing cached data on the Filesystem
type BBoltCacheConfig struct {
	// Filename represents the filename (including path) of the BotlDB database
	Filename string `toml:"filename"`
	// Bucket represents the name of the bucket within BBolt under which Trickster's keys will be stored.
	Bucket string `toml:"bucket"`
}

// FilesystemCacheConfig is a collection of Configurations for storing cached data on the Filesystem
type FilesystemCacheConfig struct {
	// CachePath represents the path on disk where our cache will live
	CachePath string `toml:"cache_path"`
}

// FrontendConfig is a collection of configurations for the main http frontend for the application
type FrontendConfig struct {
	// ListenAddress is IP address for the main http listener for the application
	ListenAddress string `toml:"listen_address"`
	// ListenPort is TCP Port for the main http listener for the application
	ListenPort int `toml:"listen_port"`
	// TLSListenAddress is IP address for the tls  http listener for the application
	TLSListenAddress string `toml:"tls_listen_address"`
	// TLSListenPort is the TCP Port for the tls http listener for the application
	TLSListenPort int `toml:"tls_listen_port"`
	// ConnectionsLimit indicates how many concurrent front end connections trickster will handle at any time
	ConnectionsLimit int `toml:"connections_limit"`

	// ServeTLS indicates whether to listen and serve on the TLS port, meaning
	// at least one origin configuration has a valid certificate and key file configured.
	ServeTLS bool `toml:"-"`
}

// LoggingConfig is a collection of Logging configurations
type LoggingConfig struct {
	// LogFile provides the filepath to the instances's logfile. Set as empty string to Log to Console
	LogFile string `toml:"log_file"`
	// LogLevel provides the most granular level (e.g., DEBUG, INFO, ERROR) to log
	LogLevel string `toml:"log_level"`
}

// MetricsConfig is a collection of Metrics Collection configurations
type MetricsConfig struct {
	// ListenAddress is IP address from which the Application Metrics are available for pulling at /metrics
	ListenAddress string `toml:"listen_address"`
	// ListenPort is TCP Port from which the Application Metrics are available for pulling at /metrics
	ListenPort int `toml:"listen_port"`
}

// Tracing provides the distributed tracing configuration
type TracingConfig struct {
	// Implementation is the particular implementation to use TODO generate with Rob Pike's Stringer
	Implementation string `toml:"tracer_implementation"`
	// CollectorEndpoint is the URL of the trace collector it MUST be of Implementation implementation
	CollectorEndpoint string `toml:"tracing_collector"`
}

// NegativeCacheConfig is a collection of response codes and their TTLs
type NegativeCacheConfig map[string]int

// Copy returns an exact copy of a NegativeCacheConfig
func (nc NegativeCacheConfig) Copy() NegativeCacheConfig {
	nc2 := make(NegativeCacheConfig)
	for k, v := range nc {
		nc2[k] = v
	}
	return nc2
}

// NewConfig returns a Config initialized with default values.
func NewConfig() *TricksterConfig {
	return &TricksterConfig{
		Caches: map[string]*CachingConfig{
			"default": NewCacheConfig(),
		},
		Logging: &LoggingConfig{
			LogFile:  defaultLogFile,
			LogLevel: defaultLogLevel,
		},
		Main: &MainConfig{
			ConfigHandlerPath: defaultConfigHandlerPath,
			PingHandlerPath:   defaultPingHandlerPath,
		},
		Metrics: &MetricsConfig{
			ListenPort: defaultMetricsListenPort,
		},
		Tracing: &TracingConfig{
			Implementation:    defaultTracerImplemetation,
			CollectorEndpoint: "",
		},
		Origins: map[string]*OriginConfig{
			"default": NewOriginConfig(),
		},
		Frontend: &FrontendConfig{
			ListenPort: defaultProxyListenPort,
		},
		NegativeCacheConfigs: map[string]NegativeCacheConfig{
			"default": NewNegativeCacheConfig(),
		},
	}
}

// NewNegativeCacheConfig returns an empty NegativeCacheConfig
func NewNegativeCacheConfig() NegativeCacheConfig {
	return NegativeCacheConfig{}
}

// NewCacheConfig will return a pointer to an OriginConfig with the default configuration settings
func NewCacheConfig() *CachingConfig {

	return &CachingConfig{
		CacheType:   defaultCacheType,
		CacheTypeID: defaultCacheTypeID,
		Compression: defaultCacheCompression,
		Redis:       RedisCacheConfig{ClientType: defaultRedisClientType, Protocol: defaultRedisProtocol, Endpoint: defaultRedisEndpoint, Endpoints: []string{defaultRedisEndpoint}},
		Filesystem:  FilesystemCacheConfig{CachePath: defaultCachePath},
		BBolt:       BBoltCacheConfig{Filename: defaultBBoltFile, Bucket: defaultBBoltBucket},
		Badger:      BadgerCacheConfig{Directory: defaultCachePath, ValueDirectory: defaultCachePath},
		Index: CacheIndexConfig{
			ReapIntervalSecs:      defaultCacheIndexReap,
			FlushIntervalSecs:     defaultCacheIndexFlush,
			MaxSizeBytes:          defaultCacheMaxSizeBytes,
			MaxSizeBackoffBytes:   defaultMaxSizeBackoffBytes,
			MaxSizeObjects:        defaultMaxSizeObjects,
			MaxSizeBackoffObjects: defaultMaxSizeBackoffObjects,
		},
	}
}

// NewOriginConfig will return a pointer to an OriginConfig with the default configuration settings
func NewOriginConfig() *OriginConfig {
	return &OriginConfig{
		BackfillTolerance:            defaultBackfillToleranceSecs,
		BackfillToleranceSecs:        defaultBackfillToleranceSecs,
		CacheName:                    defaultOriginCacheName,
		CacheKeyPrefix:               "",
		HealthCheckQuery:             defaultHealthCheckQuery,
		HealthCheckUpstreamPath:      defaultHealthCheckPath,
		HealthCheckVerb:              defaultHealthCheckVerb,
		HealthCheckHeaders:           make(map[string]string),
		KeepAliveTimeoutSecs:         defaultKeepAliveTimeoutSecs,
		MaxIdleConns:                 defaultMaxIdleConns,
		NegativeCache:                make(map[int]time.Duration),
		NegativeCacheName:            defaultOriginNegativeCacheName,
		Paths:                        make(map[string]*PathConfig),
		Timeout:                      time.Second * defaultOriginTimeoutSecs,
		TimeoutSecs:                  defaultOriginTimeoutSecs,
		TimeseriesEvictionMethod:     defaultOriginTEM,
		TimeseriesEvictionMethodName: defaultOriginTEMName,
		TimeseriesRetention:          defaultOriginTRF,
		TimeseriesRetentionFactor:    defaultOriginTRF,
		TimeseriesTTLSecs:            defaultTimeseriesTTLSecs,
		FastForwardTTLSecs:           defaultFastForwardTTLSecs,
		TimeseriesTTL:                defaultTimeseriesTTLSecs * time.Second,
		FastForwardTTL:               defaultFastForwardTTLSecs * time.Second,
		MaxTTLSecs:                   defaultMaxTTLSecs,
		MaxTTL:                       defaultMaxTTLSecs * time.Second,
		RevalidationFactor:           defaultRevalidationFactor,
		MaxObjectSizeBytes:           defaultMaxObjectSizeBytes,
		TLS:                          &TLSConfig{},
	}
}

// loadFile loads application configuration from a TOML-formatted file.
func (c *TricksterConfig) loadFile() error {
	md, err := toml.DecodeFile(Flags.ConfigPath, c)
	if err != nil {
		c.setDefaults(&toml.MetaData{})
		return err
	}
	err = c.setDefaults(&md)
	return err
}

func (c *TricksterConfig) setDefaults(metadata *toml.MetaData) error {

	c.processOriginConfigs(metadata)
	c.processCachingConfigs(metadata)
	err := c.validateConfigMappings()
	if err != nil {
		return err
	}

	err = c.verifyTLSConfigs()

	return err
}

var pathMembers = []string{"path", "match_type", "handler", "methods", "cache_key_params", "cache_key_headers", "default_ttl_secs",
	"request_headers", "response_headers", "response_headers", "response_code", "response_body", "no_metrics", "progressive_collapsed_forwarding"}

func (c *TricksterConfig) validateConfigMappings() error {
	for k, oc := range c.Origins {
		if _, ok := c.Caches[oc.CacheName]; !ok {
			return fmt.Errorf("invalid cache name [%s] provided in origin config [%s]", oc.CacheName, k)
		}
	}
	return nil
}

func (c *TricksterConfig) processOriginConfigs(metadata *toml.MetaData) {

	c.activeCaches = make(map[string]bool)

	for k, v := range c.Origins {

		oc := NewOriginConfig()
		oc.Name = k

		if metadata.IsDefined("origins", k, "origin_type") {
			oc.OriginType = v.OriginType
		}

		if metadata.IsDefined("origins", k, "is_default") {
			oc.IsDefault = v.IsDefault
		}
		// If there is only one origin and is_default is not explicitly false, make it true
		if len(c.Origins) == 1 && (!metadata.IsDefined("origins", k, "is_default")) {
			oc.IsDefault = true
		}

		if metadata.IsDefined("origins", k, "require_tls") {
			oc.RequireTLS = v.RequireTLS
		}

		if metadata.IsDefined("origins", k, "cache_name") {
			oc.CacheName = v.CacheName
		}
		c.activeCaches[oc.CacheName] = true

		if metadata.IsDefined("origins", k, "cache_key_prefix") {
			oc.CacheKeyPrefix = v.CacheKeyPrefix
		}

		if metadata.IsDefined("origins", k, "origin_url") {
			oc.OriginURL = v.OriginURL
		}

		if metadata.IsDefined("origins", k, "timeout_secs") {
			oc.TimeoutSecs = v.TimeoutSecs
		}

		if metadata.IsDefined("origins", k, "max_idle_conns") {
			oc.MaxIdleConns = v.MaxIdleConns
		}

		if metadata.IsDefined("origins", k, "keep_alive_timeout_secs") {
			oc.KeepAliveTimeoutSecs = v.KeepAliveTimeoutSecs
		}

		if metadata.IsDefined("origins", k, "timeseries_retention_factor") {
			oc.TimeseriesRetentionFactor = v.TimeseriesRetentionFactor
		}

		if metadata.IsDefined("origins", k, "timeseries_eviction_method") {
			oc.TimeseriesEvictionMethodName = strings.ToLower(v.TimeseriesEvictionMethodName)
			if p, ok := timeseriesEvictionMethodNames[oc.TimeseriesEvictionMethodName]; ok {
				oc.TimeseriesEvictionMethod = p
			}
		}

		if metadata.IsDefined("origins", k, "timeseries_ttl_secs") {
			oc.TimeseriesTTLSecs = v.TimeseriesTTLSecs
		}

		if metadata.IsDefined("origins", k, "max_ttl_secs") {
			oc.MaxTTLSecs = v.MaxTTLSecs
		}

		if metadata.IsDefined("origins", k, "fastforward_ttl_secs") {
			oc.FastForwardTTLSecs = v.FastForwardTTLSecs
		}

		if metadata.IsDefined("origins", k, "fast_forward_disable") {
			oc.FastForwardDisable = v.FastForwardDisable
		}

		if metadata.IsDefined("origins", k, "backfill_tolerance_secs") {
			oc.BackfillToleranceSecs = v.BackfillToleranceSecs
		}

		if metadata.IsDefined("origins", k, "paths") {
			var j = 0
			for l, p := range v.Paths {
				if len(p.Methods) == 0 {
					p.Methods = []string{http.MethodGet, http.MethodHead}
				}
				p.custom = make([]string, 0, 0)
				for _, pm := range pathMembers {
					if metadata.IsDefined("origins", k, "paths", l, pm) {
						p.custom = append(p.custom, pm)
					}
				}
				if metadata.IsDefined("origins", k, "paths", l, "response_body") {
					p.ResponseBodyBytes = []byte(p.ResponseBody)
					p.HasCustomResponseBody = true
				}

				if mt, ok := pathMatchTypeNames[strings.ToLower(p.MatchTypeName)]; ok {
					p.MatchType = mt
					p.MatchTypeName = p.MatchType.String()
				} else {
					p.MatchType = PathMatchTypeExact
					p.MatchTypeName = p.MatchType.String()
				}
				oc.Paths[p.Path+"-"+strings.Join(p.Methods, "-")] = p
				j++
			}
		}

		if metadata.IsDefined("origins", k, "negative_cache_name") {
			oc.NegativeCacheName = v.NegativeCacheName
		}

		if metadata.IsDefined("origins", k, "health_check_upstream_path") {
			oc.HealthCheckUpstreamPath = v.HealthCheckUpstreamPath
		}

		if metadata.IsDefined("origins", k, "health_check_verb") {
			oc.HealthCheckVerb = v.HealthCheckVerb
		}

		if metadata.IsDefined("origins", k, "health_check_query") {
			oc.HealthCheckQuery = v.HealthCheckQuery
		}

		if metadata.IsDefined("origins", k, "health_check_headers") {
			oc.HealthCheckHeaders = v.HealthCheckHeaders
		}

		if metadata.IsDefined("origins", k, "max_object_size_bytes") {
			oc.MaxObjectSizeBytes = v.MaxObjectSizeBytes
		}

		if metadata.IsDefined("origins", k, "tls") {
			oc.TLS = &TLSConfig{
				InsecureSkipVerify:        v.TLS.InsecureSkipVerify,
				CertificateAuthorityPaths: v.TLS.CertificateAuthorityPaths,
				PrivateKeyPath:            v.TLS.PrivateKeyPath,
				FullChainCertPath:         v.TLS.FullChainCertPath,
				ClientCertPath:            v.TLS.ClientCertPath,
				ClientKeyPath:             v.TLS.ClientKeyPath,
			}
		}

		c.Origins[k] = oc
	}
}

func (c *TricksterConfig) processCachingConfigs(metadata *toml.MetaData) {

	// setCachingDefaults assumes that processOriginConfigs was just ran

	for k, v := range c.Caches {

		if _, ok := c.activeCaches[k]; !ok {
			// a configured cache was not used by any origin. don't even instantiate it
			delete(c.Caches, k)
			continue
		}

		cc := NewCacheConfig()
		cc.Name = k

		if metadata.IsDefined("caches", k, "cache_type") {
			cc.CacheType = strings.ToLower(v.CacheType)
			if n, ok := CacheTypeNames[cc.CacheType]; ok {
				cc.CacheTypeID = n
			}
		}

		if metadata.IsDefined("caches", k, "compression") {
			cc.Compression = v.Compression
		}

		if metadata.IsDefined("caches", k, "index", "reap_interval_secs") {
			cc.Index.ReapIntervalSecs = v.Index.ReapIntervalSecs
		}

		if metadata.IsDefined("caches", k, "index", "flush_interval_secs") {
			cc.Index.FlushIntervalSecs = v.Index.FlushIntervalSecs
		}

		if metadata.IsDefined("caches", k, "index", "max_size_bytes") {
			cc.Index.MaxSizeBytes = v.Index.MaxSizeBytes
		}

		if metadata.IsDefined("caches", k, "index", "max_size_backoff_bytes") {
			cc.Index.MaxSizeBackoffBytes = v.Index.MaxSizeBackoffBytes
		}

		if metadata.IsDefined("caches", k, "index", "max_size_objects") {
			cc.Index.MaxSizeObjects = v.Index.MaxSizeObjects
		}

		if metadata.IsDefined("caches", k, "index", "max_size_backoff_objects") {
			cc.Index.MaxSizeBackoffObjects = v.Index.MaxSizeBackoffObjects
		}

		if cc.CacheTypeID == CacheTypeRedis {

			var hasEndpoint, hasEndpoints bool

			ct := strings.ToLower(v.Redis.ClientType)
			if metadata.IsDefined("caches", k, "redis", "client_type") {
				cc.Redis.ClientType = ct
			}

			if metadata.IsDefined("caches", k, "redis", "protocol") {
				cc.Redis.Protocol = v.Redis.Protocol
			}

			if metadata.IsDefined("caches", k, "redis", "endpoint") {
				cc.Redis.Endpoint = v.Redis.Endpoint
				hasEndpoint = true
			}

			if metadata.IsDefined("caches", k, "redis", "endpoints") {
				cc.Redis.Endpoints = v.Redis.Endpoints
				hasEndpoints = true
			}

			if cc.Redis.ClientType == "standard" {
				if hasEndpoints && !hasEndpoint {
					LoaderWarnings = append(LoaderWarnings, "'standard' redis type configured, but 'endpoints' value is provided instead of 'endpoint'")
				}
			} else {
				if hasEndpoint && !hasEndpoints {
					LoaderWarnings = append(LoaderWarnings, fmt.Sprintf("'%s' redis type configured, but 'endpoint' value is provided instead of 'endpoints'", cc.Redis.ClientType))
				}
			}

			if metadata.IsDefined("caches", k, "redis", "sentinel_master") {
				cc.Redis.SentinelMaster = v.Redis.SentinelMaster
			}

			if metadata.IsDefined("caches", k, "redis", "password") {
				cc.Redis.Password = v.Redis.Password
			}

			if metadata.IsDefined("caches", k, "redis", "db") {
				cc.Redis.DB = v.Redis.DB
			}

			if metadata.IsDefined("caches", k, "redis", "max_retries") {
				cc.Redis.MaxRetries = v.Redis.MaxRetries
			}

			if metadata.IsDefined("caches", k, "redis", "min_retry_backoff_ms") {
				cc.Redis.MinRetryBackoffMS = v.Redis.MinRetryBackoffMS
			}

			if metadata.IsDefined("caches", k, "redis", "max_retry_backoff_ms") {
				cc.Redis.MaxRetryBackoffMS = v.Redis.MaxRetryBackoffMS
			}

			if metadata.IsDefined("caches", k, "redis", "dial_timeout_ms") {
				cc.Redis.DialTimeoutMS = v.Redis.DialTimeoutMS
			}

			if metadata.IsDefined("caches", k, "redis", "read_timeout_ms") {
				cc.Redis.ReadTimeoutMS = v.Redis.ReadTimeoutMS
			}

			if metadata.IsDefined("caches", k, "redis", "write_timeout_ms") {
				cc.Redis.WriteTimeoutMS = v.Redis.WriteTimeoutMS
			}

			if metadata.IsDefined("caches", k, "redis", "pool_size") {
				cc.Redis.PoolSize = v.Redis.PoolSize
			}

			if metadata.IsDefined("caches", k, "redis", "min_idle_conns") {
				cc.Redis.MinIdleConns = v.Redis.MinIdleConns
			}

			if metadata.IsDefined("caches", k, "redis", "max_conn_age_ms") {
				cc.Redis.MaxConnAgeMS = v.Redis.MaxConnAgeMS
			}

			if metadata.IsDefined("caches", k, "redis", "pool_timeout_ms") {
				cc.Redis.PoolTimeoutMS = v.Redis.PoolTimeoutMS
			}

			if metadata.IsDefined("caches", k, "redis", "idle_timeout_ms") {
				cc.Redis.IdleTimeoutMS = v.Redis.IdleTimeoutMS
			}

			if metadata.IsDefined("caches", k, "redis", "idle_check_frequency_ms") {
				cc.Redis.IdleCheckFrequencyMS = v.Redis.IdleCheckFrequencyMS
			}
		}

		if metadata.IsDefined("caches", k, "filesystem", "cache_path") {
			cc.Filesystem.CachePath = v.Filesystem.CachePath
		}

		if metadata.IsDefined("caches", k, "bbolt", "filename") {
			cc.BBolt.Filename = v.BBolt.Filename
		}

		if metadata.IsDefined("caches", k, "bbolt", "bucket") {
			cc.BBolt.Bucket = v.BBolt.Bucket
		}

		if metadata.IsDefined("caches", k, "badger", "directory") {
			cc.Badger.Directory = v.Badger.Directory
		}

		if metadata.IsDefined("caches", k, "badger", "value_directory") {
			cc.Badger.ValueDirectory = v.Badger.ValueDirectory
		}

		c.Caches[k] = cc
	}
}

func (c *TricksterConfig) copy() *TricksterConfig {

	nc := NewConfig()
	delete(nc.Caches, "default")
	delete(nc.Origins, "default")

	nc.Main.ConfigHandlerPath = c.Main.ConfigHandlerPath
	nc.Main.InstanceID = c.Main.InstanceID
	nc.Main.PingHandlerPath = c.Main.PingHandlerPath

	nc.Logging.LogFile = c.Logging.LogFile
	nc.Logging.LogLevel = c.Logging.LogLevel

	nc.Metrics.ListenAddress = c.Metrics.ListenAddress
	nc.Metrics.ListenPort = c.Metrics.ListenPort

	nc.Tracing.Implementation = c.Tracing.Implementation
	nc.Tracing.CollectorEndpoint = c.Tracing.CollectorEndpoint

	nc.Frontend.ListenAddress = c.Frontend.ListenAddress
	nc.Frontend.ListenPort = c.Frontend.ListenPort
	nc.Frontend.TLSListenAddress = c.Frontend.TLSListenAddress
	nc.Frontend.TLSListenPort = c.Frontend.TLSListenPort
	nc.Frontend.ConnectionsLimit = c.Frontend.ConnectionsLimit
	nc.Frontend.ServeTLS = c.Frontend.ServeTLS

	for k, v := range c.Origins {
		nc.Origins[k] = v.Copy()
	}

	for k, v := range c.Caches {
		nc.Caches[k] = v.Copy()
	}

	for k, v := range c.NegativeCacheConfigs {
		nc.NegativeCacheConfigs[k] = v.Copy()
	}

	return nc
}

func (c *TricksterConfig) String() string {
	cp := c.copy()

	// the toml library will panic if the Handler is assigned,
	// even though this field is annotated as skip ("-") in the prototype
	// so we'll iterate the paths and set to nil the Handler (in our local copy only)
	if cp.Origins != nil {
		for _, v := range cp.Origins {
			if v != nil {
				for _, w := range v.Paths {
					w.Handler = nil
					w.KeyHasher = nil
				}
			}
			// also strip out potentially sensitive headers
			hideAuthorizationCredentials(v.HealthCheckHeaders)

			if v.Paths != nil {
				for _, p := range v.Paths {
					hideAuthorizationCredentials(p.RequestHeaders)
					hideAuthorizationCredentials(p.ResponseHeaders)
				}
			}
		}
	}

	// strip Redis password
	for k, v := range cp.Caches {
		if v != nil && cp.Caches[k].Redis.Password != "" {
			cp.Caches[k].Redis.Password = "*****"
		}
	}

	var buf bytes.Buffer
	e := toml.NewEncoder(&buf)
	e.Encode(cp)
	return buf.String()
}

var sensitiveCredentials = map[string]bool{headers.NameAuthorization: true}

func hideAuthorizationCredentials(headers map[string]string) {
	if headers != nil {
		// strip Authorization Headers
		for k := range headers {
			if _, ok := sensitiveCredentials[k]; ok {
				headers[k] = "*****"
			}
		}
	}
}

// Copy returns an exact copy of an *OriginConfig
func (oc *OriginConfig) Copy() *OriginConfig {

	o := &OriginConfig{}
	o.BackfillTolerance = oc.BackfillTolerance
	o.BackfillToleranceSecs = oc.BackfillToleranceSecs
	o.CacheName = oc.CacheName
	o.CacheKeyPrefix = oc.CacheKeyPrefix
	o.FastForwardDisable = oc.FastForwardDisable
	o.FastForwardTTL = oc.FastForwardTTL
	o.FastForwardTTLSecs = oc.FastForwardTTLSecs
	o.HealthCheckUpstreamPath = oc.HealthCheckUpstreamPath
	o.HealthCheckVerb = oc.HealthCheckVerb
	o.HealthCheckQuery = oc.HealthCheckQuery
	o.Host = oc.Host
	o.Name = oc.Name
	o.IsDefault = oc.IsDefault
	o.KeepAliveTimeoutSecs = oc.KeepAliveTimeoutSecs
	o.MaxIdleConns = oc.MaxIdleConns
	o.MaxTTLSecs = oc.MaxTTLSecs
	o.MaxTTL = oc.MaxTTL
	o.MaxObjectSizeBytes = oc.MaxObjectSizeBytes
	o.OriginType = oc.OriginType
	o.OriginURL = oc.OriginURL
	o.PathPrefix = oc.PathPrefix
	o.RevalidationFactor = oc.RevalidationFactor
	o.Scheme = oc.Scheme
	o.Timeout = oc.Timeout
	o.TimeoutSecs = oc.TimeoutSecs
	o.TimeseriesRetention = oc.TimeseriesRetention
	o.TimeseriesRetentionFactor = oc.TimeseriesRetentionFactor
	o.TimeseriesEvictionMethodName = oc.TimeseriesEvictionMethodName
	o.TimeseriesEvictionMethod = oc.TimeseriesEvictionMethod
	o.TimeseriesTTL = oc.TimeseriesTTL
	o.TimeseriesTTLSecs = oc.TimeseriesTTLSecs
	o.ValueRetention = oc.ValueRetention

	o.HealthCheckHeaders = make(map[string]string)
	for k, v := range oc.HealthCheckHeaders {
		o.HealthCheckHeaders[k] = v
	}

	o.Paths = make(map[string]*PathConfig)
	for l, p := range oc.Paths {
		o.Paths[l] = p.Copy()
	}

	o.NegativeCacheName = oc.NegativeCacheName
	if oc.NegativeCache != nil {
		m := make(map[int]time.Duration)
		for c, t := range oc.NegativeCache {
			m[c] = t
		}
		o.NegativeCache = m
	}

	if oc.TLS != nil {
		o.TLS = oc.TLS.Copy()
	}
	o.RequireTLS = oc.RequireTLS

	if oc.FastForwardPath != nil {
		o.FastForwardPath = oc.FastForwardPath.Copy()
	}

	return o

}

// Copy returns an exact copy of a *CachingConfig
func (cc *CachingConfig) Copy() *CachingConfig {

	c := NewCacheConfig()
	c.Name = cc.Name
	c.CacheType = cc.CacheType
	c.CacheTypeID = cc.CacheTypeID
	c.Compression = cc.Compression

	c.Index.FlushInterval = cc.Index.FlushInterval
	c.Index.FlushIntervalSecs = cc.Index.FlushIntervalSecs
	c.Index.MaxSizeBackoffBytes = cc.Index.MaxSizeBackoffBytes
	c.Index.MaxSizeBackoffObjects = cc.Index.MaxSizeBackoffObjects
	c.Index.MaxSizeBytes = cc.Index.MaxSizeBytes
	c.Index.MaxSizeObjects = cc.Index.MaxSizeObjects
	c.Index.ReapInterval = cc.Index.ReapInterval
	c.Index.ReapIntervalSecs = cc.Index.ReapIntervalSecs

	c.Badger.Directory = cc.Badger.Directory
	c.Badger.ValueDirectory = cc.Badger.ValueDirectory

	c.Filesystem.CachePath = cc.Filesystem.CachePath

	c.BBolt.Bucket = cc.BBolt.Bucket
	c.BBolt.Filename = cc.BBolt.Filename

	c.Redis.ClientType = cc.Redis.ClientType
	c.Redis.DB = cc.Redis.DB
	c.Redis.DialTimeoutMS = cc.Redis.DialTimeoutMS
	c.Redis.Endpoint = cc.Redis.Endpoint
	c.Redis.Endpoints = cc.Redis.Endpoints
	c.Redis.IdleCheckFrequencyMS = cc.Redis.IdleCheckFrequencyMS
	c.Redis.IdleTimeoutMS = cc.Redis.IdleTimeoutMS
	c.Redis.MaxConnAgeMS = cc.Redis.MaxConnAgeMS
	c.Redis.MaxRetries = cc.Redis.MaxRetries
	c.Redis.MaxRetryBackoffMS = cc.Redis.MaxRetryBackoffMS
	c.Redis.MinIdleConns = cc.Redis.MinIdleConns
	c.Redis.MinRetryBackoffMS = cc.Redis.MinRetryBackoffMS
	c.Redis.Password = cc.Redis.Password
	c.Redis.PoolSize = cc.Redis.PoolSize
	c.Redis.PoolTimeoutMS = cc.Redis.PoolTimeoutMS
	c.Redis.Protocol = cc.Redis.Protocol
	c.Redis.ReadTimeoutMS = cc.Redis.ReadTimeoutMS
	c.Redis.SentinelMaster = cc.Redis.SentinelMaster
	c.Redis.WriteTimeoutMS = cc.Redis.WriteTimeoutMS

	return c

}
