<template>
  <BkSideslider
    v-model:is-show="isShow"
    class="token-detail-drawer"
    :width="960"
    quick-close
    @closed="handleClosed"
  >
    <template #header>
      <div class="detail-header">
        <div class="header-left">
          <span class="header-title">令牌详情</span>
          <span class="header-divider">|</span>
          <span class="header-name">{{ displayName }}</span>
        </div>
        <div class="header-actions">
          <BkButton
            :disabled="editDisabled"
            @click="handleEdit"
          >
            编辑
          </BkButton>
          <BkButton
            :disabled="renewDisabled"
            @click="handleRenew"
          >
            续期
          </BkButton>
          <BkButton
            :disabled="revokeDisabled"
            @click="handleRevoke"
          >
            撤销
          </BkButton>
        </div>
      </div>
    </template>

    <div
      v-bkloading="{ loading }"
      class="detail-body"
    >
      <!-- 名称 -->
      <div class="info-row">
        <span class="info-label">名称：</span>
        <span class="info-value">{{ detail?.name || '--' }}</span>
      </div>

      <!-- 备注 -->
      <div class="info-row">
        <span class="info-label">备注：</span>
        <span class="info-value">{{ detail?.description || '--' }}</span>
      </div>

      <!-- 授权资源 -->
      <div class="info-row is-top">
        <span class="info-label">授权资源：</span>
        <div class="resource-panels">
          <!-- MCP 面板 -->
          <div class="resource-panel">
            <div
              class="panel-header"
              @click="mcpExpanded = !mcpExpanded"
            >
              <span
                class="panel-arrow"
                :class="{ 'is-collapsed': !mcpExpanded }"
              />
              <span class="panel-title">MCP ( {{ mcpResources.length }} )</span>
            </div>
            <div
              v-show="mcpExpanded"
              class="panel-body"
            >
              <div class="group-title">
                【MCP】- 共 <em>{{ mcpResources.length }}</em> 个
              </div>
              <div
                v-for="(item, index) in mcpResources"
                :key="index"
                class="resource-item"
              >
                <BkTag
                  class="item-tag"
                  size="small"
                >
                  MCP
                </BkTag>
                <span class="item-text">{{ item.name }} ( {{ item.id }} )</span>
              </div>
            </div>
          </div>

          <!-- API 面板 -->
          <div class="resource-panel">
            <div
              class="panel-header"
              @click="apiExpanded = !apiExpanded"
            >
              <span
                class="panel-arrow"
                :class="{ 'is-collapsed': !apiExpanded }"
              />
              <span class="panel-title">API ( {{ gatewayResources.length + apiResources.length }} )</span>
            </div>
            <div
              v-show="apiExpanded"
              class="panel-body"
            >
              <div class="group-title">
                【网关】- 共 <em>{{ gatewayResources.length }}</em> 个
              </div>
              <div
                v-for="(item, index) in gatewayResources"
                :key="`gw-${index}`"
                class="resource-item"
              >
                <BkTag
                  class="item-tag"
                  size="small"
                  theme="warning"
                >
                  网关
                </BkTag>
                <BkTag
                  v-if="item.is_official"
                  class="item-tag"
                  size="small"
                  theme="info"
                >
                  官方
                </BkTag>
                <span class="item-text">{{ item.name }}</span>
              </div>

              <div class="group-title mt-16px">
                【API】- 共 <em>{{ apiResources.length }}</em> 个
              </div>
              <div
                v-for="(item, index) in apiResources"
                :key="`api-${index}`"
                class="resource-item"
              >
                <BkTag
                  class="item-tag"
                  size="small"
                  theme="info"
                >
                  API
                </BkTag>
                <BkTag
                  v-if="!item.is_public"
                  class="item-tag"
                  size="small"
                  theme="danger"
                >
                  非公开
                </BkTag>
                <span class="item-text">{{ item.name }} ( {{ item.action }} )</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 状态 -->
      <div class="info-row">
        <span class="info-label">状态：</span>
        <BkTag :theme="statusInfo?.theme || undefined">
          {{ statusInfo?.text || '--' }}
        </BkTag>
      </div>

      <!-- 过期时间 -->
      <div class="info-row">
        <span class="info-label">过期时间：</span>
        <span :class="expiredClass">{{ expiredText }}</span>
      </div>

      <!-- 最近使用 -->
      <div class="info-row">
        <span class="info-label">最近使用：</span>
        <span class="info-value">{{ detail?.last_used_at || '--' }}</span>
      </div>

      <!-- 创建时间 -->
      <div class="info-row">
        <span class="info-label">创建时间：</span>
        <span class="info-value">{{ detail?.created_at || '--' }}</span>
      </div>
    </div>

    <!-- 续期弹窗 -->
    <RenewDialog
      v-model:is-show="renewDialogShow"
      :token="token"
      @success="handleRenewSuccess"
    />
  </BkSideslider>
