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
)

type createdTargetSerializer struct {
	ID          string `json:"id" binding:"required,min=3,max=16" example:"bk_ci"`
	Name        string `json:"name" binding:"required,max=32" example:"BK PaaS"`
	Description string `json:"description" binding:"omitempty" example:"Platform as A Service"`
	Clients     string `json:"clients" binding:"required" example:"bk_ci,bk_ci1,bk_ci2"`
}

func (s *createdTargetSerializer) validate() error {
	if !common.ValidIDRegex.MatchString(s.ID) {
		return common.ErrInvalidID
	}
	return nil
}

type updatedTargetSerializer struct {
	Name        string `json:"name" binding:"omitempty,max=32" example:"BK PaaS"`
	Description string `json:"description" binding:"omitempty" example:"Platform as A Service"`
	Clients     string `json:"clients" binding:"omitempty" example:"bk_ci,bk_ci1,bk_ci2"`
}

func (s *updatedTargetSerializer) validate(keys map[string]interface{}) error {
	if _, ok := keys["name"]; ok {
		if s.Name == "" {
			return fmt.Errorf("name should not be empty")
		}
	}

	if _, ok := keys["clients"]; ok {
		if s.Clients == "" {
			return fmt.Errorf("clients should not be empty")
		}
	}

	return nil
}

type targetCreateResponse struct {
	ID string `json:"id" example:"bk_ci"`
}
