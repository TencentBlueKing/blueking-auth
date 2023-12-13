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

package service

//go:generate mockgen -source=$GOFILE -destination=./mock/$GOFILE -package=mock

import (
	"bkauth/pkg/database"
	"bkauth/pkg/database/dao"
	"bkauth/pkg/errorx"
	"bkauth/pkg/service/types"
)

const TargetSvc = "TargetSvc"

type TargetService interface {
	Exists(id string) (bool, error)
	Get(id string) (types.Target, error)

	Create(target types.Target) error
	Update(target types.Target) error
}

type targetService struct {
	manager dao.TargetManager
}

func NewTargetService() TargetService {
	return &targetService{
		manager: dao.NewTargetManager(),
	}
}

func (s *targetService) Exists(id string) (bool, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(TargetSvc, "Exists")

	exists, err := s.manager.Exists(id)
	if err != nil {
		return false, errorWrapf(err, "manager.Exists id=`%s` fail", id)
	}
	return exists, nil
}

func (s *targetService) Get(id string) (target types.Target, err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(TargetSvc, "Get")

	daoTarget, err := s.manager.Get(id)
	if err != nil {
		return target, errorWrapf(err, "manager.Get id=`%s` fail", id)
	}

	target = types.Target{
		ID:          daoTarget.ID,
		Name:        daoTarget.Name,
		Description: daoTarget.Description,
		Clients:     daoTarget.Clients,
	}
	return
}

func (s *targetService) Create(target types.Target) (err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(TargetSvc, "Create")

	daoTarget := dao.Target{
		ID:          target.ID,
		Name:        target.Name,
		Description: target.Description,
		Clients:     target.Clients,
	}

	err = s.manager.Create(daoTarget)
	if err != nil {
		return errorWrapf(err, "manager.Create target=`%+v` fail", daoTarget)
	}

	return
}

func (s *targetService) Update(target types.Target) (err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(TargetSvc, "Update")

	allowBlank := database.NewAllowBlankFields()
	if target.AllowEmptyFields.HasKey("Description") {
		allowBlank.AddKey("Description")
	}

	daoTarget := dao.Target{
		Name:        target.Name,
		Description: target.Description,
		Clients:     target.Clients,

		AllowBlankFields: allowBlank,
	}

	err = s.manager.Update(target.ID, daoTarget)
	if err != nil {
		return errorWrapf(err, "manager.Update id=`%s`, target=`%+v` fail", target.ID, daoTarget)
	}

	return
}
