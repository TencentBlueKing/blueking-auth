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

package common

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// WriteOutput writes command result to stdout in the specified format.
func WriteOutput(outputFormat string, data any) error {
	format := strings.ToLower(outputFormat)
	if format == "json" {
		return WriteJSON(data)
	}
	if s, ok := data.(string); ok {
		fmt.Fprintln(os.Stdout, s)
	}
	return nil
}

func WriteJSON(data any) error {
	return json.NewEncoder(os.Stdout).Encode(data)
}
