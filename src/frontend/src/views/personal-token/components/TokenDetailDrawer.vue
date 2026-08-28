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
          <div
            v-for="panel in resourcePanels"
            :key="panel.type"
            class="resource-panel"
          >
            <div
              class="panel-header"
              @click="toggleResourcePanel(panel.type)"
            >
              <span
                class="panel-arrow"
                :class="{ 'is-collapsed': !resourceExpanded[panel.type] }"
              />
              <span class="panel-title">{{ panel.displayName }} ( {{ panel.count }} )</span>
            </div>
            <div
              v-show="resourceExpanded[panel.type]"
              class="panel-body"
            >
              <div
                v-for="(group, groupIndex) in panel.groups"
                :key="group.level"
                class="resource-group"
              >
                <div
                  v-if="group.displayName"
                  class="group-title"
                  :class="{ 'mt-16px': groupIndex > 0 }"
                >
                  【{{ group.displayName }}】- 共 <em>{{ group.items.length }}</em> 个
                </div>
                <div
                  v-for="item in group.items"
                  :key="item.audience"
                  class="resource-item"
                >
                  <BkTag
                    v-if="group.displayName"
                    class="item-tag"
                    size="small"
                    :theme="getLevelTheme(group.level)"
                  >
                    {{ group.displayName }}
                  </BkTag>
                  <BkTag
                    v-if="item.extras?.is_official === true"
                    class="item-tag"
                    size="small"
                    theme="info"
                  >
                    官方
                  </BkTag>
                  <BkTag
                    v-if="isNonPublicResource(item)"
                    class="item-tag"
                    size="small"
                    theme="danger"
                  >
                    非公开
                  </BkTag>
                  <span
                    v-bk-ellipsis
                    class="item-text"
                  >{{ item.display_name }} ( {{ item.name }} )</span>
                </div>
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

      <!-- 创建时间 -->
      <div class="info-row">
        <span class="info-label">创建时间：</span>
        <span class="info-value">{{ createdAtText }}</span>
      </div>
    </div>

    <!-- 续期弹窗 -->
    <RenewDialog
      v-model:is-show="renewDialogShow"
      :realm="realm"
      :token="detail || token"
      @success="handleRenewSuccess"
    />
  </BkSideslider>
</template>

<script setup lang="ts">
import RenewDialog from './RenewDialog.vue';
import type { PersonalTokenRealm } from '@/constants/personal-token';
import { messageSuccess } from '@/utils';
import { usePopInfoBox } from '@/hooks';
import {
  type IGrantableResourceType,
  type IPersonalToken,
  type IPersonalTokenResource,
  getGrantableResourceTypes,
  getPersonalTokenDetail,
  revokePersonalToken,
} from '@/services/source/personal-token';
import {
  type TokenStatus,
  formatUnixSeconds,
  getPersonalTokenStatus,
  getRemainDays,
} from '../utils';

type TagTheme = 'info' | 'warning' | undefined;

interface IProps {
  realm: PersonalTokenRealm
  token?: IPersonalToken | null
}

interface IEmits {
  edit: [token: IPersonalToken]
  updated: []
}

interface IResourceGroup {
  level: string
  displayName: string
  items: IPersonalTokenResource[]
}

interface IResourcePanel {
  type: string
  displayName: string
  count: number
  groups: IResourceGroup[]
}

const isShow = defineModel<boolean>('isShow', { default: false });

const {
  realm,
  token = null,
} = defineProps<IProps>();

const emit = defineEmits<IEmits>();

const { t } = useI18n();

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

const detail = ref<IPersonalToken | null>(null);
const resourceTypes = ref<IGrantableResourceType[]>([]);
const resourceExpanded = ref<Record<string, boolean>>({});
const loading = ref(false);
const renewDialogShow = ref(false);
// 标识最新详情请求，防止旧响应覆盖当前令牌
let detailRequestId = 0;

const displayName = computed(() => detail.value?.name ?? token?.name ?? '');

const currentStatus = computed(() => {
  const currentToken = detail.value ?? token;
  return currentToken ? getPersonalTokenStatus(currentToken) : undefined;
});

const statusInfo = computed(() => (currentStatus.value ? statusConfig[currentStatus.value] : null));

