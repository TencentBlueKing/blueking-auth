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
// Once every deployment has run `secret_encryption migrate`, this file and
// pkg/cli/secret_encryption.go can be deleted together, with nothing else to unwind.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"bkauth/pkg/cli"
	"bkauth/pkg/logging"
)

var secretEncryptionDryRunParam bool

var secretEncryptionCmd = &cobra.Command{
	Use:   "secret_encryption",
	Short: "audit and migrate app secrets sealed under the deprecated fixed nonce",
	Long:  "",
}

var checkSecretEncryptionCmd = &cobra.Command{
	Use:   "check",
	Short: "report access keys still encrypted with the deprecated fixed nonce (read-only)",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		secretEncryptionStart()
		defer secretEncryptionFinish()

		cli.CheckSecretEncryption()
	},
}

var migrateSecretEncryptionCmd = &cobra.Command{
	Use:   "migrate",
	Short: "re-encrypt access keys stored with the deprecated fixed nonce, example: secret_encryption migrate --dry-run",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		secretEncryptionStart()
		defer secretEncryptionFinish()

		cli.MigrateSecretEncryption(secretEncryptionDryRunParam)
	},
}

func init() {
	secretEncryptionCmd.PersistentFlags().StringVarP(
		&cfgFile, "config", "c", "", "config file (default is config.yml;required)",
	)
	secretEncryptionCmd.PersistentFlags().Bool("viper", true, "Use Viper for configuration")
	_ = secretEncryptionCmd.MarkPersistentFlagRequired("config")

	secretEncryptionCmd.AddCommand(checkSecretEncryptionCmd)

	migrateSecretEncryptionCmd.Flags().BoolVar(
		&secretEncryptionDryRunParam, "dry-run", false,
		"report what would be re-encrypted without writing to the database",
	)
	secretEncryptionCmd.AddCommand(migrateSecretEncryptionCmd)

	rootCmd.AddCommand(secretEncryptionCmd)
}

// secretEncryptionStart is called from Run rather than a PersistentPreRun hook, since
// cobra validates required flags only after those hooks have already fired.
func secretEncryptionStart() {
	fmt.Println("secret_encryption start!")

	if cfgFile != "" {
		zap.S().Infof("Load config file: %s", cfgFile)
		viper.SetConfigFile(cfgFile)
	}
	initConfig()

	if globalConfig.Debug {
		fmt.Println(globalConfig)
	}

	// No initRedis / initCaches: this command only reads and rewrites rows, and
	// deliberately leaves cached ciphertexts to expire on their own. Keeping Redis out
	// means the migration can still run while Redis is down.
	initLogger()
	initDatabase()
	initCryptos()
}

func secretEncryptionFinish() {
	logging.SyncAll()
	fmt.Println("secret_encryption finish!")
}
