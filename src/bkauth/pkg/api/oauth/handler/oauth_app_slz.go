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
	validator "github.com/go-playground/validator/v10"
)

type redirectURLsSerializer struct {
	RedirectURLs []string `json:"redirect_urls" binding:"required" example:"[https://example.com/, http://example.com]"`
}

type redirectURLSerializer struct {
	URL string `validate:"url"`
}

func (s *redirectURLsSerializer) validateRedirectURLs() error {
	validate := validator.New()
	for _, url := range s.RedirectURLs {
		if err := validate.Struct(redirectURLSerializer{URL: url}); err != nil {
			return err
		}
	}
	return nil
}

type createOAuthAppSerializer struct {
	redirectURLsSerializer
}

func (s *createOAuthAppSerializer) validate() error {
	return s.validateRedirectURLs()
}

type updateOAuthAppSerializer struct {
	redirectURLsSerializer
}

func (s *updateOAuthAppSerializer) validate() error {
	// 校验URL
	return s.validateRedirectURLs()
}
