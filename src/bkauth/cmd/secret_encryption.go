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

// This command only exists to retire secrets sealed under the deprecated fixed nonce.
// Once every deployment has run `secret-encryption migrate`, this file and
// pkg/cli/secret_encryption.go can be deleted together, with nothing else to unwind.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"bkauth/pkg/cli"
)

var secretEncryptionDryRunParam bool

var secretEncryptionCmd = &cobra.Command{
	Use:   "secret-encryption",
	Short: "audit and migrate app secrets sealed under the deprecated fixed nonce",
}

var checkSecretEncryptionCmd = &cobra.Command{
	Use:   "check",
	Short: "report access keys still encrypted with the deprecated fixed nonce (read-only)",
	Long:  "example: bkauth secret-encryption check -c config.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := secretEncryptionStart(); err != nil {
			return err
		}
		defer secretEncryptionFinish()

		return cli.CheckSecretEncryption()
	},
}

var migrateSecretEncryptionCmd = &cobra.Command{
	Use:   "migrate",
	Short: "re-encrypt access keys stored with the deprecated fixed nonce",
	Long:  "example: bkauth secret-encryption migrate -c config.yaml --dry-run",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := secretEncryptionStart(); err != nil {
			return err
		}
		defer secretEncryptionFinish()

		return cli.MigrateSecretEncryption(secretEncryptionDryRunParam)
	},
}

func init() {
	secretEncryptionCmd.AddCommand(checkSecretEncryptionCmd)

	migrateSecretEncryptionCmd.Flags().BoolVar(
		&secretEncryptionDryRunParam, "dry-run", false,
		"report what would be re-encrypted without writing to the database",
	)
	secretEncryptionCmd.AddCommand(migrateSecretEncryptionCmd)

	rootCmd.AddCommand(secretEncryptionCmd)
}

func secretEncryptionStart() error {
	fmt.Println("secret-encryption start!")

	// Cached ciphertexts are deliberately left to expire on their own.
	_, err := setupDataCommand()
	return err
}

func secretEncryptionFinish() {
	fmt.Println("secret-encryption finish!")
}
