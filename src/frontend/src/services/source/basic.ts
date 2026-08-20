import http from '../http';

export interface IPersonalTokenPolicy {
  max_ttl: number
  max_active_per_user: number
}

export interface IEnv {
  version: string
  login_url: string
  personal_token_policy?: IPersonalTokenPolicy
}

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
  return http.get<IEnv>('/api/v1/web/basic/env-vars');
}
