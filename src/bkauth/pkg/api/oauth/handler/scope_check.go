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

package handler

import (
	"errors"
	"fmt"

	"bkauth/pkg/service"
	svctypes "bkauth/pkg/service/types"
)

type AllScopes struct {
	IDSet   map[string]string
	NameSet map[string]string
}

func (a *AllScopes) ContainsID(id string) bool {
	_, ok := a.IDSet[id]
	return ok
}

func (a *AllScopes) ContainsName(name string) bool {
	_, ok := a.NameSet[name]
	return ok
}

func (a *AllScopes) ContainsNameExcludeSelf(name, baseID string) bool {
	id, ok := a.NameSet[name]
	// 不存在则直接返回
	if !ok {
		return false
	}
	// 存在则需要对比是否自身名称，若是则排除掉，若不是则表示是其他的名称
	return id != baseID
}

// newAllScopes :将所有Scope转换为 id集合 和 name集合，便于后续集合判断
func newAllScopes(scopes []svctypes.Scope) *AllScopes {
	idSet := map[string]string{}
	nameSet := map[string]string{}

	for _, s := range scopes {
		idSet[s.ID] = s.ID
		nameSet[s.Name] = s.ID
	}

	return &AllScopes{
		IDSet:   idSet,
		NameSet: nameSet,
	}
}

// checkAllScopesUnique :检查传入的Scope的唯一性，保证不与已存在的ID和Name重复
func checkAllScopesUnique(targetID string, inScopes []scopeSerializer) error {
	svc := service.NewScopeService()
	scopes, err := svc.ListByTarget(targetID)
	if err != nil {
		return errors.New("query all scope fail")
	}

	allScopes := newAllScopes(scopes)
	for _, s := range inScopes {
		if allScopes.ContainsID(s.ID) {
			return fmt.Errorf("scope id[%s] already exists", s.ID)
		}
		if allScopes.ContainsName(s.Name) {
			return fmt.Errorf("scope name[%s] already exists", s.Name)
		}
	}

	// TODO: 是否需要限制Scope数量？？

	return nil
}

// checkScopeUpdateUnique :对于更新时，检查ScopeID是否存在，以及新的Name是否重复
func checkScopeUpdateUnique(targetID string, scopeID string, name string) error {
	svc := service.NewScopeService()
	scopes, err := svc.ListByTarget(targetID)
	if err != nil {
		return errors.New("query all scope fail")
	}

	allScopes := newAllScopes(scopes)
	if !allScopes.ContainsID(scopeID) {
		return fmt.Errorf("scope id[%s] not exists", scopeID)
	}

	// check name / name_en should be unique
	if name != "" && allScopes.ContainsNameExcludeSelf(name, scopeID) {
		return fmt.Errorf("scope name[%s] already exists", name)
	}

	return nil
}
