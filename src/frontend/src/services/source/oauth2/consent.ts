import http from '../../http';

/**
 * 授权会话响应数据结构
 */
export interface ConsentResponseData {
  /** 授权会话状态 */
  status?: 'pending' | 'error'
  /** 请求授权的应用名称（status = pending 时存在） */
  client_name?: string
  /** 应用 Logo 地址 */
  client_logo_uri?: string
  /** 所属 Realm */
  realm: string
  /** 请求的资源/权限列表 */
  resources?: ResourceGroup[]
  /** 错误码（status = error 时存在） */
  error_code?: string
  /** 错误描述 */
  error_description?: string
}

/**
 * 资源组结构
 */
export interface ResourceGroup {
  /** 资源类型标识 */
  type: string
  /** 资源类型展示名 */
  display_name: string
  /** 资源项列表 */
  items: ResourceItem[]
}

/**
 * 资源项结构
 */
export interface ResourceItem {
  /** 资源项标识 */
  name?: string
  /** 资源项展示名 */
  display_name: string
  /** 子资源项（递归结构） */
  items?: ResourceItem[]
}

/**
 * 获取授权同意信息
 */
export function getConsentInfo(query: { consent_challenge: string }) {
  return http.get<ConsentResponseData>('/api/v1/web/oauth2/consent', query);
}

/**
 * 提交授权同意结果
 */
export const confirmConsent = (
  params: {
    consent_challenge: string
    action: string
  },
) =>
  http.post<{ redirect_url: string }>('/api/v1/web/oauth2/consent', params);
