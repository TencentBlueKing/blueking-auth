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

// Package service provides the business-layer services used by the application.
package service

//go:generate mockgen -source=$GOFILE -destination=./mock/$GOFILE -package=mock

import (
	"context"
	"fmt"

	"bkauth/pkg/database/dao"
	"bkauth/pkg/errorx"
	"bkauth/pkg/service/types"
	"bkauth/pkg/util"
)

const (
	AccessKeySVC = "AccessKeySVC"

	// MaxSecretsPreApp 每个 App 最多有 2 个 secret
	MaxSecretsPreApp = 2
	// MinSecretsPreApp 每个 App 至少有一个 secret
	MinSecretsPreApp = 1
)

type AccessKeyService interface {
	// TODO：Create / CreateWithSecret 可以引入一个输入的 struct（比如 AccessKeyCreateInput），避免多个 string 参数的顺序错误和跨层“散弹式修改”
	Create(ctx context.Context, appCode, createdSource, description string) (types.AccessKey, error)
	CreateWithSecret(ctx context.Context, appCode, appSecret, createdSource, description string) error
	UpdateByID(ctx context.Context, id int64, updateFieldMap map[string]any) error
	DeleteByID(ctx context.Context, appCode string, id int64) error
	ListWithCreatedAtByAppCode(ctx context.Context, appCode string) ([]types.AccessKeyWithCreatedAt, error)
	Verify(ctx context.Context, appCode, appSecret string) (bool, error)
	ListEncryptedAccessKeyByAppCode(ctx context.Context, appCode string) (appSecrets []types.AccessKey, err error)
	List(ctx context.Context) ([]types.AccessKey, error)
	ExistsByAppCodeAndID(ctx context.Context, appCode string, id int64) (bool, error)
}

type accessKeyService struct {
	manager dao.AccessKeyManager
}

// NewAccessKeyService creates an access key service.
func NewAccessKeyService() AccessKeyService {
	return &accessKeyService{
		manager: dao.NewAccessKeyManager(),
	}
}

// Create : 创建应用密钥，createdSource 为创建来源，即哪个系统创建的
func (s *accessKeyService) Create(
	ctx context.Context,
	appCode, createdSource, description string,
) (accessKey types.AccessKey, err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(AccessKeySVC, "Create")

	// 数量的保证是业务上的一个基础逻辑
	// Note: 这里没有处理并发问题导致创建超过 2 个的问题，因为多创建了也没有太多影响
	count, err := s.manager.Count(ctx, appCode)
	if err != nil {
		return accessKey, errorWrapf(err, "manager.Count appCode=`%s` fail", appCode)
	}
	if count >= MaxSecretsPreApp {
		// Note: 这里不能使用 errorWrapf，否则上层无法判断错误是系统错误还是校验不通过
		err = util.ValidationErrorWrap(
			fmt.Errorf("app(%s) can only have %d secrets, [current %d]", appCode, MaxSecretsPreApp, count))
		return accessKey, err
	}

	daoAccessKey := newDaoAccessKey(appCode, createdSource, description)
	id, err := s.manager.Create(ctx, daoAccessKey)
	if err != nil {
		return accessKey, errorWrapf(err, "manager.Create accessKey=`%+v` fail", daoAccessKey)
	}

	// 获取明文密钥
	appSecret, err := convertToPlainAppSecret(daoAccessKey.AppSecret)
	if err != nil {
		return accessKey, errorWrapf(
			err,
			"convertToPlainAppSecret encryptedAppSecret=`%s` fail",
			daoAccessKey.AppSecret,
		)
	}

	accessKey = types.AccessKey{
		ID:          id,
		AppCode:     appCode,
		AppSecret:   appSecret,
		Enabled:     daoAccessKey.Enabled,
		Description: description,
	}
	return accessKey, err
}

// CreateWithSecret : 创建应用密钥，支持指定 appSecret 的值，createdSource 为创建来源，即哪个系统创建的
func (s *accessKeyService) CreateWithSecret(
	ctx context.Context,
	appCode, appSecret, createdSource, description string,
) (err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(AccessKeySVC, "CreateWithSecret")

	daoAccessKey := newDaoAccessKeyWithAppSecret(appCode, appSecret, createdSource, description)
	_, err = s.manager.Create(ctx, daoAccessKey)
	if err != nil {
		return errorWrapf(err, "manager.Create accessKey=`%+v` fail", daoAccessKey)
	}

	return err
}

// DeleteByID deletes an access key by app code and id.
func (s *accessKeyService) DeleteByID(ctx context.Context, appCode string, id int64) (err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(AccessKeySVC, "DeleteByID")

	// 只剩下唯一一个 Secret，则无法删除
	// TODO: 这里没有处理并发问题可能会导致一个 App 没有任何一个 Secret，进而导致 App 无法调用任何蓝鲸 API
	//  Note: 乐观锁只能解决查询和修改的数据是相同的问题，这里是查询数量，并修改其中一条，乐观锁应该无法很好解决
	//  可以使用 select_for_update 之类的悲观锁，或引入全局锁，如 Redis 分布式锁解决这个问题
	//  但目前没有这个必要，因为管理 Secret 的行为是在 PaaS 端，可以让用户删除时，明确输入要删除的 Secret 做确认
	count, err := s.manager.Count(ctx, appCode)
	if err != nil {
		return errorWrapf(err, "manager.Count appCode=`%s` fail", appCode)
	}
	if count <= MinSecretsPreApp {
		return util.ValidationErrorWrap(
			fmt.Errorf("app(%s) have %d secret at least, [current %d]", appCode, MinSecretsPreApp, count))
	}

	// 防御性，避免误删除 Secret，所以需要额外 AppCode 来二次保证
	_, err = s.manager.DeleteByID(ctx, appCode, id)
	if err != nil {
		return errorWrapf(err, "manager.DeleteByID appCode=`%s` id=`%d` fail", appCode, id)
	}

	return err
}

