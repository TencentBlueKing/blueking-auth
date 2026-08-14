import http from '../http';

/** 令牌状态 */
export type TokenStatus = 'valid' | 'expired' | 'revoked';

/** 授权资源分组 */
export interface TokenResource {
  /** MCP 授权（全部或指定数量） */
  mcp?: {
    /** 是否授权全部 MCP */
    all: boolean
    /** 指定授权的 MCP 数量 */
    count: number
  }
  /** API 授权 */
  api?: {
    /** 是否授权全部 API */
    all?: boolean
    /** 授权网关数量 */
    gateway_count: number
    /** 授权 API 数量 */
    api_count: number
  }
}

/** 个人令牌列表项 */
export interface PersonalTokenItem {
  id: number
  /** 令牌名称 */
  name: string
  /** 备注 */
  description: string
  /** 授权资源 */
  resource: TokenResource
  /** 令牌（脱敏展示） */
  token: string
  /** 过期时间 */
  expired_at: string
  /** 状态 */
  status: TokenStatus
}

/** MCP 资源明细 */
export interface TokenMcpResource {
  /** MCP 名称 */
  name: string
  /** MCP 唯一标识 */
  id: string
}

/** 网关资源明细 */
export interface TokenGatewayResource {
  /** 网关唯一标识 */
  id?: string
  /** 网关名称 */
  name: string
  /** 是否官方网关 */
  is_official: boolean
}

/** API 资源明细 */
export interface TokenApiResource {
  /** API 唯一标识 */
  id?: string
  /** 所属网关唯一标识 */
  gateway_id?: string
  /** API 名称 */
  name: string
  /** API 描述 */
  action: string
  /** 是否公开 */
  is_public: boolean
}

/** 个人令牌详情 */
export interface PersonalTokenDetail extends PersonalTokenItem {
  /** 是否永久有效 */
  permanent?: boolean
  /** 最近使用时间 */
  last_used_at: string
  /** 创建时间 */
  created_at: string
  /** 授权的 MCP 资源列表 */
  mcp_resources: TokenMcpResource[]
  /** 授权的网关资源列表 */
  gateway_resources: TokenGatewayResource[]
  /** 授权的 API 资源列表 */
  api_resources: TokenApiResource[]
}

/** 自定义非公开 MCP */
export interface CustomMcpResource {
  /** MCP 服务地址 */
  server_url: string
  /** MCP 名称 */
  name: string
}

/** 自定义非公开 API */
export interface CustomApiResource {
  /** 网关名称 */
  gateway_name: string
  /** API 资源名称 */
  name: string
}

/** 新建/编辑个人令牌请求参数 */
export interface PersonalTokenPayload {
  name: string
  description: string
  permanent: boolean
  expired_at: string | null
  resource: {
    mcp: {
      all: boolean
      ids: string[]
      custom_resources: CustomMcpResource[]
    }
    api: {
      all: boolean
      gateway_ids: string[]
      ids: string[]
      custom_resources: CustomApiResource[]
    }
  }
}

/** 新建个人令牌响应 */
export interface PersonalTokenCreateResult {
  id: number
  name: string
  token: string
}

/** 列表请求参数 */
export interface PersonalTokenListParams {
  offset?: number
  limit?: number
  /** 状态筛选 */
  status?: TokenStatus | ''
  /** 关键字搜索：令牌名称、备注、MCP 名称、API 名称、网关名称 */
  keyword?: string
}

/**
 * 获取个人令牌列表
 */
export function getPersonalTokenList(params: PersonalTokenListParams) {
  // TODO: 接口就绪后替换为真实请求
  // return http.get<{
  //   count: number
  //   results: PersonalTokenItem[]
  // }>('/api/v1/web/personal-tokens/', params);
  return getMockPersonalTokenList(params);
}

/**
 * 获取个人令牌详情
 */
export function getPersonalTokenDetail(id: number) {
  // TODO: 接口就绪后替换为真实请求
  // return http.get<PersonalTokenDetail>(`/api/v1/web/personal-tokens/${id}/`);
  return getMockPersonalTokenDetail(id);
}

// ---------------------------------------------------------------------------
// Mock 数据（接口就绪后可删除）
// ---------------------------------------------------------------------------

