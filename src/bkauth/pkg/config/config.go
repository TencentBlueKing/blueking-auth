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

package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
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
	// ID names the instance. The service uses the entry with id "default".
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

	// The fields below are used when SentinelEnabled is true. MasterName is required in that mode.
	SentinelEnabled  bool
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
	// Nonce is legacy-only: it decrypts secrets stored before nonces were randomized.
	// New ciphertexts embed their own random nonce and never read this value.
	Nonce string
	Key   string
}

// KMSConfig controls loading credentials from a KMS envelope instead of the config file.
// Both files are mounted from a Secret; the config only carries their paths.
type KMSConfig struct {
	Enabled bool
	// PrivateKeyFile is the path of the base64-encoded PEM private key.
	// Required when Enabled is true; there is no default path.
	PrivateKeyFile string
	// EnvelopeFile is the path of the envelope.
	// Required when Enabled is true; there is no default path.
	EnvelopeFile string
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
	KMS    KMSConfig

	APIAllowLists []APIAllowList

	Logger Logger

	Trace     TraceConfig
	Profiling ProfilingConfig

	AppCode   string
	AppSecret string
	// BKAuthURL is the external base URL of the BKAuth service
	// (e.g., https://bkauth.example.com). Used to construct OAuth issuer,
	// well-known endpoints, and frontend redirect URLs.
	BKAuthURL            string
	BKApiURLTmpl         string
	BKLoginURL           string
	BKLoginTokenName     string
	BKLoginAPIViaGateway bool
	// CSRFTrustedOrigins additionally accepts these origins ("scheme://host[:port]")
	// on the Cookie-authenticated /api/v1/web endpoints, for deployments where the
	// frontend is served from somewhere other than BKAuthURL. The origin of
	// BKAuthURL is always trusted and does not need to be repeated here.
	CSRFTrustedOrigins []string

	OAuth         OAuth
	PersonalToken PersonalToken
}

// LoadFile reads the config file at path with a viper instance of its own.
func LoadFile(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}
	return Load(v)
}

// Load 从 viper 中读取配置文件
func Load(v *viper.Viper) (*Config, error) {
	var cfg Config
	// 将配置信息绑定到结构体上
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Must run before the lists below are copied into maps, or the maps keep the file values.
	if err := cfg.applyKMS(); err != nil {
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

	// 3. OAuth
	if err := cfg.OAuth.load(); err != nil {
		return nil, err
	}

	// 4. PersonalToken
	cfg.PersonalToken.load()

	return &cfg, nil
}
