import http from '../../http';
import type { ResourceGroup } from '@/services/source/oauth2/consent.ts';

/**
 * 验证
 */
export const verifyDeviceCode = (
  params: { user_code: string },
) =>
  http.post<{
    client_name: string
    client_logo_uri?: string
    realm_name: string
    resources: ResourceGroup[]
  }>('/api/v1/web/oauth2/device/verify', params);

/**
 * 提交
 */
export const confirmDeviceCode = (
  params: {
    user_code: string
    action: string
  },
) =>
  http.post<{ result: string }>('/api/v1/web/oauth2/device/confirm', params);
