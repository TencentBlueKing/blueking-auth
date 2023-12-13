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
	"strings"

	"bkauth/pkg/database/dao"
	"bkauth/pkg/errorx"
	"bkauth/pkg/service/types"
)

const OAuthAppSvc = "OAuthAppSVC"

type OAuthAppService interface {
	Exists(appCode string) (bool, error)
	Get(appCode string) (types.OAuthApp, error)

	Create(oauthApp types.OAuthApp) error
	Update(oauthApp types.OAuthApp) error
}

type oauthAppService struct {
	manager dao.OAuthAppManager
}

func NewOAuthAppService() OAuthAppService {
	return &oauthAppService{
		manager: dao.NewOAuthAppManager(),
	}
}

func (s *oauthAppService) Exists(appCode string) (bool, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthAppSvc, "Exists")

	exists, err := s.manager.Exists(appCode)
	if err != nil {
		return false, errorWrapf(err, "manager.Exists appCode=`%s` fail", appCode)
	}
	return exists, nil
}

func (s *oauthAppService) Get(appCode string) (oauthApp types.OAuthApp, err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthAppSvc, "Get")

	daoOauthApp, err := s.manager.Get(appCode)
	if err != nil {
		return oauthApp, errorWrapf(err, "manager.Get appCode=`%s` fail", appCode)
	}

	oauthApp = types.OAuthApp{
		AppCode:      daoOauthApp.AppCode,
		RedirectURLs: strings.Split(daoOauthApp.RedirectURLs, ","),
	}
	return
}

func (s *oauthAppService) Create(oauthApp types.OAuthApp) (err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthAppSvc, "Create")

	daoOAuthApp := dao.OAuthApp{
		AppCode:      oauthApp.AppCode,
		RedirectURLs: strings.Join(oauthApp.RedirectURLs, ","),
	}

	err = s.manager.Create(daoOAuthApp)
	if err != nil {
		return errorWrapf(err, "manager.Create oauthApp=`%+v` fail", daoOAuthApp)
	}

	return
}

func (s *oauthAppService) Update(oauthApp types.OAuthApp) (err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthAppSvc, "Update")

	daoOAuthApp := dao.OAuthApp{
		RedirectURLs: strings.Join(oauthApp.RedirectURLs, ","),
	}

	err = s.manager.Update(oauthApp.AppCode, daoOAuthApp)
	if err != nil {
		return errorWrapf(err, "manager.Update appCode=`%s` oauthApp=`%+v` fail", oauthApp.AppCode, daoOAuthApp)
	}

	return
}
