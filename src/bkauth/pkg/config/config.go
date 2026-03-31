/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - Auth 服务 (BlueKing - Auth) available.
 * Copyright (C) 2017 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *     http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

// Package config defines application configuration structures and loaders.
package config

import (
	"errors"

	"github.com/spf13/viper"
)

const (
	defaultAccessTokenTTL  int64 = 7200    // 2 hours
	defaultRefreshTokenTTL int64 = 2592000 // 30 days
)

// Server ...
type Server struct {
	Host string
	Port int

	GraceTimeout int64

	ReadTimeout  int
	WriteTimeout int
	IdleTimeout  int
}

// LogConfig ...
type LogConfig struct {
	Level    string
	Encoding string
	Writer   string
	Settings map[string]string
	// 日志脱敏
	Desensitization DesensitizationConfig
}

type DesensitizationConfig struct {
	// 脱敏日志开关
	Enabled bool
	// 敏感字段列表
	Fields []DesensitizationFiled
}

// DesensitizationFiled ...
type DesensitizationFiled struct {
	// 敏感字段所属的 filed key
	Key string
	// 敏感字段 JsonPath
	JsonPath []string
}

// Logger ...
type Logger struct {
	System LogConfig
	API    LogConfig
	SQL    LogConfig
	Audit  LogConfig
	Web    LogConfig
}

// TLS ...
type TLS struct {
	Enabled     bool
	CertCaFile  string
	CertFile    string
	CertKeyFile string
	// for testing only, default false is secure;
	// if set true will skip hostname verification, don't enable it in production
	InsecureSkipVerify bool
}

// Database ...
type Database struct {
	ID       string
	Host     string
	Port     int
	User     string
	Password string
	Name     string

	MaxOpenConns          int
	MaxIdleConns          int
	ConnMaxLifetimeSecond int

	// tls
	TLS TLS
}

// Redis ...
type Redis struct {
	ID           string
	Addr         string
	Password     string
	DB           int
	DialTimeout  int
	ReadTimeout  int
	WriteTimeout int
	PoolSize     int
	MinIdleConns int
	ChannelKey   string

	// mode=sentinel required
	SentinelAddr     string
	MasterName       string
	SentinelPassword string

	// tls
	TLS TLS
}

// Sentry ...
type Sentry struct {
	Enable bool
	DSN    string
}

type Crypto struct {
	Nonce string
	Key   string
}

type APIAllowList struct {
	API       string
	AllowList string
}

type OTLPEndpoint struct {
	Host  string
	Port  int
	Token string
	Type  string
}

type TraceConfig struct {
	Enabled     bool
	OTLP        OTLPEndpoint
	ServiceName string
	Sampler     string
}

type PyroscopeEndpoint struct {
	Host  string
	Port  int
	Type  string
	Token string
	Path  string
}

type ProfilingConfig struct {
	Enabled        bool
	Pyroscope      PyroscopeEndpoint
	ServiceName    string
	UploadInterval string
}

// TokenTTLOverride allows overriding the default AccessToken/RefreshToken TTL
// for a specific (RealmName, ClientID) combination.
// ClientID can be "*" to match all clients within a realm.
type TokenTTLOverride struct {
	RealmName       string
	ClientID        string
	AccessTokenTTL  int64
	RefreshTokenTTL int64
}

// tokenTTLKey is the lookup key for pre-computed TTL override map.
type tokenTTLKey struct {
	RealmName string
	ClientID  string
}

// ConfidentialClientSecretExemption exempts a confidential client from
// client_secret verification on the specified realm (exact match only).
type ConfidentialClientSecretExemption struct {
	RealmName string
	ClientID  string
}

// IntrospectAllowedAppCode grants an AppCode access to the introspect
// endpoint for a specific realm (exact match only).
type IntrospectAllowedAppCode struct {
	RealmName string
	AppCode   string
}

// OAuth holds OAuth 2.0 protocol-specific configuration.
type OAuth struct {
	// AccessTokenTTL is the lifetime of access token in seconds (default: 7200)
	AccessTokenTTL int64
	// RefreshTokenTTL is the lifetime of refresh token in seconds (default: 2592000)
	RefreshTokenTTL int64
	// DCREnabled indicates whether Dynamic Client Registration is enabled
	DCREnabled bool
	// DefaultRealmName is used for backward-compatible endpoints that don't specify a realm.
	DefaultRealmName string
	// IntrospectAllowedAppCodes controls which AppCodes may call the introspect
	// endpoint, on a per-realm basis (exact match only).
	// If empty, all requests are denied.
	IntrospectAllowedAppCodes []IntrospectAllowedAppCode
	// ConfidentialClientSecretExemptions exempts specific confidential clients
	// from client_secret verification on a per-(Realm, ClientID) basis (exact match only).
	// Default: empty (all confidential clients must provide client_secret).
	ConfidentialClientSecretExemptions []ConfidentialClientSecretExemption
	// TokenTTLOverrides allows per-(realm, clientID) TTL configuration.
	// Lookup priority: exact (realm, clientID) > realm wildcard (realm, "*") > global default.
	TokenTTLOverrides []TokenTTLOverride

	// tokenTTLMap is pre-computed in Load() for O(1) lookups.
	tokenTTLMap map[tokenTTLKey]*TokenTTLOverride
	// secretExemptMap is pre-computed in Load() for O(1) lookups.
	secretExemptMap map[ConfidentialClientSecretExemption]struct{}
	// introspectAllowedMap is pre-computed in Load() for O(1) lookups.
	introspectAllowedMap map[IntrospectAllowedAppCode]struct{}
}

