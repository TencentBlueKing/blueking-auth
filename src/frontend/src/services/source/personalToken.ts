import http from '../http';
import type { ResourceGroup } from '@/services/source/oauth2/consent.ts';

/**
 * 本期仅开放 devops realm。后端不按 realm 拦截，realm 完全由前端路由约束，
 * 该常量是路由缺少 realm 参数时的兜底值。blueking 就绪后扩展为可切换的 realm。
 */
export const PERSONAL_TOKEN_REALM = 'bk-devops';

/** 令牌生命周期状态（后端派生，非存储）：active / expired / revoked */
export type PersonalTokenStatus = 'active' | 'expired' | 'revoked';

/** 列表状态过滤：active=有效，inactive=已过期或已撤销 */
export type PersonalTokenState = 'active' | 'inactive';

/**
 * 个人令牌管理视图，列表 / 详情 / 编辑 / 续期 / 撤销均返回该结构，不含明文。
 * 时间字段为 RFC3339（UTC），展示时转本地时区。
 */
export interface PersonalToken {
  id: number
  name: string
  description: string
  realm_name: string
  audience: string[]
  token_mask: string
  status: PersonalTokenStatus
  expires_at: string
  revoked_at: string | null
  last_used_at: string | null
  created_at: string
  updated_at: string
}

/** 创建响应：在管理视图基础上附带一次性明文 token */
export interface CreatedPersonalToken extends PersonalToken { token: string }

export interface CreatePersonalTokenParams {
  name: string
  description?: string
  /** 逗号分隔的资源选择串，如 "service:codecc,service:foo" */
  resource: string
  /** 生命周期（秒），0/缺省表示使用后端默认 TTL */
  expires_in?: number
}

export interface UpdatePersonalTokenParams {
  name: string
  description?: string
  resource: string
}

const base = (realm: string = PERSONAL_TOKEN_REALM) =>
  `/api/v1/web/realms/${realm}/personal-tokens`;

/** 列表（支持 state 过滤） */
export const getPersonalTokens = (
  params: { state?: PersonalTokenState } = {},
  realm: string = PERSONAL_TOKEN_REALM,
) => http.get<PersonalToken[]>(base(realm), params);

/** 详情 */
export const getPersonalToken = (id: number, realm: string = PERSONAL_TOKEN_REALM) =>
  http.get<PersonalToken>(`${base(realm)}/${id}`);

/** 创建（明文仅此响应返回一次） */
export const createPersonalToken = (
  params: CreatePersonalTokenParams,
  realm: string = PERSONAL_TOKEN_REALM,
) => http.post<CreatedPersonalToken>(base(realm), params);

/** 编辑 name / description / audience */
export const updatePersonalToken = (
  id: number,
  params: UpdatePersonalTokenParams,
  realm: string = PERSONAL_TOKEN_REALM,
) => http.put<PersonalToken>(`${base(realm)}/${id}`, params);

/** 续期 */
export const renewPersonalToken = (
  id: number,
  params: { expires_in?: number },
  realm: string = PERSONAL_TOKEN_REALM,
) => http.post<PersonalToken>(`${base(realm)}/${id}/renew`, params);

/** 撤销（立即失效，记录保留用于审计，不可删除） */
export const revokePersonalToken = (id: number, realm: string = PERSONAL_TOKEN_REALM) =>
  http.post<PersonalToken>(`${base(realm)}/${id}/revoke`);

/**
 * 可选授权资源目录（创建/编辑页资源选择器数据源）。
 * Demo 阶段直接在前端固定；后续由网关侧按 allowed_app_code=personal 提供权威目录。
 */
export const SELECTABLE_RESOURCES: Record<string, ResourceGroup[]> = {
  [PERSONAL_TOKEN_REALM]: [
    {
      type: 'service',
      display_name: '蓝盾',
      items: [
        { name: 'codecc', display_name: 'CodeCC' },
      ],
    },
  ],
};

/** 取某 realm 的可选资源目录，缺省返回空 */
export const getSelectableResources = (realm: string = PERSONAL_TOKEN_REALM): ResourceGroup[] =>
  SELECTABLE_RESOURCES[realm] ?? [];
