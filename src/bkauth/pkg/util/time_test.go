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

package util_test

import (
	"encoding/json"
	"testing"
	"time"

	"bkauth/pkg/util"
)

func TestUnixTime_MarshalJSON(t *testing.T) {
	// 创建一个固定的时间
	testTime := time.Unix(1737876111, 0)
	ut := util.FromTime(testTime)

	// 序列化为 JSON
	data, err := json.Marshal(ut)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	// 验证输出是时间戳
	expected := "1737876111"
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

func TestUnixTime_UnmarshalJSON(t *testing.T) {
	// 准备 JSON 数据
	jsonData := []byte("1737876111")

	var ut util.UnixTime
	err := json.Unmarshal(jsonData, &ut)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	// 验证解析后的时间
	expected := time.Unix(1737876111, 0)
	if ut.ToTime().Unix() != expected.Unix() {
		t.Errorf("Expected %v, got %v", expected, ut.ToTime())
	}
}

func TestUnixTime_String(t *testing.T) {
	testTime := time.Unix(1737876111, 0)
	ut := util.FromTime(testTime)

	str := ut.String()
	if str == "" {
		t.Error("String() returned empty string")
	}
}

func TestUnixTime_ToTime(t *testing.T) {
	testTime := time.Unix(1737876111, 0)
	ut := util.FromTime(testTime)

	converted := ut.ToTime()
	if converted.Unix() != testTime.Unix() {
		t.Errorf("Expected %v, got %v", testTime, converted)
	}
}

func TestUnixTime_RoundTrip(t *testing.T) {
	// 测试完整的序列化和反序列化流程
	type TestStruct struct {
		Time util.UnixTime `json:"time"`
	}

	original := TestStruct{
		Time: util.FromTime(time.Unix(1737876111, 0)),
	}

	// 序列化
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 验证 JSON 格式
	expectedJSON := `{"time":1737876111}`
	if string(data) != expectedJSON {
		t.Errorf("Expected %s, got %s", expectedJSON, string(data))
	}

	// 反序列化
	var decoded TestStruct
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// 验证时间相等
	if decoded.Time.ToTime().Unix() != original.Time.ToTime().Unix() {
		t.Errorf("Time mismatch: expected %v, got %v",
			original.Time.ToTime(), decoded.Time.ToTime())
	}
}