</template>

<script setup lang="ts">
import RenewDialog from './RenewDialog.vue';
import { messageSuccess } from '@/utils';
import { usePopInfoBox } from '@/hooks';
import {
  type PersonalTokenDetail,
  type PersonalTokenItem,
  type TokenStatus,
  getPersonalTokenDetail,
  revokePersonalToken,
} from '@/services/source/personal-token';

const isShow = defineModel<boolean>('isShow', { default: false });

const { token = null } = defineProps<{ token?: PersonalTokenItem | null }>();

const emit = defineEmits<{
  edit: [token: PersonalTokenItem]
  updated: []
}>();

// 临期阈值（天）
const EXPIRING_THRESHOLD_DAYS = 7;

const statusConfig: Record<TokenStatus, {
  text: string
  theme: 'success' | 'danger' | ''
}> = {
  valid: {
    text: '有效',
    theme: 'success',
  },
  expired: {
    text: '已过期',
    theme: 'danger',
  },
  revoked: {
    text: '已撤销',
    theme: '',
  },
};

const detail = ref<PersonalTokenDetail | null>(null);
const loading = ref(false);
const mcpExpanded = ref(true);
const apiExpanded = ref(true);
const renewDialogShow = ref(false);

const displayName = computed(() => detail.value?.name ?? token?.name ?? '');
const currentStatus = computed(() => detail.value?.status ?? token?.status);
const statusInfo = computed(() => (currentStatus.value ? statusConfig[currentStatus.value] : null));

const mcpResources = computed(() => detail.value?.mcp_resources ?? []);
const gatewayResources = computed(() => detail.value?.gateway_resources ?? []);
const apiResources = computed(() => detail.value?.api_resources ?? []);

const editDisabled = computed(() => currentStatus.value === 'expired' || currentStatus.value === 'revoked');
const renewDisabled = computed(() => currentStatus.value === 'expired' || currentStatus.value === 'revoked');
const revokeDisabled = computed(() => currentStatus.value === 'revoked');

const getRemainDays = (expiredAt: string) => {
  const diff = new Date(expiredAt.replace(/-/g, '/')).getTime() - Date.now();
  return Math.ceil(diff / (24 * 60 * 60 * 1000));
};

const expiredText = computed(() => {
  const data = detail.value;
  if (!data) {
    return '--';
  }
  if (data.status === 'valid') {
    const remainDays = getRemainDays(data.expired_at);
    if (remainDays >= 0 && remainDays <= EXPIRING_THRESHOLD_DAYS) {
      return `${data.expired_at}（${remainDays} 天后过期）`;
    }
  }
  return data.expired_at;
});

const expiredClass = computed(() => {
  const data = detail.value;
  if (!data) {
    return 'info-value';
  }
  if (data.status === 'expired') {
    return 'text-expired';
  }
  if (data.status === 'valid') {
    const remainDays = getRemainDays(data.expired_at);
    if (remainDays >= 0 && remainDays <= EXPIRING_THRESHOLD_DAYS) {
      return 'text-expiring';
    }
  }
  return 'info-value';
});

