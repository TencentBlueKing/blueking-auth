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

type Config struct {
	Debug bool

	Server Server
	Sentry Sentry

	PprofPassword string

	Databases   []Database
	DatabaseMap map[string]Database

	Redis    []Redis
	RedisMap map[string]Redis

	Crypto Crypto

	AccessKeys map[string]string

	APIAllowLists []APIAllowList

	Logger Logger
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

	return &cfg, nil
}
