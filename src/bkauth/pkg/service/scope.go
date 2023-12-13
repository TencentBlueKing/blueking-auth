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

import (
	"bkauth/pkg/database"
	"bkauth/pkg/database/dao"
	"bkauth/pkg/errorx"
	"bkauth/pkg/service/types"
)

//go:generate mockgen -source=$GOFILE -destination=./mock/$GOFILE -package=mock

const ScopeSvc = "ScopeSvc"

type ScopeService interface {
	ListByTarget(targetID string) ([]types.Scope, error)
	BulkCreate(targetID string, scopes []types.Scope) error
	BulkDelete(targetID string, ids []string) error
	Update(targetID string, scope types.Scope) error
}

type scopeService struct {
	manager dao.ScopeManager
}

func NewScopeService() ScopeService {
	return &scopeService{
		manager: dao.NewScopeManager(),
	}
}

func (s *scopeService) ListByTarget(targetID string) (scopes []types.Scope, err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(ScopeSvc, "ListByTarget")

	daoScopes, err := s.manager.ListByTargetID(targetID)
	if err != nil {
		return scopes, errorWrapf(err, "manager.ListByTargetID targetID=`%s` fail", targetID)
	}

	scopes = make([]types.Scope, 0, len(daoScopes))
	for _, daoScope := range daoScopes {
		scopes = append(scopes, types.Scope{
			ID:          daoScope.ID,
			Name:        daoScope.Name,
			Description: daoScope.Description,
		})
	}

	return
}

func (s *scopeService) BulkCreate(targetID string, scopes []types.Scope) error {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(ScopeSvc, "BulkCreate")

	daoScopes := make([]dao.Scope, 0, len(scopes))
	for _, scope := range scopes {
		daoScopes = append(daoScopes, dao.Scope{
			TargetID:    targetID,
			ID:          scope.ID,
			Name:        scope.Name,
			Description: scope.Description,
		})
	}

	err := s.manager.BulkCreate(daoScopes)
	if err != nil {
		return errorWrapf(err, "manager.BulkCreate targetID=`%s` scopes=`%+v` fail", targetID, daoScopes)
	}

	return nil
}

func (s *scopeService) BulkDelete(targetID string, ids []string) error {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(ScopeSvc, "BulkDelete")

	err := s.manager.BulkDelete(targetID, ids)
	if err != nil {
		return errorWrapf(err, "manager.BulkDelete targetID=`%s` ids=`%+v` fail", targetID, ids)
	}

	return nil
}

func (s *scopeService) Update(targetID string, scope types.Scope) (err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(ScopeSvc, "Update")

	allowBlank := database.NewAllowBlankFields()
	if scope.AllowEmptyFields.HasKey("Description") {
		allowBlank.AddKey("Description")
	}

	daoScope := dao.Scope{
		Name:        scope.Name,
		Description: scope.Description,

		AllowBlankFields: allowBlank,
	}

	err = s.manager.Update(targetID, scope.ID, daoScope)
	if err != nil {
		return errorWrapf(
			err,
			"manager.Update target_id=`%s`, scope_id=`%s`, scope=`%+v` fail",
			targetID,
			scope.ID,
			daoScope,
		)
	}

	return
}
