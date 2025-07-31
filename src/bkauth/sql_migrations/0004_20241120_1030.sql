-- TencentBlueKing is pleased to support the open source community by making
-- 蓝鲸智云 - Auth服务(BlueKing - Auth) available.
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

-- add fields
ALTER TABLE `bkauth`.`app` ADD COLUMN tenant_mode VARCHAR(32) NULL;
ALTER TABLE `bkauth`.`app` ADD COLUMN tenant_id VARCHAR(32) NULL;

-- update legacy data
UPDATE `bkauth`.`app` SET tenant_mode = 'single', tenant_id = 'default' WHERE tenant_mode IS NULL;

-- update fields
ALTER TABLE `bkauth`.`app` MODIFY COLUMN tenant_mode VARCHAR(32) NOT NULL COMMENT 'global or single';
ALTER TABLE `bkauth`.`app` MODIFY COLUMN tenant_id VARCHAR(32) NOT NULL COMMENT 'empty or specific tenant_id';