const fetchDetail = async () => {
  if (!token?.id) {
    return;
  }
  loading.value = true;
  try {
    detail.value = await getPersonalTokenDetail(token.id);
  }
  finally {
    loading.value = false;
  }
};

const handleEdit = () => {
  if (token) {
    emit('edit', token);
  }
};

const handleRenew = () => {
  renewDialogShow.value = true;
};

const handleRenewSuccess = () => {
  fetchDetail();
  emit('updated');
};

const handleRevoke = () => {
  if (!token) {
    return;
  }
  usePopInfoBox({
    type: 'warning',
    isShow: true,
    title: '确认撤销该个人令牌？',
    subTitle: `撤销后令牌 ${token.name} 将立即失效且无法恢复。`,
    confirmText: '撤销',
    cancelText: '取消',
    onConfirm: async () => {
      await revokePersonalToken(token.id);
      messageSuccess('撤销成功');
      emit('updated');
      isShow.value = false;
    },
  });
};

const handleClosed = () => {
  detail.value = null;
};

watch(isShow, (val) => {
  if (val) {
    detail.value = null;
    mcpExpanded.value = true;
    apiExpanded.value = true;
    fetchDetail();
  }
});

defineExpose({ refresh: fetchDetail });
</script>

<style scoped lang="scss">
.detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding-right: 24px;

  .header-left {
    display: flex;
    align-items: center;
    font-size: 16px;

    .header-title {
      color: #313238;
    }

    .header-divider {
      margin: 0 10px;
      color: #dcdee5;
    }

    .header-name {
      font-size: 14px;
      color: #979ba5;
    }
  }

  .header-actions {
    display: flex;
    gap: 8px;

    :deep(.bk-button) {
      width: 48px;
      height: 26px;
      padding: 0;
    }

    :deep(.bk-button-text) {
      font-size: 12px;
    }
  }
}

.detail-body {
  min-height: 200px;
  padding: 24px 40px;

  .info-row {
    display: flex;
    align-items: center;
    margin-bottom: 20px;
    font-size: 14px;

    &.is-top {
      align-items: flex-start;
    }

    .info-label {
      flex-shrink: 0;
      width: 72px;
      color: #979ba5;
      text-align: right;
    }

    .info-value {
      color: #313238;
    }

    .text-expired {
      color: #ea3636;
    }

    .text-expiring {
      color: #ff9c01;
    }
  }

  .resource-panels {
    display: flex;
    flex: 1;
    gap: 16px;
    min-width: 0;

    .resource-panel {
      flex: 1;
      min-width: 0;
      font-size: 12px;

      .panel-header {
        display: flex;
        align-items: center;
        height: 32px;
        padding: 0 12px;
        cursor: pointer;
        background-color: #f0f1f5;
        border-radius: 2px;

        .panel-arrow {
          width: 0;
          height: 0;
          margin-right: 8px;
          border-top: 6px solid #63656e;
          border-right: 5px solid transparent;
          border-left: 5px solid transparent;
          transition: transform 0.2s;

          &.is-collapsed {
            transform: rotate(-90deg);
          }
        }

        .panel-title {
          font-weight: 700;
          color: #313238;
        }
      }

      .panel-body {
        padding-top: 12px;

        .group-title {
          margin-bottom: 8px;
          color: #63656e;

          em {
            font-style: normal;
            color: #3a84ff;
          }
        }

        .resource-item {
          display: flex;
          align-items: center;
          height: 32px;
          padding: 0 12px;
          margin-bottom: 8px;
          background-color: #f5f7fa;
          border-radius: 2px;

          .item-tag {
            flex-shrink: 0;
            margin-right: 8px;
          }

          .item-text {
            overflow: hidden;
            color: #63656e;
            text-overflow: ellipsis;
            white-space: nowrap;
          }
        }
      }
    }
  }
}
</style>