// 将详情资源按类型和层级整理为折叠面板数据
const resourcePanels = computed<IResourcePanel[]>(() => {
  const data = detail.value;
  if (!data) {
    return [];
  }
  if (data.resources === null) {
    return [{
      type: 'raw-audience',
      displayName: '授权资源',
      count: data.audience.length,
      groups: [{
        level: '',
        displayName: '',
        items: data.audience.map(audience => ({
          type: '',
          level: '',
          name: audience,
          display_name: audience,
          audience,
        })),
      }],
    }];
  }

  const typeNames = [
    ...resourceTypes.value
      .map(item => item.name)
      .filter(typeName => data.resources?.some(resource => resource.type === typeName)),
    ...data.resources
      .map(resource => resource.type)
      .filter(typeName => !resourceTypes.value.some(item => item.name === typeName)),
  ];
  return [...new Set(typeNames)].map((typeName) => {
    const items = data.resources?.filter(resource => resource.type === typeName) ?? [];
    const resourceType = resourceTypes.value.find(item => item.name === typeName);
    const knownLevels = resourceType?.levels ?? [];
    const unknownLevels = [...new Set(items.map(item => item.level).filter(Boolean))]
      .filter(level => !knownLevels.some(item => item.name === level))
      .map(level => ({
        name: level,
        display_name: level,
      }));
    const levels = [
      {
        name: '',
        display_name: 'ALL',
      },
      ...knownLevels,
      ...unknownLevels,
    ];
    return {
      type: typeName,
      displayName: resourceType?.display_name ?? typeName,
      count: items.length,
      groups: levels
        .map(level => ({
          level: level.name,
          displayName: level.display_name,
          items: items.filter(item => item.level === level.name),
        }))
        .filter(group => group.items.length > 0),
    };
  });
});

const editDisabled = computed(() => currentStatus.value === 'revoked');

const renewDisabled = computed(() => currentStatus.value === 'revoked');

const revokeDisabled = computed(() => currentStatus.value === 'revoked');

const createdAtText = computed(() => formatUnixSeconds(detail.value?.created_at));

const expiredText = computed(() => {
  const data = detail.value;
  if (!data) {
    return '--';
  }
  const expiredAtText = formatUnixSeconds(data.expires_at);
  if (currentStatus.value === 'valid') {
    const remainDays = getRemainDays(data.expires_at);
    if (remainDays >= 0 && remainDays <= EXPIRING_THRESHOLD_DAYS) {
      return expiredAtText + '（' + remainDays + ' 天后过期）';
    }
  }
  return expiredAtText;
});

const expiredClass = computed(() => {
  const data = detail.value;
  if (!data) {
    return 'info-value';
  }
  if (currentStatus.value === 'expired') {
    return 'text-expired';
  }
  if (currentStatus.value === 'valid') {
    const remainDays = getRemainDays(data.expires_at);
    if (remainDays >= 0 && remainDays <= EXPIRING_THRESHOLD_DAYS) {
      return 'text-expiring';
    }
  }
  return 'info-value';
});

// 并行加载令牌详情和资源元数据，仅接收当前抽屉的最新请求
const fetchDetail = async () => {
  if (!token?.id) {
    return;
  }
  const tokenId = token.id;
  const currentRealm = realm;
  const requestId = detailRequestId + 1;
  detailRequestId = requestId;
  loading.value = true;
  try {
    const [tokenDetail, types] = await Promise.all([
      getPersonalTokenDetail(currentRealm, tokenId),
      getGrantableResourceTypes(currentRealm),
    ]);
    if (
      detailRequestId === requestId
      && isShow.value
      && token?.id === tokenId
      && realm === currentRealm
    ) {
      detail.value = tokenDetail;
      resourceTypes.value = types;
      resourceExpanded.value = Object.fromEntries(
        resourcePanels.value.map(panel => [panel.type, true]),
      );
    }
  }
  finally {
    if (detailRequestId === requestId) {
      loading.value = false;
    }
  }
};

const handleEdit = () => {
  const currentToken = detail.value ?? token;
  if (currentToken) {
    emit('edit', currentToken);
  }
};

const toggleResourcePanel = (type: string) => {
  resourceExpanded.value[type] = !resourceExpanded.value[type];
};

const getLevelTheme = (level: string): TagTheme => {
  if (level === 'gateway') {
    return 'warning';
  }
  return level ? 'info' : undefined;
};

const isNonPublicResource = (resource: IPersonalTokenResource) =>
  Object.prototype.hasOwnProperty.call(resource.extras ?? {}, 'is_public')
  && resource.extras?.is_public === false;

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
    subTitle: t('撤销后令牌【{name}】将在 5 分钟内失效且无法恢复。', { name: token.name }),
    confirmText: '撤销',
    cancelText: '取消',
    onConfirm: async () => {
      await revokePersonalToken(realm, token.id);
      messageSuccess(t('撤销成功，5 分钟内失效'));
      emit('updated');
      isShow.value = false;
    },
  });
};

const handleClosed = () => {
  // 使抽屉关闭前尚未完成的请求失效
  detailRequestId += 1;
  loading.value = false;
  detail.value = null;
  resourceTypes.value = [];
  resourceExpanded.value = {};
};

watch(
  [isShow, () => realm],
  ([visible]) => {
    if (visible) {
      detail.value = null;
      resourceTypes.value = [];
      resourceExpanded.value = {};
      fetchDetail();
    }
  },
);

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
            cursor: pointer;
          }
        }
      }
    }
  }
}
</style>
