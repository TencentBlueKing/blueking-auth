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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// 统一 CLI 结果码，便于脚本和后续命令扩展
const (
	CodeSuccess = 0
	CodeError   = 1
)

const (
	OutputJSON = "json"
)

type Result struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func WriteResultJSON(w io.Writer, code int, msg string, data interface{}) {
	b, _ := json.Marshal(Result{Code: code, Msg: msg, Data: data})
	w.Write(b)
	io.WriteString(w, "\n")
}

func IsJSON(outputFormat string) bool {
	return strings.ToLower(outputFormat) == OutputJSON
}

// RespondError 统一处理错误输出：-o json 时写 JSON 到 stdout 并返回 err，table 时仅返回 err（由 Cobra 打印）
// 后续新增 key 子命令可直接 return cli.RespondError(outputFormat, err)
func RespondError(outputFormat string, err error) error {
	if err == nil {
		return nil
	}
	if IsJSON(outputFormat) {
		WriteResultJSON(os.Stdout, CodeError, err.Error(), nil)
	}
	return err
}

// RespondErrorMsg 校验/参数类错误时使用，避免重复写 if isJSON { WriteResultJSON(...); return }
func RespondErrorMsg(outputFormat, msg string) error {
	return RespondError(outputFormat, fmt.Errorf("%s", msg))
}

// RespondEmptyMsg 无数据时使用：json 输出 code=1 的 JSON 并返回 error（便于 exit 1），table 只打印 msg 并返回 nil（exit 0）
// 保证 json 的 msg 与 table 输出同一句，只维护一句文案
func RespondEmptyMsg(outputFormat, msg string) error {
	if IsJSON(outputFormat) {
		WriteResultJSON(os.Stdout, CodeError, msg, nil)
		return fmt.Errorf("%s", msg)
	}
	fmt.Fprintln(os.Stdout, msg)
	return nil
}

// RespondSuccess 统一处理成功输出：-o json 时写 JSON 到 stdout，否则执行 tableOutput
// 后续新增 key 子命令可直接 return cli.RespondSuccess(outputFormat, data, func() { ... })
func RespondSuccess(outputFormat string, data interface{}, tableOutput func()) error {
	return RespondSuccessWithMsg(outputFormat, "ok", data, tableOutput)
}

// RespondSuccessWithMsg 与 RespondSuccess 相同，但 json 的 msg 与 table 输出共用同一文案，便于只维护一句
func RespondSuccessWithMsg(outputFormat, msg string, data interface{}, tableOutput func()) error {
	if IsJSON(outputFormat) {
		WriteResultJSON(os.Stdout, CodeSuccess, msg, data)
		return nil
	}
	if tableOutput != nil {
		tableOutput()
	} else {
		fmt.Fprintln(os.Stdout, msg)
	}
	return nil
}
