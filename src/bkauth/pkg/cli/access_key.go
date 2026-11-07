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

package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bkauth/pkg/config"
	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
)

// AccessKeyResult tells what CreateAccessKey did.
type AccessKeyResult string

const (
	AccessKeyAppCreated  AccessKeyResult = "app_created"
	AccessKeySecretAdded AccessKeyResult = "secret_added"
	AccessKeyUnchanged   AccessKeyResult = "unchanged"

	accessKeyCreatedSource = "deploy_init"
)

// CreateAccessKey makes appSecret a secret of appCode, creating the app if it does not
// exist. Safe to re-run: a secret the app already has is left alone. Adding a secret
// fails once the app holds service.MaxSecretsPreApp of them.
func CreateAccessKey(cfg *config.Config, appCode, appSecret string) (AccessKeyResult, error) {
	if appCode == "" || appSecret == "" {
		return "", errors.New("app code and app secret must not be empty")
	}

	ctx := context.Background()
	appSvc := service.NewAppService()
	exists, err := appSvc.Exists(ctx, appCode)
	if err != nil {
		return "", fmt.Errorf("check app %s exists: %w", appCode, err)
	}
	if !exists {
		tenantMode, tenantID := initTenant(cfg)
		app := types.App{
			Code:        appCode,
			Name:        appCode,
			Description: appCode,
			TenantMode:  tenantMode,
			TenantID:    tenantID,
		}
		if err := appSvc.CreateWithSecret(ctx, app, appSecret, accessKeyCreatedSource); err != nil {
			return "", fmt.Errorf("create app %s: %w", appCode, err)
		}
		return AccessKeyAppCreated, nil
	}

	svc := service.NewAccessKeyService()
	exists, err = svc.Verify(ctx, appCode, appSecret)
	if err != nil {
		return "", fmt.Errorf("verify secret of app %s: %w", appCode, err)
	}
	if exists {
		return AccessKeyUnchanged, nil
	}

	err = svc.CreateWithSecret(ctx, appCode, appSecret, accessKeyCreatedSource, "initialized during shell deployment")
	if err != nil {
		return "", fmt.Errorf("add secret to app %s: %w", appCode, err)
	}
	return AccessKeySecretAdded, nil
}

// ListAccessKey prints the access keys of appCodes, secrets in plaintext.
func ListAccessKey(appCodes []string) error {
	if len(appCodes) == 0 {
		return errors.New("app code must not be empty")
	}

	ctx := context.Background()
	svc := service.NewAccessKeyService()
	// An app holds at most service.MaxSecretsPreApp secrets.
	accessKeyList := make([]types.AccessKeyWithCreatedAt, 0, len(appCodes)*service.MaxSecretsPreApp)
	for _, appCode := range appCodes {
		accessKeys, err := svc.ListWithCreatedAtByAppCode(ctx, appCode)
		if err != nil {
			return fmt.Errorf("list access keys of app %s: %w", appCode, err)
		}
		accessKeyList = append(accessKeyList, accessKeys...)
	}

	if len(accessKeyList) == 0 {
		fmt.Println("no accessKey")
		return nil
	}

	fmt.Println("ID\tAppCode\tAppSecret\tCreatedAt")
	for _, ak := range accessKeyList {
		fmt.Printf("%d\t%s\t%s\t%v\n", ak.ID, ak.AppCode, ak.AppSecret, time.Unix(ak.CreatedAt, 0))
	}
	return nil
}

// DeleteAccessKey deletes the access key accessKeyID of appCode. The app must keep at
// least one secret.
func DeleteAccessKey(appCode string, accessKeyID int64) error {
	if appCode == "" {
		return errors.New("app code must not be empty")
	}
	if accessKeyID <= 0 {
		return errors.New("access key id must be a positive integer")
	}

	err := service.NewAccessKeyService().DeleteByID(context.Background(), appCode, accessKeyID)
	if err != nil {
		return fmt.Errorf("delete access key %d of app %s: %w", accessKeyID, appCode, err)
	}
	return nil
}
