import { defineStore } from 'pinia';

import { getUserInfo } from '@/services/source/basic';

type InfoType = Awaited<ReturnType<typeof getUserInfo>>;

export const useUserInfo = defineStore('useUserInfo', {
  state: (): Record<string, InfoType> => ({ info: { username: '' } }),
  actions: {
    async fetchUserInfo() {
      this.info = await getUserInfo();
      return this.info;
    },
  },
});