// UpdateByID 更新 accessKey
func (s *accessKeyService) UpdateByID(
	ctx context.Context,
	id int64,
	updateFieldMap map[string]any,
) (err error) {
	if len(updateFieldMap) == 0 {
		return err
	}

	errorWrapf := errorx.NewLayerFunctionErrorWrapf(AccessKeySVC, "UpdateByID")
	_, err = s.manager.UpdateByID(ctx, id, updateFieldMap)
	if err != nil {
		return errorWrapf(err, "manager.UpdateByID updateFieldMap=`%+v` id=`%d` fail", updateFieldMap, id)
	}

	return err
}

// ListWithCreatedAtByAppCode lists access keys with creation time for the given app.
func (s *accessKeyService) ListWithCreatedAtByAppCode(ctx context.Context, appCode string) (
	accessKeys []types.AccessKeyWithCreatedAt, err error,
) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(AccessKeySVC, "ListWithCreatedAtByAppCode")

	daoAccessKeys, err := s.manager.ListWithCreatedAtByAppCode(ctx, appCode)
	if err != nil {
		return accessKeys, errorWrapf(err, "manager.ListWithCreatedAtByAppCode appCode=`%s` fail", appCode)
	}

	accessKeys = make([]types.AccessKeyWithCreatedAt, 0, len(daoAccessKeys))
	for _, accessKey := range daoAccessKeys {
		// 获取明文密钥
		appSecret, err := convertToPlainAppSecret(accessKey.AppSecret)
		if err != nil {
			return accessKeys, errorWrapf(
				err,
				"convertToPlainAppSecret encryptedAppSecret=`%s` fail",
				accessKey.AppSecret,
			)
		}

		accessKeys = append(accessKeys, types.AccessKeyWithCreatedAt{
			AccessKey: types.AccessKey{
				ID:          accessKey.ID,
				AppCode:     accessKey.AppCode,
				AppSecret:   appSecret,
				Enabled:     accessKey.Enabled,
				Description: accessKey.Description,
			},
			CreatedAt: accessKey.CreatedAt.Unix(),
		})
	}

	return accessKeys, err
}

// Verify checks whether the given app code and app secret are valid.
func (s *accessKeyService) Verify(ctx context.Context, appCode, appSecret string) (exists bool, err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(AccessKeySVC, "Verify")

	// DB 里存储的是加密后的密钥，需要对即将校验的 Secret 加密后查询
	encryptedAppSecret := ConvertToEncryptedAppSecret(appSecret)

	exists, err = s.manager.Exists(ctx, appCode, encryptedAppSecret)
	if err != nil {
		return false, errorWrapf(err, "manager.Exists appCode=`%s` appSecret=`%s` fail", appCode, appSecret)
	}

	return exists, err
}

// ListEncryptedAccessKeyByAppCode lists encrypted access keys for the given app.
func (s *accessKeyService) ListEncryptedAccessKeyByAppCode(
	ctx context.Context,
	appCode string,
) (appSecrets []types.AccessKey, err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(AccessKeySVC, "ListEncryptedSecretByAppCode")

	appSecretList, err := s.manager.ListAccessKeyByAppCode(ctx, appCode)
	if err != nil {
		return appSecrets, errorWrapf(err, "manager.ListAccessKeyByAppCode appCode=`%s` fail", appCode)
	}
	for _, appSecret := range appSecretList {
		appSecrets = append(appSecrets, types.AccessKey{
			AppSecret: appSecret.AppSecret,
			Enabled:   appSecret.Enabled,
		})
	}

	return appSecrets, err
}

// List lists all access keys with decrypted secrets.
func (s *accessKeyService) List(ctx context.Context) (accessKeys []types.AccessKey, err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(AccessKeySVC, "List")

	daoAccessKeys, err := s.manager.List(ctx)
	if err != nil {
		return accessKeys, errorWrapf(err, "manager.List fail")
	}

	accessKeys = make([]types.AccessKey, 0, len(daoAccessKeys))
	for _, daoAccessKey := range daoAccessKeys {
		// 获取明文密钥
		appSecret, err := convertToPlainAppSecret(daoAccessKey.AppSecret)
		if err != nil {
			return accessKeys, errorWrapf(
				err, "convertToPlainAppSecret encryptedAppSecret=`%s` fail", daoAccessKey.AppSecret)
		}
		accessKeys = append(accessKeys, types.AccessKey{
			ID:          daoAccessKey.ID,
			AppCode:     daoAccessKey.AppCode,
			AppSecret:   appSecret,
			Enabled:     daoAccessKey.Enabled,
			Description: daoAccessKey.Description,
		})
	}

	return accessKeys, err
}

// ExistsByAppCodeAndID checks whether the given app code and id pair exists.
func (s *accessKeyService) ExistsByAppCodeAndID(ctx context.Context, appCode string, id int64) (bool, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(AccessKeySVC, "ExistsByAppCodeAndID")

	exists, err := s.manager.ExistsByAppCodeAndID(ctx, appCode, id)
	if err != nil {
		return exists, errorWrapf(err, "manager.ExistsByAppCodeAndID appCode=`%s` id=`%d` fail", appCode, id)
	}

	return exists, nil
}
