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

-- Personal Access Token (PAT).
--
-- A separate table from oauth_access_token on purpose: oauth_access_token's core
-- invariant is "frozen at issuance" (audience / expires_at never change), which is
-- what lets its cache stay valid for minutes. A PAT is the opposite — a mutable
-- management object whose audience is editable and whose expires_at can be renewed,
-- all while the plaintext stays fixed. The two invariants cannot share one table.
--
-- Every time column is DATETIME(6) COMMENT 'UTC' per AGENTS.md 3.7; TIMESTAMP is
-- banned because a 365-day expires_at will cross the 2038 boundary. expires_at has
-- no DEFAULT so that a missing value errors out instead of silently writing an
-- already-expired row.
CREATE TABLE IF NOT EXISTS `bkauth`.`personal_access_token` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `token_hash` VARCHAR(64) NOT NULL UNIQUE,
    `token_mask` VARCHAR(32) NOT NULL DEFAULT '',
    `realm_name` VARCHAR(64) NOT NULL DEFAULT '',
    `tenant_id` VARCHAR(32) NOT NULL DEFAULT '',
    `sub` VARCHAR(64) NOT NULL DEFAULT '',
    `username` VARCHAR(64) NOT NULL DEFAULT '',
    `name` VARCHAR(64) NOT NULL,
    `description` VARCHAR(255) NOT NULL DEFAULT '',
    `audience` JSON NOT NULL,
    `expires_at` DATETIME(6) NOT NULL COMMENT 'UTC',
    `revoked` TINYINT(1) NOT NULL DEFAULT 0,
    `revoked_at` DATETIME(6) NULL DEFAULT NULL COMMENT 'UTC',
    -- Written on the cache-miss path only, and always together with
    -- `updated_at` = `updated_at` so that `updated_at` keeps meaning
    -- "last edited by the owner" rather than "last used". Wiring is deferred.
    `last_used_at` DATETIME(6) NULL DEFAULT NULL COMMENT 'UTC',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT 'UTC',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
        ON UPDATE CURRENT_TIMESTAMP(6) COMMENT 'UTC',
    -- idx_owner's trailing `id` gives list pagination a stable sort without a
    -- filesort. idx_expires_at / idx_revoked_at each serve one of the two purge
    -- queries (their timelines differ); both are unused this iteration but adding
    -- them to a large table later costs far more than declaring them at create time.
    INDEX `idx_owner` (`realm_name`, `sub`, `id`),
    INDEX `idx_expires_at` (`expires_at`),
    INDEX `idx_revoked_at` (`revoked_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
