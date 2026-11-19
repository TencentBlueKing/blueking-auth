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

import type { AxiosError, AxiosInterceptorManager, AxiosResponse } from 'axios';
import { Message } from 'bkui-vue';

import { downloadFile, getLoginURL, messageError } from '@/utils';

import { showLoginModal } from '@blueking/login-modal';

import RequestError from '../lib/request-error';

import { t } from '@/locales';

// 首次登录不弹小窗
let hasLoggedIn = false;

// 错误代码映射
const errorMessageMap = {
  INVALID_ARGUMENT: {
    overview: t('参数错误。'),
    suggestion: t('请检查提交的内容是否正确。'),
  },
  NO_PERMISSION: {
    overview: t('没有权限。'),
    suggestion: t('您没有执行该操作的权限。'),
  },
  RESOURCE_EXHAUSTED: {
    overview: t('活跃令牌数量已达上限。'),
    suggestion: t('请先撤销一个现有令牌后再试。'),
  },
};

const redirectLogin = (loginUrl: string) => {
  if (hasLoggedIn) {
    const loginSuccessURL = `${window.location.origin}/login-success.html`;
    showLoginModal({ loginUrl: getLoginURL(loginUrl, loginSuccessURL) });
  }
  else {
    window.location.href = getLoginURL(loginUrl, window.location.href);
  }
};

export default (interceptors: AxiosInterceptorManager<AxiosResponse>) => {
  interceptors.use(
    (response: AxiosResponse) => {
      if (response.headers['content-type'] === 'application/octet-stream') {
        downloadFile(response);
        return response.data;
      }
      if (response.status === 204 || response.data.data !== undefined || response.status < 400) {
        // 以下接口不鉴权，即使请求成功也不判定为已登录状态
        if (!['/version-log/', '/ping/', '/healthz/'].includes(response.config?.url || '')) {
          hasLoggedIn = true;
        }
        // 204 只根据状态码判定成功，不读取响应体。
        return response.status === 204 ? undefined : response.data;
      }
      // 后端逻辑处理报错
      const { code, message = '系统错误' } = response.data;
      throw new RequestError(code, message, response);
    },
    (
      error: AxiosError<{
        error: {
          code: string
          message: string
          data: any
        }
      }> & { __CANCEL__: any },
    ) => {
      // 主动取消请求
      if (error.__CANCEL__) {
        return Promise.reject(new RequestError('CANCEL', '请求已取消', undefined, error.config));
      }
      // 请求超时（无响应体）
      if (error.code === 'ECONNABORTED') {
        return Promise.reject(new RequestError('ECONNABORTED', t('请求超时'), undefined, error.config));
      }
      // 处理 http 错误响应逻辑
      if (error.response) {
        // 登录状态失效
        if (error.response.data.error?.code === 'UNAUTHENTICATED') {
          return Promise.reject(new RequestError(401, '登录状态失效', error.response, error.config));
        }
        // 默认使用 http 错误描述，如果有自定义错误描述优先使用
        let errorMessage = error.response.statusText;
        if (error.response.data && error.response.data.error?.code) {
          errorMessage = error.response.data.error.message;
        }
        return Promise.reject(
          new RequestError(error.response.status || -1, errorMessage, error.response, error.config),
        );
      }
      return Promise.reject(new RequestError(-1, `${window.BK_AUTH_URL} 无法访问`, undefined, error.config));
    },
  );

  // 统一错误处理逻辑
  interceptors.use(undefined, (error: RequestError) => {
    // 主动取消请求保持静默
    if (error.code === 'CANCEL') {
      return Promise.reject(error);
    }
    const catchError = Boolean(error.response?.config?.payload?.catchError ?? error.config?.payload?.catchError);
    // 无 HTTP 响应（超时或网络异常）按 catchError 策略提示
    if (!error.response) {
      if (!catchError) {
        messageError(error.code === 'ECONNABORTED' ? t('请求超时') : t('网络异常，请稍后重试'));
      }
      return Promise.reject(error);
    }
    const errorObj = error.response.data?.error;
    switch (error.code) {
      // 未登陆
      case 401:
        redirectLogin(errorObj.data.login_url);
        break;
      default:
        if (!catchError) {
          if (!errorObj) {
            messageError(error.message || t('系统错误'));
            break;
          }
          Message({
            theme: 'error',
            actions: [
              {
                id: 'assistant',
                disabled: true,
              },
            ],
            message: {
              code: errorObj.code,
              overview: errorMessageMap[errorObj.code as keyof typeof errorMessageMap]?.overview || t('系统错误'),
              suggestion: errorObj.code === 'RESOURCE_EXHAUSTED'
                ? errorMessageMap.RESOURCE_EXHAUSTED.suggestion
                : errorObj.message
                  ? JSON.stringify(errorObj.message)
                  : errorMessageMap[errorObj.code as keyof typeof errorMessageMap]?.suggestion || '',
              type: 'json',
              details: errorObj,
              assistant: '',
            },
          });
        }
    }
    return Promise.reject(error.response.data);
  });
};
