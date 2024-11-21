/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - Auth 服务 (BlueKing - Auth) available.
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

const AppSVC = "AppSVC"

type AppService interface {
	Get(code string) (types.App, error)
	Exists(code string) (bool, error)
	NameExists(name string) (bool, error)
	Create(app types.App, createdSource string) error
	CreateWithSecret(app types.App, appSecret, createdSource string) error
	List() ([]types.App, error)
}

type appService struct {
	manager          dao.AppManager
	accessKeyManager dao.AccessKeyManager
}

func NewAppService() AppService {
	return &appService{
		manager:          dao.NewAppManager(),
		accessKeyManager: dao.NewAccessKeyManager(),
	}
}

func (s *appService) Get(code string) (app types.App, err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(AppSVC, "Get")

	daoApp, err := s.manager.Get(code)
	if err != nil {
		return app, errorWrapf(err, "manager.Get fail")
	}

	return types.App{
		Code:        daoApp.Code,
		Name:        daoApp.Name,
		Description: daoApp.Description,
		TenantType:  daoApp.TenantType,
		TenantID:    daoApp.TenantID,
	}, nil
}

func (s *appService) Exists(code string) (bool, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(AppSVC, "Exists")

	exists, err := s.manager.Exists(code)
	if err != nil {
		return false, errorWrapf(err, "manager.Exists code=`%s` fail", code)
	}
	return exists, nil
}

func (s *appService) NameExists(name string) (bool, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(AppSVC, "NameExists")

	exists, err := s.manager.NameExists(name)
	if err != nil {
		return false, errorWrapf(err, "manager.NameExists name=`%s` fail", name)
	}
	return exists, nil
}

// Create :创建应用，createdSource 为创建的来源，即哪个系统创建了该 APP
func (s *appService) Create(app types.App, createdSource string) (err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(AppSVC, "Create")

	// 使用事务
	tx, err := database.GenerateDefaultDBTx()

	defer database.RollBackWithLog(tx)

	if err != nil {
		return errorWrapf(err, "define tx fail")
	}

	// 创建应用
	daoApp := dao.App{
		Code:        app.Code,
		Name:        app.Name,
		Description: app.Description,
		TenantType:  app.TenantType,
		TenantID:    app.TenantID,
	}
	err = s.manager.CreateWithTx(tx, daoApp)
	if err != nil {
		return errorWrapf(err, "manager.CreateWithTx app=`%+v` fail", daoApp)
	}

	// 创建应用对应 Secret
	daoAccessKey := newDaoAccessKey(app.Code, createdSource)
	_, err = s.accessKeyManager.CreateWithTx(tx, daoAccessKey)
	if err != nil {
		return errorWrapf(err, "accessKeyManager.CreateWithTx secret=`%+v` fail", daoAccessKey)
	}

	err = tx.Commit()
	return
}

// CreateWithSecret :创建应用，但支持指定 appSecret 的值，createdSource 为创建的来源，即哪个系统创建了该 APP
func (s *appService) CreateWithSecret(app types.App, appSecret, createdSource string) (err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(AppSVC, "CreateWithSecret")

	// 使用事务
	tx, err := database.GenerateDefaultDBTx()
	defer database.RollBackWithLog(tx)

	if err != nil {
		return errorWrapf(err, "define tx fail")
	}

	// 创建应用
	daoApp := dao.App{
		Code:        app.Code,
		Name:        app.Name,
		Description: app.Description,
		TenantType:  app.TenantType,
		TenantID:    app.TenantID,
	}
	err = s.manager.CreateWithTx(tx, daoApp)
	if err != nil {
		return errorWrapf(err, "manager.CreateWithTx app=`%+v` fail", daoApp)
	}

	// 创建应用对应 Secret
	daoAccessKey := newDaoAccessKeyWithAppSecret(app.Code, appSecret, createdSource)
	_, err = s.accessKeyManager.CreateWithTx(tx, daoAccessKey)
	if err != nil {
		return errorWrapf(err, "accessKeyManager.CreateWithTx secret=`%+v` fail", daoAccessKey)
	}

	err = tx.Commit()

	return
}

func (s *appService) List() (apps []types.App, err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(AppSVC, "List")

	daoApps, err := s.manager.List()
	if err != nil {
		return apps, errorWrapf(err, "manager.List fail")
	}

	apps = make([]types.App, 0, len(daoApps))
	for _, daoApp := range daoApps {
		apps = append(apps, types.App{
			Code:        daoApp.Code,
			Name:        daoApp.Name,
			Description: daoApp.Description,
			TenantType:  daoApp.TenantType,
			TenantID:    daoApp.TenantID,
		})
	}

	return
}