/** 生成相对当前时间偏移 days 天的日期字符串（YYYY-MM-DD HH:mm:ss） */
function offsetDate(days: number): string {
  const date = new Date(Date.now() + days * 24 * 60 * 60 * 1000);
  const pad = (num: number) => String(num).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} 12:00:00`;
}

const mockPersonalTokenList: PersonalTokenItem[] = [
  {
    // 有效 + 未过期（30 天后）
    id: 1,
    name: 'Cursor 日常开发',
    description: '本地 IDE 使用',
    resource: {
      mcp: {
        all: true,
        count: 0,
      },
      api: {
        all: false,
        gateway_count: 3,
        api_count: 34,
      },
    },
    token: 'pat_bk_******dsexef',
    expired_at: offsetDate(30),
    status: 'valid',
  },
  {
    // 有效 + 即将过期（3 天后，7 天内）
    id: 2,
    name: 'Cursor 日常开发',
    description: '本地 IDE 使用',
    resource: {
      mcp: {
        all: false,
        count: 4,
      },
      api: {
        all: false,
        gateway_count: 3,
        api_count: 34,
      },
    },
    token: 'pat_bk_******dsexef',
    expired_at: offsetDate(3),
    status: 'valid',
  },
  {
    // 有效 + 即将过期（1 天后，7 天内）
    id: 3,
    name: 'Cursor 日常开发',
    description: '本地 IDE 使用',
    resource: {
      mcp: {
        all: true,
        count: 0,
      },
      api: {
        all: false,
        gateway_count: 3,
        api_count: 34,
      },
    },
    token: 'pat_bk_******dsexef',
    expired_at: offsetDate(1),
    status: 'valid',
  },
  {
    // 已过期（状态为 expired，过期时间已过）
    id: 4,
    name: 'Cursor 日常开发',
    description: '本地 IDE 使用',
    resource: {
      mcp: {
        all: true,
        count: 0,
      },
      api: {
        all: false,
        gateway_count: 3,
        api_count: 34,
      },
    },
    token: 'pat_bk_******dsexef',
    expired_at: offsetDate(-2),
    status: 'expired',
  },
  {
    // 已过期（状态为 expired，过期时间已过）
    id: 5,
    name: 'Cursor 日常开发',
    description: '本地 IDE 使用',
    resource: {
      mcp: {
        all: false,
        count: 4,
      },
      api: {
        all: false,
        gateway_count: 3,
        api_count: 34,
      },
    },
    token: 'pat_bk_******dsexef',
    expired_at: offsetDate(-10),
    status: 'expired',
  },
  {
    // 已撤销
    id: 6,
    name: 'Cursor 日常开发',
    description: '本地 IDE 使用',
    resource: {
      mcp: {
        all: true,
        count: 0,
      },
      api: {
        all: false,
        gateway_count: 3,
        api_count: 34,
      },
    },
    token: 'pat_bk_******dsexef',
    expired_at: offsetDate(20),
    status: 'revoked',
  },
];

/** 本地 mock 列表：支持状态筛选、关键字搜索与分页 */
function getMockPersonalTokenList(params: PersonalTokenListParams): Promise<{
  count: number
  results: PersonalTokenItem[]
}> {
  const {
    offset = 0,
    limit = 10,
    status = '',
    keyword = '',
  } = params;

  let list = [...mockPersonalTokenList];

  if (status) {
    list = list.filter(item => item.status === status);
  }

  if (keyword) {
    const kw = keyword.trim().toLowerCase();
    list = list.filter(item =>
      item.name.toLowerCase().includes(kw)
      || item.description.toLowerCase().includes(kw)
      || item.token.toLowerCase().includes(kw),
    );
  }

  const count = list.length;
  const results = list.slice(offset, offset + limit);

  return new Promise((resolve) => {
    setTimeout(() => {
      resolve({
        count,
        results,
      });
    }, 300);
  });
}

/** 本地 mock 详情：基于列表数据补充资源明细与时间字段 */
function getMockPersonalTokenDetail(id: number): Promise<PersonalTokenDetail> {
  const base = mockPersonalTokenList.find(item => item.id === id) ?? mockPersonalTokenList[0]!;

  return new Promise((resolve) => {
    setTimeout(() => {
      resolve({
        ...base,
        last_used_at: offsetDate(-1),
        created_at: offsetDate(-30),
        mcp_resources: [
          {
            name: 'MCP名称2',
            id: 'mcp-id2',
          },
        ],
        gateway_resources: [
          {
            id: 'gateway-01',
            name: '业务网关01',
            is_official: true,
          },
          {
            id: 'gateway-02',
            name: '业务网关01',
            is_official: false,
          },
          {
            id: 'gateway-03',
            name: '业务网关很长很长很长哪个01名称也许会超出显示区域范围',
            is_official: false,
          },
        ],
        api_resources: [
          {
            id: 'api-id1',
            gateway_id: 'gateway-01',
            name: 'API 名称2',
            action: '查询资源列表',
            is_public: true,
          },
          {
            id: 'api-id2',
            gateway_id: 'gateway-01',
            name: 'API 名称2',
            action: '查询资源列表',
            is_public: false,
          },
        ],
      });
    }, 300);
  });
}

/**
 * 新增个人令牌
 */
export function createPersonalToken(data: PersonalTokenPayload) {
  return http.post<PersonalTokenCreateResult>('/api/v1/web/personal-tokens/', data);
}

/**
 * 编辑个人令牌
 */
export function updatePersonalToken(id: number, data: PersonalTokenPayload) {
  return http.put<void>(`/api/v1/web/personal-tokens/${id}/`, data);
}

/**
 * 续期个人令牌
 */
export function renewPersonalToken(id: number, data: Record<string, any>) {
  return http.post(`/api/v1/web/personal-tokens/${id}/renew/`, data);
}

/**
 * 撤销个人令牌
 */
export function revokePersonalToken(id: number) {
  return http.post(`/api/v1/web/personal-tokens/${id}/revoke/`, {});
}
