import type { LocationQuery } from 'vue-router';
import type { PersonalTokenRealm } from '@/constants/personal-token';
import {
  type IPersonalTokenResource,
  getGrantableAudiencesResolvePath,
  resolveGrantableAudiences,
} from '@/services/source/personal-token';
import { paramsSerializer } from '@/services/http/lib/utils';
import { useEnv } from '@/stores';
import {
  getLoginURL,
  messageError,
  messageSuccess,
  messageWarn,
} from '@/utils';
import {
  type PersonalTokenAudienceTagType,
  parsePersonalTokenAudience,
  unixSecondsToDate,
} from './utils';

export const PERSONAL_TOKEN_SHARE_QUERY_KEYS = ['audience', 'name', 'description', 'expires_at'] as const;
export const MAX_SHARED_AUDIENCES = 100;
const MAX_AUDIENCE_LENGTH = 512;
const SHARE_REQUEST_BYTE_LIMIT = 7.5 * 1024;
const MAX_CONCURRENT_RESOLVE_REQUESTS = 4;

export interface IPersonalTokenShare {
  audience: string[]
  name: string
  description: string
  expiresAt: string
}

interface IPersonalTokenShareOptions {
  audience: string[]
  name?: string
  description?: string
  expires_at?: number
}

export interface ISharedAudienceIssue {
  audience: string
  reason: 'invalid-format' | 'unavailable'
}

export type SharedAudienceIssueTagType = PersonalTokenAudienceTagType;

export interface IParsedSharedAudienceIssue {
  audience: string
  reason: ISharedAudienceIssue['reason']
  tagType?: SharedAudienceIssueTagType
  name: string
  gateway?: string
  isRawAudience: boolean
}

// 按各 realm 的 audience 规范解析无法选中的授权范围，解析失败时回退展示原始值
export const parseSharedAudienceIssue = (
  realm: PersonalTokenRealm,
  issue: ISharedAudienceIssue,
): IParsedSharedAudienceIssue => {
  const { audience, reason } = issue;
  const parsed = parsePersonalTokenAudience(realm, audience);
  if (parsed) {
    return {
      audience,
      reason,
      tagType: parsed.tagType,
      name: parsed.name,
      gateway: parsed.gateway,
      isRawAudience: false,
    };
  }

  return {
    audience,
    reason,
    name: audience,
    isRawAudience: true,
  };
};

const getQueryValue = (value: LocationQuery[string] | undefined) => (Array.isArray(value) ? value[0] : value) ?? '';

export const readPersonalTokenShare = (query: LocationQuery): IPersonalTokenShare | null => {
  const audience = getQueryValue(query.audience);
  if (!audience.trim()) {
    return null;
  }
  return {
    audience: audience.split(','),
    name: [...getQueryValue(query.name)].slice(0, 64).join(''),
    description: [...getQueryValue(query.description)].slice(0, 255).join(''),
    expiresAt: getQueryValue(query.expires_at),
  };
};

export const getSharedExpiry = (value: string, maxTtl: number, now = Math.floor(Date.now() / 1000)) => {
  const expiresAt = Number(value);
  if (!/^\d+$/.test(value) || !Number.isSafeInteger(expiresAt) || expiresAt <= now || expiresAt > now + maxTtl) {
    return null;
  }
  return unixSecondsToDate(expiresAt);
};

export const resolveSharedAudiences = async (
  realm: PersonalTokenRealm,
  audiences: string[],
  signal: AbortSignal,
) => {
  const ordered = [...new Set(audiences.map(audience => audience.trim()))];
  const invalid = new Set(ordered.filter(audience => !audience || [...audience].length > MAX_AUDIENCE_LENGTH));
  const candidates = ordered.filter(audience => !invalid.has(audience));
  const resources = new Map<string, IPersonalTokenResource>();
  const pendingBatches = realm === 'bk-gpu' ? candidates.map(audience => [audience]) : [candidates];

  // 现有接口会因一项格式错误拒绝整批，仅在遇到参数错误时拆分重试。
  while (candidates.length && pendingBatches.length) {
    signal.throwIfAborted();
    const batches = pendingBatches.splice(0, MAX_CONCURRENT_RESOLVE_REQUESTS);
    await Promise.all(batches.map(async (batch) => {
      try {
        const displays = await resolveGrantableAudiences(realm, batch, signal);
        for (const resource of displays) {
          if (batch.includes(resource.audience)) {
            resources.set(resource.audience, resource);
          }
        }
      }
      catch (error) {
        if ((error as { error?: { code?: string } })?.error?.code !== 'INVALID_ARGUMENT') {
          throw error;
        }
        if (batch.length === 1) {
          invalid.add(batch[0]!);
          return;
        }
        const middle = Math.floor(batch.length / 2);
        pendingBatches.push(batch.slice(0, middle), batch.slice(middle));
      }
    }));
  }

  signal.throwIfAborted();
  return {
    resources: ordered.map(audience => resources.get(audience)).filter(resource => resource !== undefined),
    issues: ordered.filter(audience => !resources.has(audience)).map((audience): ISharedAudienceIssue => ({
      audience,
      reason: invalid.has(audience) ? 'invalid-format' : 'unavailable',
    })),
  };
};

const isRequestTooLong = (url: URL) =>
  new TextEncoder().encode(url.pathname + url.search).byteLength > SHARE_REQUEST_BYTE_LIMIT;

export const useCopyPersonalTokenShareLink = () => {
  const envStore = useEnv();
  const { t } = useI18n();
  const router = useRouter();

  return async (realm: PersonalTokenRealm, options: IPersonalTokenShareOptions) => {
    const audience = [...new Set(options.audience.filter(Boolean))];
    if (!audience.length) {
      return;
    }
    if (audience.length > MAX_SHARED_AUDIENCES) {
      messageWarn(t('授权范围链接最多支持 {count} 项授权范围，请减少后重试', { count: MAX_SHARED_AUDIENCES }));
      return;
    }
    if (audience.some(item => item.includes(','))) {
      messageWarn(t('授权范围包含英文逗号，无法生成授权范围链接'));
      return;
    }

    const { origin } = window.location;
    const { href } = router.resolve({
      name: 'PersonalToken',
      params: { realm },
    });
    const shareUrl = new URL(href, origin);
    const audiences = audience.join(',');
    shareUrl.searchParams.set('audience', audiences);
    for (const key of ['name', 'description', 'expires_at'] as const) {
      const value = options[key];
      if (value) {
        shareUrl.searchParams.set(key, String(value));
      }
    }

    const resolvePath = getGrantableAudiencesResolvePath(realm);
    const apiBase = (window.BK_AUTH_URL || '').replace(/\/+$/, '');
    const resolveUrl = new URL(`${apiBase}${resolvePath}?${paramsSerializer({ audiences })}`, origin);
    const loginUrl = new URL(getLoginURL(envStore.env.login_url || origin, shareUrl.href));
    if ([shareUrl, resolveUrl, loginUrl].some(isRequestTooLong)) {
      messageWarn(t('授权范围链接过长，请减少授权范围，或在需要整个范围时使用全选'));
      return;
    }

    try {
      await navigator.clipboard.writeText(shareUrl.href);
      messageSuccess(t('已复制授权范围链接'));
    }
    catch {
      messageError(t('复制失败'));
    }
  };
};
