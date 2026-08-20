/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - BkAuth available.
 * Copyright (C) 2025 Tencent. All rights reserved.
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
import { defineStore } from 'pinia';
import {
  type IEnv,
  type IPersonalTokenPolicy,
  getEnv,
} from '@/services/source/basic';

interface IEnvState extends Omit<IEnv, 'personal_token_policy'> { personal_token_policy: IPersonalTokenPolicy }

interface IState { env: IEnvState }

const DEFAULT_PERSONAL_TOKEN_POLICY = {
  max_ttl: 94_608_000,
  max_active_per_user: 20,
};

export const useEnv = defineStore('useEnv', {
  state: (): IState => ({
    env: {
      version: '',
      login_url: '',
      personal_token_policy: { ...DEFAULT_PERSONAL_TOKEN_POLICY },
    },
  }),
  actions: {
    /**
     * 查询环境变量信息
     */
    async fetchEnv() {
      const result = await getEnv();
      this.env = {
        ...this.env,
        ...(result ?? {}),
        personal_token_policy: {
          ...DEFAULT_PERSONAL_TOKEN_POLICY,
          ...(result?.personal_token_policy ?? {}),
        },
      };
      return this.env;
    },
  },
});
