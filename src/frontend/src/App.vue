<template>
  <BkConfigProvider :locale="bkuiLocale">
    <div
      v-if="userLoaded"
      class="bg-[#F5F7FA]"
    >
      <RouterView />
    </div>
    <div class="global-footer">
      Copyright © 2026 Tencent BlueKing. All Rights Reserved. 3.0
    </div>
  </BkConfigProvider>
</template>

<script setup lang="ts">
import En from '../node_modules/bkui-vue/dist/locale/en.esm.js';
import ZhCn from '../node_modules/bkui-vue/dist/locale/zh-cn.esm.js';
import { useEnv, useUserInfo } from '@/stores';

const { locale } = useI18n();
const route = useRoute();
const userInfoStore = useUserInfo();
const envStore = useEnv();

const userLoaded = ref(false);

const bkuiLocale = computed(() => {
  if (locale.value === 'zh-cn') {
    return ZhCn;
  }
  return En;
});

watch(
  () => route.path,
  () => {
    getUserInfo();
  },
  {
    immediate: true,
    deep: true,
  },
);

async function getUserInfo() {
  try {
    await Promise.all([userInfoStore.fetchUserInfo(), envStore.fetchEnv()]);
    userLoaded.value = true;
  }
  catch {
    userLoaded.value = false;
  }
}

</script>

<style>
/* 整个滚动条 */

::-webkit-scrollbar {
  width: 4px;           /* 纵向滚动条宽度 */
  height: 4px;          /* 横向滚动条高度 */
}

/* 滚动条轨道 */

::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 4px;
}

/* 滚动条滑块 */

::-webkit-scrollbar-thumb {
  background: #DCDEE5;
  border-radius: 2px;
}

/* 滑块悬停 */

::-webkit-scrollbar-thumb:hover {
  background: #a1a1a1;
}

/* 滚动条角落（横纵滚动条交汇处） */

::-webkit-scrollbar-corner {
  background: transparent;
}
</style>

<style scoped lang="scss">

#app {
  width: 100%;
  min-width: 1366px;
  overflow: hidden;
  font-size: 14px;
  color: #63656e;
  text-align: left;
  background: #f5f7fb;

  .global-footer {
    z-index: 100;
    display: flex;
    height: 48px;
    font-size: 12px;
    color: #DCDEE5;
    background: #172B4C;
    align-items: center;
    justify-content: center;
  }
}

</style>
