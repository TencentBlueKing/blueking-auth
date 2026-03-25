-- TencentBlueKing is pleased to support the open source community by making
-- 蓝鲸智云 - Auth 服务 (BlueKing - Auth) available.
-- Copyright (C) 2017 THL A29 Limited, a Tencent company. All rights reserved.
-- Licensed under the MIT License (the "License"); you may not use this file except
-- in compliance with the License. You may obtain a copy of the License at
--     http://opensource.org/licenses/MIT
-- Unless required by applicable law or agreed to in writing, software distributed under
-- the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
-- either express or implied. See the License for the specific language governing permissions and
-- limitations under the License.
-- We undertake not to change the open source license (MIT license) applicable
-- to the current version of the project delivered to anyone in the future.

-- OAuth Client table
-- token_endpoint_auth_method is NOT stored; it is derived from type at runtime:
--   public -> "none", confidential -> "client_secret_basic"
CREATE TABLE IF NOT EXISTS `bkauth`.`oauth_client` (
    `id` VARCHAR(128) NOT NULL PRIMARY KEY,
    `name` VARCHAR(256) NOT NULL,
    `type` ENUM('public', 'confidential') NOT NULL DEFAULT 'public',
    `redirect_uris` JSON NOT NULL,
    `grant_types` VARCHAR(256) NOT NULL DEFAULT 'authorization_code,refresh_token',
    `logo_uri` VARCHAR(512) NOT NULL DEFAULT '',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- OAuth Authorization Code table
CREATE TABLE IF NOT EXISTS `bkauth`.`oauth_authorization_code` (
    `code` VARCHAR(128) NOT NULL PRIMARY KEY,
    `client_id` VARCHAR(128) NOT NULL,
    `realm_name` VARCHAR(64) NOT NULL DEFAULT '',
    `tenant_id` VARCHAR(32) NOT NULL DEFAULT '',
    `sub` VARCHAR(64) NOT NULL,
    `username` VARCHAR(64) NOT NULL,
    `redirect_uri` VARCHAR(512) NOT NULL,
    `scope` VARCHAR(256) NOT NULL DEFAULT '',
    `audience` JSON NOT NULL,
    `code_challenge` VARCHAR(128) NOT NULL DEFAULT '',
    `code_challenge_method` VARCHAR(16) NOT NULL DEFAULT '',
    `expires_at` TIMESTAMP NOT NULL DEFAULT '1970-01-01 00:00:01',
    `used` TINYINT(1) NOT NULL DEFAULT 0,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- OAuth Access Token table
CREATE TABLE IF NOT EXISTS `bkauth`.`oauth_access_token` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `jti` VARCHAR(64) NOT NULL UNIQUE,
    `token_hash` VARCHAR(64) NOT NULL UNIQUE,
    `token_mask` VARCHAR(32) NOT NULL DEFAULT '',
    `grant_id` VARCHAR(64) NOT NULL,
    `client_id` VARCHAR(128) NOT NULL,
    `realm_name` VARCHAR(64) NOT NULL DEFAULT '',
    `tenant_id` VARCHAR(32) NOT NULL DEFAULT '',
    `sub` VARCHAR(64) NOT NULL DEFAULT '',
    `username` VARCHAR(64) NOT NULL DEFAULT '',
    `audience` JSON NOT NULL,
    `scope` VARCHAR(256) NOT NULL DEFAULT '',
    `expires_at` TIMESTAMP NOT NULL DEFAULT '1970-01-01 00:00:01',
    `revoked` TINYINT(1) NOT NULL DEFAULT 0,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX `idx_grant_id` (`grant_id`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- OAuth Refresh Token table
CREATE TABLE IF NOT EXISTS `bkauth`.`oauth_refresh_token` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `token_hash` VARCHAR(64) NOT NULL UNIQUE,
    `token_mask` VARCHAR(32) NOT NULL DEFAULT '',
    `grant_id` VARCHAR(64) NOT NULL,
    `access_token_id` BIGINT UNSIGNED NOT NULL,
    `client_id` VARCHAR(128) NOT NULL,
    `realm_name` VARCHAR(64) NOT NULL DEFAULT '',
    `tenant_id` VARCHAR(32) NOT NULL DEFAULT '',
    `sub` VARCHAR(64) NOT NULL DEFAULT '',
    `username` VARCHAR(64) NOT NULL DEFAULT '',
    `audience` JSON NOT NULL,
    `scope` VARCHAR(256) NOT NULL DEFAULT '',
    `expires_at` TIMESTAMP NOT NULL DEFAULT '1970-01-01 00:00:01',
    `revoked` TINYINT(1) NOT NULL DEFAULT 0,
    `rotation_count` INT NOT NULL DEFAULT 0,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX `idx_grant_id` (`grant_id`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- OAuth 2.0 Device Authorization Grant (RFC 8628)
CREATE TABLE IF NOT EXISTS `bkauth`.`oauth_device_code` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `device_code` VARCHAR(128) NOT NULL UNIQUE,
    `user_code` VARCHAR(16) NOT NULL UNIQUE,
    `client_id` VARCHAR(128) NOT NULL,
    `scope` VARCHAR(256) NOT NULL DEFAULT '',
    `resource` VARCHAR(2048) NOT NULL DEFAULT '',
    `realm_name` VARCHAR(64) NOT NULL DEFAULT 'blueking',
    `audience` JSON NULL,
    `status` ENUM('pending', 'approved', 'denied', 'consumed') NOT NULL DEFAULT 'pending',
    `tenant_id` VARCHAR(32) NOT NULL DEFAULT '',
    `sub` VARCHAR(64) NOT NULL DEFAULT '',
    `username` VARCHAR(64) NOT NULL DEFAULT '',
    `poll_interval` INT NOT NULL DEFAULT 5,
    `last_polled_at` TIMESTAMP NULL,
    `expires_at` TIMESTAMP NOT NULL DEFAULT '1970-01-01 00:00:01',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
