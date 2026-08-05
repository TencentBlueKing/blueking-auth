import dayjs from 'dayjs';
import type { PersonalToken } from '@/services/source/personalToken';

/**
 * 展示态在后端三态（active/expired/revoked）基础上，把「有效但临近过期」单独拆出
 * 为「待续期」，与设计稿一致。阈值为距过期 <= 15 天。
 */
export type DisplayStatus = 'normal' | 'expiring' | 'expired' | 'revoked';

const EXPIRING_THRESHOLD_DAYS = 15;

export interface StatusMeta {
  label: string
  /** bk-tag theme，revoked 用空表示灰色默认样式 */
  theme?: 'success' | 'warning' | 'danger'
}

export const STATUS_META: Record<DisplayStatus, StatusMeta> = {
  normal: {
    label: '正常',
    theme: 'success',
  },
  expiring: {
    label: '待续期',
    theme: 'warning',
  },
  expired: {
    label: '已过期',
    theme: 'danger',
  },
  revoked: { label: '已撤销' },
};

export function getDisplayStatus(token: PersonalToken): DisplayStatus {
  if (token.status === 'revoked') {
    return 'revoked';
  }
  if (token.status === 'expired') {
    return 'expired';
  }
  const days = dayjs(token.expires_at).diff(dayjs(), 'day');
  return days <= EXPIRING_THRESHOLD_DAYS ? 'expiring' : 'normal';
}

/** 令牌是否可续期/撤销（仅未撤销的令牌可操作） */
export function isTokenActionable(token: PersonalToken): boolean {
  return token.status !== 'revoked';
}

/** 后端时间（RFC3339 UTC）转本地时区展示 */
export function formatDateTime(value?: string | null): string {
  if (!value) {
    return '--';
  }
  const d = dayjs(value);
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm:ss') : '--';
}

/**
 * 把资源 audience（如 "service:codecc"）解析成 { type, name }，
 * 便于按类型分组展示。
 */
export function parseAudience(aud: string): {
  type: string
  name: string
} {
  const idx = aud.indexOf(':');
  if (idx < 0) {
    return {
      type: '',
      name: aud,
    };
  }
  return {
    type: aud.slice(0, idx),
    name: aud.slice(idx + 1),
  };
}

/** 常用续期时长（秒） */
export const RENEW_PRESETS: {
  label: string
  seconds: number
}[] = [
  {
    label: '30 天',
    seconds: 30 * 86400,
  },
  {
    label: '90 天',
    seconds: 90 * 86400,
  },
  {
    label: '180 天',
    seconds: 180 * 86400,
  },
  {
    label: '365 天',
    seconds: 365 * 86400,
  },
];

/** 创建/续期默认预选时长（秒），90 天 */
export const DEFAULT_TTL_SECONDS = 90 * 86400;

/** 默认预选天数，用于日期选择器初值 */
export const DEFAULT_TTL_DAYS = 90;
