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

-- Migrate every time column from TIMESTAMP to DATETIME(6).
--
-- Why: TIMESTAMP cannot store anything past 2038-01-19 03:14:07 UTC, which any
-- long-lived expires_at will eventually cross. MySQL has no plan to widen the
-- 4-byte storage format, so DATETIME is the only option on this engine.
-- DATETIME carries no time zone semantics, so the values are UTC by convention:
-- the application enforces it via loc=UTC on the connection, and the driver
-- converts every bound time.Time to UTC before serialising it.
--
-- The (6) precision is not cosmetic: MySQL rounds rather than truncates the
-- fractional part that does not fit the column, so a second-resolution column
-- stretches an expires_at by up to 500ms.
--
-- Operational note: changing a column type forces ALGORITHM=COPY, which rebuilds
-- the table and blocks concurrent DML. Check the row counts of
-- oauth_access_token / oauth_refresh_token first and use gh-ost or a maintenance
-- window if they are large.

-- Converting TIMESTAMP to DATETIME renders the stored UTC epoch as a wall-clock
-- literal using the session time zone. sql-migrate connects with the dbconfig
-- DSN, not the application DSN, so the session may well default to the server
-- time zone; running unpinned would shift every historical value by the zone
-- offset with no error at all.
SET time_zone = '+00:00';

-- One ALTER per table so that each table is rebuilt exactly once.
ALTER TABLE `bkauth`.`app`
    MODIFY COLUMN `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT 'UTC',
    MODIFY COLUMN `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
        ON UPDATE CURRENT_TIMESTAMP(6) COMMENT 'UTC';

ALTER TABLE `bkauth`.`access_key`
    MODIFY COLUMN `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT 'UTC',
    MODIFY COLUMN `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
        ON UPDATE CURRENT_TIMESTAMP(6) COMMENT 'UTC';

ALTER TABLE `bkauth`.`oauth_client`
    MODIFY COLUMN `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT 'UTC',
    MODIFY COLUMN `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
        ON UPDATE CURRENT_TIMESTAMP(6) COMMENT 'UTC';

ALTER TABLE `bkauth`.`oauth_authorization_code`
    MODIFY COLUMN `expires_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT 'UTC',
    MODIFY COLUMN `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT 'UTC';

ALTER TABLE `bkauth`.`oauth_access_token`
    MODIFY COLUMN `expires_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT 'UTC',
    MODIFY COLUMN `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT 'UTC',
    MODIFY COLUMN `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
        ON UPDATE CURRENT_TIMESTAMP(6) COMMENT 'UTC';

ALTER TABLE `bkauth`.`oauth_refresh_token`
    MODIFY COLUMN `expires_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT 'UTC',
    MODIFY COLUMN `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT 'UTC',
    MODIFY COLUMN `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
        ON UPDATE CURRENT_TIMESTAMP(6) COMMENT 'UTC';

ALTER TABLE `bkauth`.`oauth_device_code`
    MODIFY COLUMN `last_polled_at` DATETIME(6) NULL COMMENT 'UTC',
    MODIFY COLUMN `expires_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT 'UTC',
    MODIFY COLUMN `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT 'UTC',
    MODIFY COLUMN `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
        ON UPDATE CURRENT_TIMESTAMP(6) COMMENT 'UTC';
