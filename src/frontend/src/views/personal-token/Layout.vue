<template>
  <div class="console">
    <header class="console__header">
      <div class="console__brand">
        蓝鲸认证中心
        <span class="console__brand-sub">令牌管理 · {{ realm }}</span>
      </div>
      <div class="console__user">
        <AgIcon
          name="user-circle"
          size="16"
          color="#c4c6cc"
          class="mr-4px"
        />
        {{ userInfoStore.info?.username || '--' }}
      </div>
    </header>

    <div class="console__body">
      <nav class="console__nav">
        <div
          v-for="item in navItems"
          :key="item.key"
          class="console__nav-item"
          :class="{
            'is-active': item.active,
            'is-disabled': item.disabled,
          }"
          :title="item.disabled ? '暂未开放' : ''"
        >
          <span>{{ item.label }}</span>
          <span
            v-if="item.disabled"
            class="console__nav-badge"
          >待开放</span>
        </div>
      </nav>

      <main class="console__content">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { PERSONAL_TOKEN_REALM } from '@/services/source/personalToken';
import { useUserInfo } from '@/stores';

const userInfoStore = useUserInfo();

const route = useRoute();
const realm = computed(() => String(route.params.realm || PERSONAL_TOKEN_REALM));

// 一期仅「个人令牌」可用，另两个模块置灰待开放。
const navItems = [
  {
    key: 'personal-token',
    label: '个人令牌',
    active: true,
    disabled: false,
  },
  {
    key: 'public-client',
    label: '公开客户端授权管理',
    active: false,
    disabled: true,
  },
  {
    key: 'app-auth',
    label: '应用授权管理',
    active: false,
    disabled: true,
  },
];
</script>

<style scoped lang="scss">
.console {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 48px);
  background: #f5f7fa;

  &__header {
    display: flex;
    flex-shrink: 0;
    align-items: center;
    justify-content: space-between;
    height: 52px;
    padding: 0 24px;
    color: #fff;
    background: #182132;
  }

  &__brand {
    font-size: 16px;
    font-weight: 700;

    &-sub {
      margin-left: 10px;
      font-size: 13px;
      font-weight: 400;
      color: #c4c6cc;
    }
  }

  &__user {
    display: flex;
    align-items: center;
    font-size: 13px;
    color: #dcdee5;
  }

  &__body {
    display: flex;
    flex: 1;
    overflow: hidden;
  }

  &__nav {
    flex-shrink: 0;
    width: 220px;
    padding: 12px 0;
    background: #fff;
    border-right: 1px solid #f0f1f5;
  }

  &__nav-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 42px;
    padding: 0 20px;
    color: #63656e;
    cursor: pointer;

    &.is-active {
      color: #3a84ff;
      background: #e1ecff;
    }

    &.is-disabled {
      color: #c4c6cc;
      cursor: not-allowed;
    }
  }

  &__nav-badge {
    padding: 0 6px;
    font-size: 12px;
    line-height: 18px;
    color: #979ba5;
    background: #f0f1f5;
    border-radius: 2px;
  }

  &__content {
    flex: 1;
    overflow: auto;
  }
}
</style>
