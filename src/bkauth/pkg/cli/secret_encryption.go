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

// Backing the `secret_encryption` command, this file retires secrets sealed under the
// deprecated fixed nonce. Delete it together with cmd/secret_encryption.go once every
// deployment has migrated.

package cli

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"bkauth/pkg/service"
)

// CheckSecretEncryption reports which stored secrets still use the deprecated fixed
// nonce. Read-only: it never writes to the database.
func CheckSecretEncryption() {
	ctx := context.Background()
	svc := service.NewSecretEncryptionService()

	statuses, err := svc.ListEncryptionStatus(ctx)
	if err != nil {
		zap.S().Error(err, "svc.ListEncryptionStatus fail")
		fmt.Println("check fail, see logs for detail")
		return
	}

	if len(statuses) == 0 {
		fmt.Println("no accessKey")
		return
	}

	legacyCount := 0
	fmt.Println("ID\tAppCode\tNonce")
	for _, status := range statuses {
		nonce := "random"
		if status.LegacyNonce {
			nonce = "fixed(legacy)"
			legacyCount++
		}
		fmt.Printf("%d\t%s\t%s\n", status.ID, status.AppCode, nonce)
	}

	fmt.Printf("\ntotal=%d, legacy=%d, ok=%d\n", len(statuses), legacyCount, len(statuses)-legacyCount)
	if legacyCount > 0 {
		fmt.Println("run `secret_encryption migrate` to re-encrypt the legacy rows")
	}
}

// MigrateSecretEncryption re-encrypts every secret still sealed under the fixed
// nonce. Safe to re-run: rows already migrated are skipped.
func MigrateSecretEncryption(dryRun bool) {
	ctx := context.Background()
	svc := service.NewSecretEncryptionService()

	if dryRun {
		statuses, err := svc.ListEncryptionStatus(ctx)
		if err != nil {
			zap.S().Error(err, "svc.ListEncryptionStatus fail")
			fmt.Println("dry run fail, see logs for detail")
			return
		}

		legacyCount := 0
		for _, status := range statuses {
			if !status.LegacyNonce {
				continue
			}
			legacyCount++
			fmt.Printf("would re-encrypt: id=%d app_code=%s\n", status.ID, status.AppCode)
		}
		fmt.Printf("\ndry run: %d of %d access keys would be re-encrypted\n", legacyCount, len(statuses))
		return
	}

	reencrypted, err := svc.ReencryptLegacySecret(ctx)
	// Rows rewritten before the error are already committed, so report them either way.
	for _, status := range reencrypted {
		fmt.Printf("re-encrypted: id=%d app_code=%s\n", status.ID, status.AppCode)
	}
	if err != nil {
		zap.S().Error(err, "svc.ReencryptLegacySecret fail")
		fmt.Printf("\nmigrate fail after %d access keys, re-run to resume\n", len(reencrypted))
		return
	}

	// Cached entries are deliberately left alone. They hold the pre-migration
	// ciphertext, which stays decryptable, and verification compares plaintext, so a
	// stale entry still answers correctly. Evicting every migrated app at once would
	// instead push them all back to the database together; the 5 minute TTL retires
	// them gradually for free.
	fmt.Printf("\nmigrate success: %d access keys re-encrypted\n", len(reencrypted))
}
