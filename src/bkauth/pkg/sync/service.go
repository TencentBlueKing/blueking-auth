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

package sync

import (
	"bkauth/pkg/errorx"
)

const (
	OpenPaaSAccessKeySVC = "OpenPaaSAccessKeySVC"
)

type OpenPaaSAccessKey struct {
	AppCode   string
	AppSecret string
}

type OpenPaaSService interface {
	List() ([]OpenPaaSAccessKey, error)
	Create(appCode, appSecret string) error
}

type openPaaSService struct {
	manager OpenPaaSManager
}

func NewOpenPaaSService() OpenPaaSService {
	return &openPaaSService{
		manager: NewOpenPaaSManager(),
	}
}

func (s *openPaaSService) List() (openPaaSAccessKeys []OpenPaaSAccessKey, err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OpenPaaSAccessKeySVC, "List")
	apps, err := s.manager.ListBKPaaSApp()
	if err != nil {
		return openPaaSAccessKeys, errorWrapf(err, "manager.ListBKPaaSApp fail")
	}

	esbAccounts, err := s.manager.ListESBAppAccount()
	if err != nil {
		return openPaaSAccessKeys, errorWrapf(err, "manager.ListESBAppAccount fail")
	}

	openPaaSAccessKeys = make([]OpenPaaSAccessKey, 0, len(apps)+len(esbAccounts))

	for _, app := range apps {
		openPaaSAccessKeys = append(openPaaSAccessKeys, OpenPaaSAccessKey{
			AppCode:   app.Code,
			AppSecret: app.AuthToken,
		})
	}

	for _, esbAccount := range esbAccounts {
		openPaaSAccessKeys = append(openPaaSAccessKeys, OpenPaaSAccessKey{
			AppCode:   esbAccount.AppCode,
			AppSecret: esbAccount.AppToken,
		})
	}

	return
}

func (s *openPaaSService) Create(appCode, appSecret string) (err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OpenPaaSAccessKeySVC, "Create")

	// 1. 判断是否在paas_app表里，存在auth_token为NULL 或者 空字符串的，若存在，则更新
	exists, err := s.manager.AuthTokenEmptyExists(appCode)
	if err != nil {
		return errorWrapf(err, "manager.AuthTokenEmptyExists appCode=`%s` fail", appCode)
	}
	// paas_app里存在，但是auth_token为空，则更新
	if exists {
		err = s.manager.UpdateBKPaaSApp(appCode, appSecret)
		if err != nil {
			return errorWrapf(err, "manager.UpdateBKPaaSApp appCode=`%s` fail", appCode)
		}
		return

	}

	// 2. 直接添加到esb_app_account表里
	err = s.manager.CreateESBAppAccount(appCode, appSecret)
	if err != nil {
		return errorWrapf(err, "manager.CreateESBAppAccount appCode=`%s` fail", appCode)
	}
	return
}