// ResolveTokenTTL returns the effective (accessTokenTTL, refreshTokenTTL) for the
// given realm and clientID. Lookup priority:
//  1. Exact match: (realmName, clientID)
//  2. Realm wildcard: (realmName, "*")
//  3. Global defaults: OAuth.AccessTokenTTL / OAuth.RefreshTokenTTL
//
// Within each level, only non-zero override values replace the inherited value.
func (o *OAuth) ResolveTokenTTL(realmName, clientID string) (accessTTL, refreshTTL int64) {
	accessTTL = o.AccessTokenTTL
	refreshTTL = o.RefreshTokenTTL

	if o.tokenTTLMap == nil {
		return accessTTL, refreshTTL
	}

	if ov, ok := o.tokenTTLMap[tokenTTLKey{RealmName: realmName, ClientID: "*"}]; ok {
		if ov.AccessTokenTTL > 0 {
			accessTTL = ov.AccessTokenTTL
		}
		if ov.RefreshTokenTTL > 0 {
			refreshTTL = ov.RefreshTokenTTL
		}
	}

	if ov, ok := o.tokenTTLMap[tokenTTLKey{RealmName: realmName, ClientID: clientID}]; ok {
		if ov.AccessTokenTTL > 0 {
			accessTTL = ov.AccessTokenTTL
		}
		if ov.RefreshTokenTTL > 0 {
			refreshTTL = ov.RefreshTokenTTL
		}
	}

	return accessTTL, refreshTTL
}

// IsIntrospectAllowed reports whether the given appCode is allowed to call
// the introspect endpoint for the specified realm.
// Returns false when no entries are configured (deny by default).
func (o *OAuth) IsIntrospectAllowed(realmName, appCode string) bool {
	_, ok := o.introspectAllowedMap[IntrospectAllowedAppCode{RealmName: realmName, AppCode: appCode}]
	return ok
}

// IsClientSecretExempt reports whether the given (realmName, clientID) is exempt
// from client_secret verification (exact match only).
func (o *OAuth) IsClientSecretExempt(realmName, clientID string) bool {
	_, ok := o.secretExemptMap[ConfidentialClientSecretExemption{RealmName: realmName, ClientID: clientID}]
	return ok
}

type Config struct {
	Debug bool
	// 是否开启多租户模式
	EnableMultiTenantMode bool

	Server Server
	Sentry Sentry

	PprofPassword   string
	MonitoringToken string

	Databases   []Database
	DatabaseMap map[string]Database

	Redis    []Redis
	RedisMap map[string]Redis

	Crypto Crypto

	AccessKeys map[string]string

	APIAllowLists []APIAllowList

	Logger Logger

	Trace     TraceConfig
	Profiling ProfilingConfig

	// BKAuthURL is the external base URL of the BKAuth service
	// (e.g., https://bkauth.example.com). Used to construct OAuth issuer,
	// well-known endpoints, and frontend redirect URLs.
	BKAuthURL string

	AppCode              string
	AppSecret            string
	BKApiURLTmpl         string
	BKLoginURL           string
	BKLoginTokenName     string
	BKLoginAPIViaGateway bool

	OAuth OAuth
}

// Load 从 viper 中读取配置文件
func Load(v *viper.Viper) (*Config, error) {
	var cfg Config
	// 将配置信息绑定到结构体上
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// parse the list to map
	// 1. database
	cfg.DatabaseMap = make(map[string]Database)
	for _, db := range cfg.Databases {
		cfg.DatabaseMap[db.ID] = db
	}

	if len(cfg.DatabaseMap) == 0 {
		return nil, errors.New("database cannot be empty")
	}

	// 2. redis
	cfg.RedisMap = make(map[string]Redis)
	for _, rds := range cfg.Redis {
		cfg.RedisMap[rds.ID] = rds
	}

	// 3. OAuth defaults
	if cfg.OAuth.AccessTokenTTL == 0 {
		cfg.OAuth.AccessTokenTTL = defaultAccessTokenTTL
	}
	if cfg.OAuth.RefreshTokenTTL == 0 {
		cfg.OAuth.RefreshTokenTTL = defaultRefreshTokenTTL
	}
	// 5. Build token TTL override map for O(1) lookups
	cfg.OAuth.tokenTTLMap = make(map[tokenTTLKey]*TokenTTLOverride, len(cfg.OAuth.TokenTTLOverrides))
	for i := range cfg.OAuth.TokenTTLOverrides {
		ov := &cfg.OAuth.TokenTTLOverrides[i]
		cfg.OAuth.tokenTTLMap[tokenTTLKey{RealmName: ov.RealmName, ClientID: ov.ClientID}] = ov
	}

	// 6. Build secret exemption map for O(1) lookups
	exemptions := cfg.OAuth.ConfidentialClientSecretExemptions
	cfg.OAuth.secretExemptMap = make(map[ConfidentialClientSecretExemption]struct{}, len(exemptions))
	for _, ex := range exemptions {
		cfg.OAuth.secretExemptMap[ex] = struct{}{}
	}

	// 7. Build introspect allowed map for O(1) lookups
	cfg.OAuth.introspectAllowedMap = make(
		map[IntrospectAllowedAppCode]struct{}, len(cfg.OAuth.IntrospectAllowedAppCodes),
	)
	for _, entry := range cfg.OAuth.IntrospectAllowedAppCodes {
		cfg.OAuth.introspectAllowedMap[entry] = struct{}{}
	}

	return &cfg, nil
}
