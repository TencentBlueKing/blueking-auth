/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - Auth服务(BlueKing - Auth) available.
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

package handler

import (
	"fmt"

	"bkauth/pkg/service"
)

// checkTargetCreateUnique :检查Target的唯一性，这里主要是检测target_id是否唯一
func checkTargetCreateUnique(id string) error {
	svc := service.NewTargetService()

	// check target id exists
	exists, err := svc.Exists(id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("target(%s) already exists", id)
	}

	return nil
}
