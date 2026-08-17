<script setup lang="ts">
import { CollapseLeft } from 'bkui-vue/lib/icon';
import { getLoginURL } from '@/utils';
import { useEnv, useUserInfo } from '@/stores';

import BkLoginUserinfo, { ActionItem } from '@blueking/login-userinfo';
import '@blueking/login-userinfo/vue3/vue3.css';

const { t } = useI18n();
const userInfoStore = useUserInfo();
const envStore = useEnv();

const userinfo = computed(() => ({
  name: userInfoStore.info?.username || '',
  email: '',
  organization: undefined,
  timezone: undefined,
}));

const handleLogout = () => {
  location.href = getLoginURL(envStore.env.login_url, `${location.origin}/dashboard`, 'small');
};
</script>

<template>
  <div>
    <BkLoginUserinfo :userinfo="userinfo">
      <div>
        {{ userInfoStore.info?.username }}
      </div>
      <template #action>
        <ActionItem
          theme="danger"
          @click="handleLogout"
        >
          <template #icon>
            <CollapseLeft />
          </template>
          {{ t('退出登录') }}
        </ActionItem>
      </template>
    </BkLoginUserinfo>
  </div>
</template>
