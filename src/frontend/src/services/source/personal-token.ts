import type { PersonalTokenRealm } from '@/constants/personal-token';

import http from '../http';

export interface IPersonalTokenResourceExtras {
  is_public?: boolean
  is_official?: boolean
  [key: string]: unknown
}

/** 已授予资源；与 audience 通过 audience 字段相等关联。 */
export interface IPersonalTokenResource {
  type: string
  level: string
  name: string
  display_name: string
  audience: string
  extras?: IPersonalTokenResourceExtras
}

/** 列表与详情共用的令牌对象。 */
export interface IPersonalToken {
  id: number
  name: string
  description: string
  token_mask: string
  audience: string[]
  resources: IPersonalTokenResource[] | null
  expires_at: number
  revoked: boolean
  revoked_at?: number
  created_at: number
  updated_at: number
}

export interface IGrantableResourceLevel {
  name: string
  display_name: string
}

/** 可授予资源类型元数据。 */
export interface IGrantableResourceType {
  name: string
  display_name: string
  audience: string
  levels: IGrantableResourceLevel[]
}

/** 可授予资源树节点；items 缺席时为叶子节点。 */
export interface IGrantableResource {
  name: string
  display_name: string
  audience: string
  extras?: IPersonalTokenResourceExtras
  items?: IGrantableResource[]
}

export interface IGrantableResourceList {
  count: number
  results: IGrantableResource[]
}

export interface IGrantableResourceListParams {
  type: string
  keyword?: string
  page?: number
  page_size?: number
}

export interface IGrantableResourceLookupParams {
  type: string
  name: string
}

/** 创建与编辑共用的全量请求参数。 */
export interface IPersonalTokenPayload {
  name: string
  description: string
  audience: string[]
  expires_at: number
}

/** 明文令牌只在创建响应中返回一次。 */
export interface IPersonalTokenCreateResult {
  id: number
  token: string
}

export interface IPersonalTokenRenewPayload { expires_at: number }

const resourceTypeRequestCache = new Map<PersonalTokenRealm, Promise<IGrantableResourceType[]>>();

/** 获取当前用户在指定 realm 下的全部令牌。 */
export const getPersonalTokenList = (realm: PersonalTokenRealm) =>
  http.get<IPersonalToken[]>(`/api/v1/web/realms/${realm}/personal-tokens`);

/** 获取令牌详情。 */
export const getPersonalTokenDetail = (realm: PersonalTokenRealm, id: number) =>
  http.get<IPersonalToken>(`/api/v1/web/realms/${realm}/personal-tokens/${id}`);

/** 创建令牌。 */
export const createPersonalToken = (realm: PersonalTokenRealm, data: IPersonalTokenPayload) =>
  http.post<IPersonalTokenCreateResult>(`/api/v1/web/realms/${realm}/personal-tokens`, data);

/** 全量编辑令牌。 */
export const updatePersonalToken = (realm: PersonalTokenRealm, id: number, data: IPersonalTokenPayload) =>
  http.put<Record<string, never>>(`/api/v1/web/realms/${realm}/personal-tokens/${id}`, data);

/** 续期令牌。 */
export const renewPersonalToken = (
  realm: PersonalTokenRealm,
  id: number,
  data: IPersonalTokenRenewPayload,
) => http.post<Record<string, never>>(
  `/api/v1/web/realms/${realm}/personal-tokens/${id}/renew`,
  data,
);

/** 撤销令牌；接口不读取请求体。 */
export const revokePersonalToken = (realm: PersonalTokenRealm, id: number) =>
  http.post<Record<string, never>>(`/api/v1/web/realms/${realm}/personal-tokens/${id}/revoke`);

/** 获取可授予资源类型；静态元数据按 realm 缓存。 */
export const getGrantableResourceTypes = (realm: PersonalTokenRealm) => {
  const cachedRequest = resourceTypeRequestCache.get(realm);
  if (cachedRequest) {
    return cachedRequest;
  }

  const request = http.get<IGrantableResourceType[]>(
    `/api/v1/web/realms/${realm}/personal-tokens/grantable-resource-types`,
  );
  resourceTypeRequestCache.set(realm, request);
  request.catch(() => {
    resourceTypeRequestCache.delete(realm);
  });
  return request;
};

/** 分页获取指定类型的可授予资源。 */
export const getGrantableResourceList = (
  realm: PersonalTokenRealm,
  params: IGrantableResourceListParams,
) => http.get<IGrantableResourceList>(
  `/api/v1/web/realms/${realm}/personal-tokens/grantable-resources`,
  params,
);

/** 按完整层级名称精确查询一个可授予资源。 */
export const lookupGrantableResource = (
  realm: PersonalTokenRealm,
  params: IGrantableResourceLookupParams,
) => http.get<IGrantableResource>(
  `/api/v1/web/realms/${realm}/personal-tokens/grantable-resources/-/lookup`,
  params,
  { catchError: true },
);
