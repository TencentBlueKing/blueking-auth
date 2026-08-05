<template>
  <BkSideslider
    :is-show="isShow"
    title="令牌详情"
    :width="640"
    quick-close
    @update:is-show="handleClose"
  >
    <template #default>
      <div
        v-if="token"
        class="token-detail"
      >
        <div class="token-detail__row">
          <span class="token-detail__label">名称</span>
          <span class="token-detail__value">{{ token.name }}</span>
        </div>
        <div class="token-detail__row">
          <span class="token-detail__label">描述</span>
          <span class="token-detail__value">{{ token.description || '--' }}</span>
        </div>
        <div class="token-detail__row">
          <span class="token-detail__label">Token</span>
          <span class="token-detail__value token-detail__mask">{{ token.token_mask }}</span>
        </div>
        <div class="token-detail__row">
          <span class="token-detail__label">状态</span>
          <span class="token-detail__value">
            <BkTag :theme="statusMeta.theme">{{ statusMeta.label }}</BkTag>
          </span>
        </div>

        <div class="token-detail__row token-detail__row--top">
          <span class="token-detail__label">授权资源</span>
          <div class="token-detail__value">
            <template v-if="audienceGroups.length">
              <div
                v-for="group in audienceGroups"
                :key="group.type"
                class="token-detail__aud-group"
              >
                <div class="token-detail__aud-title">
                  {{ group.typeLabel }}（{{ group.names.length }}）
                </div>
                <div class="token-detail__aud-items">
                  <BkTag
                    v-for="name in group.names"
                    :key="name"
                  >
                    {{ name }}
                  </BkTag>
                </div>
              </div>
            </template>
            <span v-else>--</span>
          </div>
        </div>

        <div class="token-detail__row">
          <span class="token-detail__label">过期时间</span>
          <span class="token-detail__value">{{ formatDateTime(token.expires_at) }}</span>
        </div>
        <div class="token-detail__row">
          <span class="token-detail__label">最近使用</span>
          <span class="token-detail__value">{{ formatDateTime(token.last_used_at) }}</span>
        </div>
        <div class="token-detail__row">
          <span class="token-detail__label">创建时间</span>
          <span class="token-detail__value">{{ formatDateTime(token.created_at) }}</span>
        </div>
        <div class="token-detail__row">
          <span class="token-detail__label">更新时间</span>
          <span class="token-detail__value">{{ formatDateTime(token.updated_at) }}</span>
        </div>
        <div
          v-if="token.revoked_at"
          class="token-detail__row"
        >
          <span class="token-detail__label">撤销时间</span>
          <span class="token-detail__value">{{ formatDateTime(token.revoked_at) }}</span>
        </div>
      </div>
    </template>
  </BkSideslider>
</template>

<script setup lang="ts">
import type { PersonalToken } from '@/services/source/personalToken';
import { STATUS_META, formatDateTime, getDisplayStatus, parseAudience } from '../utils';

const { token } = defineProps<{
  isShow: boolean
  token: PersonalToken | null
}>();

const emit = defineEmits<{ 'update:isShow': [value: boolean] }>();

const TYPE_LABELS: Record<string, string> = {
  service: '服务',
  gateway: '网关',
  api: 'API',
};

const statusMeta = computed(() =>
  (token ? STATUS_META[getDisplayStatus(token)] : STATUS_META.normal));

const audienceGroups = computed(() => {
  const map = new Map<string, string[]>();
  (token?.audience ?? []).forEach((aud) => {
    const { type, name } = parseAudience(aud);
    const list = map.get(type) ?? [];
    list.push(name);
    map.set(type, list);
  });
  return Array.from(map.entries()).map(([type, names]) => ({
    type,
    typeLabel: TYPE_LABELS[type] || type || '资源',
    names,
  }));
});

const handleClose = (value: boolean) => {
  emit('update:isShow', value);
};
</script>

<style scoped lang="scss">
.token-detail {
  padding: 20px 24px;

  &__row {
    display: flex;
    align-items: center;
    padding: 10px 0;
    border-bottom: 1px solid #f0f1f5;

    &--top {
      align-items: flex-start;
    }
  }

  &__label {
    flex-shrink: 0;
    width: 90px;
    color: #979ba5;
  }

  &__value {
    color: #313238;
    word-break: break-all;
  }

  &__mask {
    font-family: monospace;
  }

  &__aud-group {
    margin-bottom: 10px;
  }

  &__aud-title {
    margin-bottom: 6px;
    font-size: 12px;
    color: #63656e;
  }

  &__aud-items {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
}
</style>
