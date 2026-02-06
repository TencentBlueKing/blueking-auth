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
 * We undertake not to change the open source license (MIT license) applicable to
 * the current version of the project delivered to anyone in the future.
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

func RespondSuccess(outputFormat string, data any, tableOutput func()) error {
	if strings.ToLower(outputFormat) == "json" {
		writeJSON(data)
		return nil
	}
	if tableOutput != nil {
		tableOutput()
	} else {
		if s, ok := data.(string); ok {
			fmt.Fprintln(os.Stdout, s)
		} else {
			writeJSON(data)
		}
	}
	return nil
}

func writeJSON(data any) {
	b, _ := json.Marshal(data)
	os.Stdout.Write(b)
	io.WriteString(os.Stdout, "\n")
}
