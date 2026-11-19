import type { PersonalTokenRealm } from '@/constants/personal-token';
import type { IPersonalToken } from '@/services/source/personal-token';

export type TokenStatus = 'valid' | 'expired' | 'revoked';
export type PersonalTokenAudienceTagType = 'mcp' | 'api' | 'gateway' | 'service' | 'resource';

export interface IParsedPersonalTokenAudience {
  tagType: PersonalTokenAudienceTagType
  name: string
  gateway?: string
}

const GATEWAY_AUDIENCE_PREFIX = 'gateway:';
const API_AUDIENCE_SEGMENT = '/api:';

// 单段前缀 audience 按 realm 映射到展示类型
const SIMPLE_AUDIENCE_PREFIX_MAP: Partial<Record<PersonalTokenRealm, {
  prefix: string
  tagType: PersonalTokenAudienceTagType
  disallowWildcard?: boolean
}>> = {
  'blueking': {
    prefix: 'mcp:',
    tagType: 'mcp',
    disallowWildcard: true,
  },
  'bk-devops': {
    prefix: 'service:',
    tagType: 'service',
  },
  'bk-gpu': {
    prefix: 'resource:',
    tagType: 'resource',
  },
};

// 按各 realm 契约解析 audience，无法识别时返回 null
export const parsePersonalTokenAudience = (
  realm: PersonalTokenRealm,
  audience: string,
): IParsedPersonalTokenAudience | null => {
  const simpleRule = SIMPLE_AUDIENCE_PREFIX_MAP[realm];
  if (simpleRule && audience.startsWith(simpleRule.prefix)) {
    const name = audience.slice(simpleRule.prefix.length);
    if (name && (!simpleRule.disallowWildcard || name !== '*')) {
      return {
        tagType: simpleRule.tagType,
        name,
      };
    }
    return null;
  }

  if (realm === 'blueking' && audience.startsWith(GATEWAY_AUDIENCE_PREFIX)) {
    const rest = audience.slice(GATEWAY_AUDIENCE_PREFIX.length);
    const separatorIndex = rest.indexOf(API_AUDIENCE_SEGMENT);
    if (separatorIndex > 0) {
      const gateway = rest.slice(0, separatorIndex);
      const apiName = rest.slice(separatorIndex + API_AUDIENCE_SEGMENT.length);
      if (gateway && gateway !== '*' && apiName) {
        return apiName === '*'
          ? {
            tagType: 'gateway',
            name: gateway,
            gateway,
          }
          : {
            tagType: 'api',
            name: apiName,
            gateway,
          };
      }
    }
  }

  return null;
};

const MILLISECONDS_PER_SECOND = 1000;
export const SECONDS_PER_MINUTE = 60;
export const SECONDS_PER_HOUR = 60 * SECONDS_PER_MINUTE;
export const SECONDS_PER_DAY = 24 * SECONDS_PER_HOUR;
const MAX_TTL_MARGIN_SECONDS = SECONDS_PER_DAY;

// 撤销状态优先于过期状态
export const getPersonalTokenStatus = (
  token: Pick<IPersonalToken, 'expires_at' | 'revoked'>,
  now = Date.now(),
): TokenStatus => {
  if (token.revoked) {
    return 'revoked';
  }
  if (token.expires_at * MILLISECONDS_PER_SECOND <= now) {
    return 'expired';
  }
  return 'valid';
};

export const unixSecondsToDate = (value: number) => new Date(value * MILLISECONDS_PER_SECOND);

export const dateToUnixSeconds = (value: Date | string) => {
  const date = value instanceof Date ? value : new Date(value.replace(/-/g, '/'));
  return Math.floor(date.getTime() / MILLISECONDS_PER_SECOND);
};

export const getEndOfDay = (value: Date | string) => {
  const date = value instanceof Date
    ? new Date(value)
    : unixSecondsToDate(dateToUnixSeconds(value));
  if (Number.isNaN(date.getTime())) {
    return null;
  }
  date.setHours(23, 59, 59, 0);
  return date;
};

export const formatUnixSeconds = (value?: number) => {
  if (!value) {
    return '--';
  }
  const date = unixSecondsToDate(value);
  const pad = (number: number) => String(number).padStart(2, '0');
  return date.getFullYear() + '-' + pad(date.getMonth() + 1) + '-' + pad(date.getDate()) + ' '
    + pad(date.getHours()) + ':' + pad(date.getMinutes()) + ':' + pad(date.getSeconds());
};

export const getRemainDays = (expiresAt: number, now = Date.now()) =>
  Math.ceil((expiresAt * MILLISECONDS_PER_SECOND - now) / (SECONDS_PER_DAY * MILLISECONDS_PER_SECOND));

// 最大有效期预留一天，避免客户端与服务端时间边界不一致
export const getEstimatedMaxExpiresAt = (maxTTL: number, now = Date.now()) =>
  new Date(now + (maxTTL - MAX_TTL_MARGIN_SECONDS) * MILLISECONDS_PER_SECOND);
