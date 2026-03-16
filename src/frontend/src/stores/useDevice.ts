import { defineStore } from 'pinia';
import type { ConsentResponseData } from '@/services/source/oauth2/consent.ts';

export const useDevice = defineStore('useDevice', {
  state: (): {
    consentInfo: ConsentResponseData | null
    code: string
  } => ({
    consentInfo: null,
    code: '',
  }),
  actions: {
    setConsentInfo(info: ConsentResponseData) {
      this.consentInfo = info;
    },
    setCode(code: string = '') {
      this.code = code;
    },
  },
});
