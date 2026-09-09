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

-- Move oauth_device_code.resource from VARCHAR(2048) to JSON, the column now
-- holding the resource indicators of one device authorization as a JSON array.

-- A row written before the column held an array holds one bare indicator, which is
-- not valid JSON and would fail the ALTER outright. Clearing the table first is
-- what lets this migration run unattended.
--
-- Nothing durable is lost. A row here is a device authorization in flight and
-- nothing else: it lives 600 seconds (DeviceCodeTTL), and the tokens it leads to
-- are rows of their own, tied to a grant_id rather than back to this table. So a
-- consumed row is already spent, an expired one is already unusable, and the only
-- row a user can notice is a pending one -- whose loss is the device flow being
-- restarted, the same thing that happens when the ten minutes run out.
DELETE FROM `bkauth`.`oauth_device_code`;

ALTER TABLE `bkauth`.`oauth_device_code` MODIFY COLUMN `resource` JSON NOT NULL;
