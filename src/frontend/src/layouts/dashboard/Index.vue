<template>
  <BkNavigation
    class="navigation-content"
    navigation-type="top-bottom"
    :need-menu="false"
    default-open
  >
    <template #side-header>
      <div
        class="flex items-center gap-16px"
      >
        <div>
          <img
            :src="LogoWithoutTitle"
            alt="Bk Auth"
            class="max-w-none h-28px cursor-pointer"
          >
        </div>
        <div class="text-16px font-bold color-#eaebf0 cursor-pointer">
          蓝鲸认证中心
        </div>
      </div>
    </template>
    <template #header>
      <div class="flex items-center flex flex-1 gap-40px">
        <div class="px-12px text-14px color-#fff cursor-pointer">
          令牌管理
        </div>
      </div>
      <div class="header-aside-wrap">
        <UserInfo />
      </div>
    </template>
    <div class="content">
      <RouterView />
    </div>
  </BkNavigation>
</template>

<script setup lang="ts">
import LogoWithoutTitle from '@/assets/APIgateway-logo.png';
import UserInfo from '@/components/user-info/Index.vue';
import { useEnv, useUserInfo } from '@/stores';

const userInfoStore = useUserInfo();
const envStore = useEnv();

userInfoStore.fetchUserInfo();
envStore.fetchEnv();
</script>

<style scoped lang="scss">
.navigation-content {

  :deep(.bk-navigation-header) {
    z-index: 999;
    overflow: visible;
  }

  :deep(.bk-navigation-wrapper) {

    .container-content {
      // 最小宽度应为 1280px 减去左侧菜单栏展开时的宽度 260px，即为 1020px
      min-width: 1020px;
      padding: 0 !important;
    }
  }
}

.header-aside-wrap {
  color: #fff;
}

.content {
  height: 100%;
}
</style>
