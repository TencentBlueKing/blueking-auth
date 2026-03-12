import http from '../http';

/**
 * 当前用户信息
 */
export function getUserInfo() {
  return http.get<{ username: string }>('/api/v1/web/basic/userinfo');
}

/**
 * 当前环境相关配置
 */
export function getEnv() {
  return http.get<{
    version: string
    login_url: string
  }>('/api/v1/web/basic/env-vars');
}
