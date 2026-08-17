<template>
  <div
    v-if="userLoaded"
    class="bg-[#F5F7FA]"
  >
    <RouterView />
  </div>
  <div class="global-footer">
    Copyright © 2026 Tencent BlueKing. All Rights Reserved. 3.0
  </div>
</template>

<script setup lang="ts">
import { useEnv, useUserInfo } from '@/stores';

const route = useRoute();
const userInfoStore = useUserInfo();
const envStore = useEnv();

const userLoaded = ref(false);

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

<style scoped lang="scss">

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

</style>
