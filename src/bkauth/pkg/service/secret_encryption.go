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

// Backing the `secret_encryption` command, this service retires secrets sealed under
// the deprecated fixed nonce. It is deliberately kept out of AccessKeyService so the
// whole migration can be dropped by deleting this file, its test,
// pkg/cli/secret_encryption.go and cmd/secret_encryption.go.
//
// No mockgen directive here on purpose: the only consumer is pkg/cli, which builds the
// service inline and has nothing to inject a mock into.

package service

import (
	"context"

	"bkauth/pkg/app"
	"bkauth/pkg/database/dao"
	"bkauth/pkg/errorx"
)

const SecretEncryptionSVC = "SecretEncryptionSVC"

// AccessKeyEncryptionStatus reports how one stored secret is currently encrypted.
// It lives here rather than in pkg/service/types, against the convention the other
// services follow, so the migration stays confined to files named secret_encryption.
type AccessKeyEncryptionStatus struct {
	ID      int64
	AppCode string
	// LegacyNonce is true while the row still uses the deprecated fixed nonce.
	LegacyNonce bool
}

type SecretEncryptionService interface {
	ListEncryptionStatus(ctx context.Context) ([]AccessKeyEncryptionStatus, error)
	ReencryptLegacySecret(ctx context.Context) ([]AccessKeyEncryptionStatus, error)
}

type secretEncryptionService struct {
	accessKeyManager dao.AccessKeyManager
}

func NewSecretEncryptionService() SecretEncryptionService {
	return &secretEncryptionService{
		accessKeyManager: dao.NewAccessKeyManager(),
	}
}

// ListEncryptionStatus reports, for every stored secret, whether it still uses the
// deprecated fixed nonce. Intended for the offline audit command.
func (s *secretEncryptionService) ListEncryptionStatus(
	ctx context.Context,
) ([]AccessKeyEncryptionStatus, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(SecretEncryptionSVC, "ListEncryptionStatus")

	daoAccessKeys, err := s.accessKeyManager.List(ctx)
	if err != nil {
		return nil, errorWrapf(err, "accessKeyManager.List fail")
	}

	statuses := make([]AccessKeyEncryptionStatus, 0, len(daoAccessKeys))
	for _, daoAccessKey := range daoAccessKeys {
		legacy, err := app.IsLegacyEncryptedSecret(daoAccessKey.AppSecret)
		if err != nil {
			return nil, errorWrapf(err, "app.IsLegacyEncryptedSecret accessKeyID=`%d` fail", daoAccessKey.ID)
		}

		statuses = append(statuses, AccessKeyEncryptionStatus{
			ID:          daoAccessKey.ID,
			AppCode:     daoAccessKey.AppCode,
			LegacyNonce: legacy,
		})
	}

	return statuses, nil
}

// ReencryptLegacySecret rewrites every secret still sealed under the fixed nonce,
// returning the rows it rewrote. Each row is updated independently: a failure
// halfway through leaves the already-rewritten rows valid, since both layouts stay
// decryptable, and re-running the command resumes from where it stopped.
func (s *secretEncryptionService) ReencryptLegacySecret(
	ctx context.Context,
) ([]AccessKeyEncryptionStatus, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(SecretEncryptionSVC, "ReencryptLegacySecret")

	daoAccessKeys, err := s.accessKeyManager.List(ctx)
	if err != nil {
		return nil, errorWrapf(err, "accessKeyManager.List fail")
	}

	reencrypted := make([]AccessKeyEncryptionStatus, 0, len(daoAccessKeys))
	for _, daoAccessKey := range daoAccessKeys {
		legacy, err := app.IsLegacyEncryptedSecret(daoAccessKey.AppSecret)
		if err != nil {
			return reencrypted, errorWrapf(err, "app.IsLegacyEncryptedSecret accessKeyID=`%d` fail", daoAccessKey.ID)
		}
		if !legacy {
			continue
		}

		plainSecret, err := app.DecryptSecret(daoAccessKey.AppSecret)
		if err != nil {
			return reencrypted, errorWrapf(err, "app.DecryptSecret accessKeyID=`%d` fail", daoAccessKey.ID)
		}

		encryptedSecret, err := app.EncryptSecret(plainSecret)
		if err != nil {
			return reencrypted, errorWrapf(err, "app.EncryptSecret accessKeyID=`%d` fail", daoAccessKey.ID)
		}

		_, err = s.accessKeyManager.UpdateByID(ctx, daoAccessKey.ID, map[string]interface{}{
			"app_secret": encryptedSecret,
		})
		if err != nil {
			return reencrypted, errorWrapf(err, "accessKeyManager.UpdateByID accessKeyID=`%d` fail", daoAccessKey.ID)
		}

		reencrypted = append(reencrypted, AccessKeyEncryptionStatus{
			ID:          daoAccessKey.ID,
			AppCode:     daoAccessKey.AppCode,
			LegacyNonce: false,
		})
	}

	return reencrypted, nil
}
