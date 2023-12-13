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

	"bkauth/pkg/api/common"
	"bkauth/pkg/util"
)

type scopeSerializer struct {
	ID          string `json:"id" binding:"required,min=3,max=16" example:"read"`
	Name        string `json:"name" binding:"required,max=32" example:"Read"`
	Description string `json:"description" binding:"omitempty" example:"Read"`
}

func (s *scopeSerializer) validate() error {
	if !common.ValidIDRegex.MatchString(s.ID) {
		return common.ErrInvalidID
	}
	return nil
}

// validateScopesRepeat :校验提交的数据里是否有重复ID和Name
func validateScopesRepeat(scopes []scopeSerializer) error {
	idSet := util.NewStringSet()
	nameSet := util.NewStringSet()
	for _, scope := range scopes {
		if idSet.Has(scope.ID) {
			return fmt.Errorf("scope id[%s] repeat", scope.ID)
		}
		if nameSet.Has(scope.Name) {
			return fmt.Errorf("scope name[%s] repeat", scope.Name)
		}

		idSet.Add(scope.ID)
		nameSet.Add(scope.Name)
	}
	return nil
}

type deleteViaID struct {
	ID string `json:"id" binding:"required" example:"read"`
}

type scopeAndTargetSerializer struct {
	common.TargetIDSerializer
	ScopeID string `uri:"scope_id" binding:"required,min=3,max=16" example:"read"`
}

type updateScopeSerializer struct {
	Name        string `json:"name" binding:"omitempty,max=32" example:"Read"`
	Description string `json:"description" binding:"omitempty" example:"Read"`
}

func (s *updateScopeSerializer) validate(keys map[string]interface{}) error {
	if _, ok := keys["name"]; ok {
		if s.Name == "" {
			return fmt.Errorf("name should not be empty")
		}
	}

	return nil
}
