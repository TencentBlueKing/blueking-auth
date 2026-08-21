<template>
  <div class="personal-token">
    <!-- 顶部提示 -->
    <BkAlert
      theme="info"
      class="mb-16px"
    >
      <template #title>
        管理你个人身份签发的访问令牌（AccessToken），令牌明文仅在创建时展示一次，后续无法查看
      </template>
    </BkAlert>

    <!-- 工具栏 -->
    <div class="toolbar mb-16px">
      <div class="toolbar-left">
        <span
          v-bk-tooltips="{
            content: t('活跃令牌数量已达上限，请先撤销一个现有令牌。'),
            disabled: !isActiveTokenLimitReached,
          }"
          class="create-button-wrapper"
        >
          <BkButton
            theme="primary"
            :disabled="isActiveTokenLimitReached"
            @click="handleCreate"
          >
            新增个人令牌
          </BkButton>
        </span>
        <span
          class="active-token-count"
          :class="{ 'is-limit-reached': isActiveTokenLimitReached }"
        >
          {{ t('活跃令牌：{count} / {limit}', {
            count: activeTokenCount,
            limit: maxActivePerUser,
          }) }}
        </span>
      </div>

      <div class="toolbar-right">
        <BkSelect
          v-model="filterStatus"
          class="status-select"
          placeholder="状态"
          clearable
          multiple
          @change="handleToolbarStatusChange"
        >
          <BkOption
            v-for="option in statusOptions"
            :id="option.value"
            :key="option.value"
            :name="option.label"
          />
        </BkSelect>
        <BkInput
          v-model="searchKey"
          class="search-input"
          :placeholder="t('搜索令牌名称、备注、授权资源')"
          type="search"
          clearable
          @enter="handleSearch"
          @clear="handleSearch"
        />
      </div>
    </div>

    <!-- 表格 -->
    <CommonTable
      ref="tableRef"
      :api-method="fetchPersonalTokenTableData"
      :columns="columns"
      :filter-value="tableFilterValue"
      :sort="tableSort"
      @filter-change="handleTableFilterChange"
      @sort-change="handleTableSortChange"
      @clear-filter="handleClearFilter"
    />

    <!-- 新建/编辑令牌侧滑 -->
    <TokenCreateDrawer
      v-model:is-show="createDrawerShow"
      :realm="realm"
      :token="currentRow"
      @success="handleTokenUpdated"
    />

    <!-- 令牌详情侧滑 -->
    <TokenDetailDrawer
      v-model:is-show="detailDrawerShow"
      :realm="realm"
      :token="currentRow"
      @edit="handleEdit"
      @updated="handleTokenUpdated"
    />

    <!-- 续期弹窗 -->
    <RenewDialog
      v-model:is-show="renewDialogShow"
      :realm="realm"
      :token="currentRow"
      @success="handleTokenUpdated"
    />
  </div>
</template>

<script setup lang="tsx">
import {
  Button as BkButton,
  Tag as BkTag,
} from 'bkui-vue';
import CommonTable from '@/components/common-table/Index.vue';
import RenewDialog from './components/RenewDialog.vue';
import TokenCreateDrawer from './components/TokenCreateDrawer.vue';
import TokenDetailDrawer from './components/TokenDetailDrawer.vue';
import {
  DEFAULT_PERSONAL_TOKEN_REALM,
  type PersonalTokenRealm,
  isPersonalTokenRealm,
} from '@/constants/personal-token';
import { messageSuccess } from '@/utils';
import {
  type IGrantableResourceType,
  type IPersonalToken,
  getGrantableResourceTypes,
  getPersonalTokenList,
  revokePersonalToken,
} from '@/services/source/personal-token';
import { usePopInfoBox } from '@/hooks';
import { useEnv } from '@/stores';
import {
  SECONDS_PER_DAY,
  SECONDS_PER_HOUR,
  SECONDS_PER_MINUTE,
  type TokenStatus,
  formatUnixSeconds,
  getPersonalTokenStatus,
} from './utils';

type TableInstance = InstanceType<typeof CommonTable>;

interface IPersonalTokenTableItem extends IPersonalToken { status: TokenStatus }

interface ITableQuery {
  offset?: number
  limit?: number
  statuses?: TokenStatus[]
  keyword?: string
}

interface ITableSort {
  sortBy: string
  descending: boolean
}

interface ITableFilterValue {
  status?: unknown
  [key: string]: unknown
}

type TableSortValue = ITableSort | ITableSort[] | undefined;

const envStore = useEnv();

const { t } = useI18n();
const route = useRoute();

const searchKey = ref('');
const filterStatus = ref<TokenStatus[]>([]);
const activeTokenCount = ref(0);
// 表头和工具栏筛选最终同步到该查询状态
const tableFilterStatuses = ref<TokenStatus[]>([]);
const tableSort = ref<ITableSort>();
// 资源类型元数据用于控制授权资源的展示顺序和名称
const resourceTypes = ref<IGrantableResourceType[]>([]);

// 新建/编辑侧滑
const createDrawerShow = ref(false);

// 详情侧滑
const detailDrawerShow = ref(false);

// 续期弹窗
const renewDialogShow = ref(false);
// 三类弹窗共用当前操作的令牌上下文
const currentRow = ref<IPersonalToken | null>(null);

const tableRef = useTemplateRef<TableInstance>('tableRef');

const realm = computed<PersonalTokenRealm>(() => (
  isPersonalTokenRealm(route.params.realm)
    ? route.params.realm
    : DEFAULT_PERSONAL_TOKEN_REALM
));
const tableFilterValue = computed<ITableFilterValue>(() => ({ status: tableFilterStatuses.value }));
const maxActivePerUser = computed(() => envStore.env.personal_token_policy.max_active_per_user);
const isActiveTokenLimitReached = computed(() => activeTokenCount.value >= maxActivePerUser.value);

const handleTokenUpdated = () => {
  tableRef.value?.fetchData(
    {
      statuses: tableFilterStatuses.value,
      keyword: searchKey.value,
    },
    { resetPage: false },
  );
};

const statusOptions: {
  value: TokenStatus
  label: string
}[] = [
  {
    value: 'valid',
    label: '有效',
  },
  {
    value: 'expired',
    label: '已过期',
  },
  {
    value: 'revoked',
    label: '已撤销',
  },
];

// 临期阈值（天）
const EXPIRING_THRESHOLD_DAYS = 7;

// 状态展示配置
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

// 将接口全量数据按当前条件筛选、排序并分页为表格数据
const fetchPersonalTokenTableData = async (params: ITableQuery = {}) => {
  const currentRealm = realm.value;
  const {
    offset = 0,
    limit = 10,
    statuses = [],
    keyword = '',
  } = params;
  const tokens = await getPersonalTokenList(currentRealm);
  const normalizedKeyword = keyword.trim().toLowerCase();
  const normalizedTokens = tokens.map<IPersonalTokenTableItem>(token => ({
    ...token,
    status: getPersonalTokenStatus(token),
  }));
  if (realm.value === currentRealm) {
    activeTokenCount.value = normalizedTokens.filter(token => token.status === 'valid').length;
  }
  const filteredTokens = normalizedTokens
    .filter(token => !statuses.length || statuses.includes(token.status))
    .filter((token) => {
      if (!normalizedKeyword) {
        return true;
      }
      const resourceTexts = token.resources?.flatMap(resource => [
        resource.name,
        resource.display_name,
        resource.audience,
      ]) ?? [];
      return [
        token.name,
        token.description,
        ...token.audience,
        ...resourceTexts,
      ].some(text => text.toLowerCase().includes(normalizedKeyword));
    });
  const sortedTokens = [...filteredTokens];
  if (tableSort.value?.sortBy === 'expires_at') {
    const direction = tableSort.value.descending ? -1 : 1;
    sortedTokens.sort((first, second) =>
      (first.expires_at - second.expires_at) * direction || second.id - first.id);
  }

  return {
    count: sortedTokens.length,
    results: sortedTokens.slice(offset, offset + limit),
  };
};

const fetchResourceTypeMetadata = async (currentRealm: PersonalTokenRealm) => {
  const result = await getGrantableResourceTypes(currentRealm);
  if (realm.value === currentRealm) {
    resourceTypes.value = result;
  }
};

// 按资源类型和层级聚合授权资源标签
const renderResource = (token: IPersonalToken) => {
  if (token.resources === null) {
    return (
      <div class="resource-cell">
        { token.audience.map(audience => (
          <div key={audience} class="resource-row">
            <BkTag class="resource-count-tag">{ audience }</BkTag>
          </div>
        )) }
      </div>
    );
  }

  const resourceTypeNames = [
    ...resourceTypes.value
      .map(item => item.name)
      .filter(typeName => token.resources?.some(resource => resource.type === typeName)),
    ...token.resources
      .map(resource => resource.type)
      .filter(typeName => !resourceTypes.value.some(item => item.name === typeName)),
  ];

  return (
    <div class="resource-cell">
      { [...new Set(resourceTypeNames)].map((typeName) => {
        const typeResources = token.resources?.filter(resource => resource.type === typeName) ?? [];
        const resourceType = resourceTypes.value.find(item => item.name === typeName);
        const typeDisplayName = resourceType?.display_name ?? typeName;
        const fallbackLevels = [...new Set(typeResources.map(item => item.level).filter(Boolean))]
          .map(level => ({
            name: level,
            display_name: level,
          }));
        const levels = resourceType?.levels ?? fallbackLevels;
        const allCount = typeResources.filter(resource => !resource.level).length;

        return (
          <div key={typeName} class="resource-row">
            <span class="resource-label">{ typeDisplayName }</span>
            <span class="resource-colon">：</span>
            { allCount > 0 && (
              <BkTag class="resource-count-tag mr-4px" theme="info">
                { '全部 ' + typeDisplayName }
              </BkTag>
            ) }
            { levels.map((level) => {
              const count = typeResources.filter(resource => resource.level === level.name).length;
              return count > 0 && (
                <BkTag
                  key={level.name}
                  class="resource-count-tag mr-4px"
                  theme={level.name === 'gateway' ? 'warning' : undefined}
                >
                  { level.display_name + ' ( ' + count + ' )' }
                </BkTag>
              );
            }) }
          </div>
        );
      }) }
    </div>
  );
};

// 根据剩余秒数生成过期提示
const getExpirationDescription = (remainingSeconds: number) => {
  if (remainingSeconds <= 0) {
    return t('已过期');
  }
  if (remainingSeconds < SECONDS_PER_MINUTE) {
    return t('即将过期');
  }
  if (remainingSeconds < SECONDS_PER_HOUR) {
    return t('{count} 分钟后过期', { count: Math.floor(remainingSeconds / SECONDS_PER_MINUTE) });
  }
  if (remainingSeconds < SECONDS_PER_DAY) {
    return t('{count} 小时后过期', { count: Math.floor(remainingSeconds / SECONDS_PER_HOUR) });
  }
  return t('{count} 天后过期', { count: Math.floor(remainingSeconds / SECONDS_PER_DAY) });
};

// 渲染过期时间单元格
const renderExpiredAt = (row: IPersonalTokenTableItem) => {
  const expiredAtText = formatUnixSeconds(row.expires_at);
  if (row.status === 'revoked') {
    return <span>{ expiredAtText }</span>;
  }
  const remainingSeconds = row.expires_at - Date.now() / 1000;
  const description = getExpirationDescription(remainingSeconds);
  const className = row.status === 'expired'
    ? 'expired-text'
    : row.status === 'valid' && remainingSeconds <= EXPIRING_THRESHOLD_DAYS * SECONDS_PER_DAY
      ? 'expiring-text'
      : undefined;
  return <span class={className}>{ `${expiredAtText}（${description}）` }</span>;
};

// 表格列及自定义单元格配置
const columns: any[] = [
  {
    title: '名称/备注',
    colKey: 'name',
    minWidth: 160,
    cell: (_h: any, { row }: { row: IPersonalTokenTableItem }) => (
      <div
        class={{
          'name-cell': true,
          'revoked': row.status === 'revoked',
        }}
      >
        <span
          class="name-text"
          onClick={() => handleView(row)}
        >
          { row.name }
        </span>
        <span class="desc-text">{ row.description }</span>
      </div>
    ),
  },
  {
    title: '授权资源',
    colKey: 'resources',
    minWidth: 200,
    cell: (_h: any, { row }: { row: IPersonalTokenTableItem }) => renderResource(row),
  },
  {
    title: 'Token',
    colKey: 'token_mask',
    minWidth: 160,
  },
  {
    title: '过期时间',
    colKey: 'expires_at',
    minWidth: 200,
    sorter: true,
    cell: (_h: any, { row }: { row: IPersonalTokenTableItem }) => renderExpiredAt(row),
  },
  {
    title: '状态',
    colKey: 'status',
    width: 120,
    filter: {
      type: 'multiple',
      list: statusOptions.map(item => ({
        label: item.label,
        value: item.value,
      })),
      popupProps: {
        overlayClassName: [
          't-table__filter-pop',
          'personal-token-status-filter',
        ],
      },
      resetValue: [],
      showConfirmAndReset: true,
    },
    cell: (_h: any, { row }: { row: IPersonalTokenTableItem }) => {
      const config = statusConfig[row.status];
      return <BkTag theme={config.theme || undefined}>{ config.text }</BkTag>;
    },
  },
  {
    title: '操作',
    colKey: 'operation',
    width: 160,
    cell: (_h: any, { row }: { row: IPersonalTokenTableItem }) => {
      // 已撤销：仅查看
      if (row.status === 'revoked') {
        return (
          <BkButton
            theme="primary"
            text
            onClick={() => handleView(row)}
          >
            查看
          </BkButton>
        );
      }
      return (
        <div class="operation-cell">
          <BkButton
            theme="primary"
            text
            onClick={() => handleView(row)}
          >
            查看
          </BkButton>
          <BkButton
            theme="primary"
            text
            onClick={() => handleEdit(row)}
          >
            编辑
          </BkButton>
          <BkButton
            theme="primary"
            text
            onClick={() => handleRenew(row)}
          >
            续期
          </BkButton>
          <BkButton
            theme="primary"
            text
            onClick={() => handleRevoke(row)}
          >
            撤销
          </BkButton>
        </div>
      );
    },
  },
];

const fetchList = (options: { resetPage?: boolean } = {}) => {
  tableRef.value?.fetchData(
    {
      statuses: tableFilterStatuses.value,
      keyword: searchKey.value,
    },
    { resetPage: options.resetPage ?? true },
  );
};

// 切换 realm 时清空弹窗上下文并重新加载资源元数据
watch(
  realm,
  (value, oldValue) => {
    resourceTypes.value = [];
    activeTokenCount.value = 0;
    currentRow.value = null;
    createDrawerShow.value = false;
    detailDrawerShow.value = false;
    renewDialogShow.value = false;
    fetchResourceTypeMetadata(value);
    if (oldValue) {
      nextTick(() => {
        fetchList({ resetPage: true });
      });
    }
  },
  { immediate: true },
);

const handleSearch = () => {
  fetchList({ resetPage: true });
};

const handleToolbarStatusChange = () => {
  tableFilterStatuses.value = filterStatus.value.length ? [...filterStatus.value] : [];
  fetchList({ resetPage: true });
};

// 兼容单列和多列排序事件，仅保留过期时间排序
const handleTableSortChange = (sort: TableSortValue) => {
  const currentSort = Array.isArray(sort)
    ? sort.find(item => item.sortBy === 'expires_at')
    : sort;
  tableSort.value = currentSort?.sortBy === 'expires_at'
    ? {
      sortBy: currentSort.sortBy,
      descending: currentSort.descending,
    }
    : undefined;
  fetchList({ resetPage: true });
};

// 将表头筛选结果同步回工具栏状态选择器
const handleTableFilterChange = (value: ITableFilterValue) => {
  tableFilterStatuses.value = Array.isArray(value.status)
    ? value.status.filter((item): item is TokenStatus =>
      statusOptions.some(option => option.value === item))
    : [];
  filterStatus.value = tableFilterStatuses.value.length ? [...tableFilterStatuses.value] : [];
  fetchList({ resetPage: true });
};

const handleClearFilter = () => {
  filterStatus.value = [];
  searchKey.value = '';
  tableFilterStatuses.value = [];
  fetchList({ resetPage: true });
};

const handleCreate = () => {
  if (isActiveTokenLimitReached.value) {
    return;
  }
  currentRow.value = null;
  createDrawerShow.value = true;
};

const handleView = (row: IPersonalTokenTableItem) => {
  currentRow.value = row;
  detailDrawerShow.value = true;
};

const handleEdit = (row: IPersonalToken) => {
  currentRow.value = row;
  detailDrawerShow.value = false;
  createDrawerShow.value = true;
};

const handleRenew = (row: IPersonalTokenTableItem) => {
  currentRow.value = row;
  renewDialogShow.value = true;
};

const handleRevoke = (row: IPersonalTokenTableItem) => {
  usePopInfoBox({
    type: 'warning',
    isShow: true,
    title: '确认撤销该个人令牌？',
    subTitle: `撤销后令牌 ${row.name} 将立即失效且无法恢复。`,
    confirmText: '撤销',
    cancelText: '取消',
    onConfirm: async () => {
      await revokePersonalToken(realm.value, row.id);
      messageSuccess('撤销成功');
      fetchList({ resetPage: false });
    },
  });
};
</script>

<style scoped lang="scss">
.personal-token {
  padding: 24px;

  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .toolbar-left {
    display: flex;
    align-items: center;
    gap: 12px;

    .create-button-wrapper {
      display: inline-flex;
    }

    .active-token-count {
      font-size: 12px;
      color: #63656e;

      &.is-limit-reached {
        color: #ea3636;
      }
    }
  }

  .toolbar-right {
    display: flex;
    align-items: center;
    gap: 8px;

    .status-select {
      width: 150px;
    }

    .search-input {
      width: 480px;
    }
  }

  :deep(.name-cell) {
    display: flex;
    flex-direction: column;
    line-height: 20px;

    .name-text {
      font-weight: bold;
      color: #3a84ff;
      cursor: pointer;

      &:hover:not(.revoked) {
        color: #699df4;
      }
    }

    .desc-text {
      color: #979ba5;
    }

    &.revoked {
      color: #c4c6cc;

      .name-text, .desc-text {
        color: #c4c6cc;

        &:hover {
          color: #c4c6cc;
        }
      }
    }
  }

  :deep(.resource-cell) {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 6px 0;

    .resource-row {
      display: flex;
      align-items: center;

      .resource-label {
        display: inline-block;
        width: 30px;
        color: #979ba5;
        text-align: justify;
        text-align-last: justify;
      }

      .resource-colon {
        margin-right: 4px;
        color: #979ba5;
      }
    }

    .resource-count-tag {
      min-width: 72px;
    }
  }

  :deep(.expired-text) {
    color: #ea3636;
  }

  :deep(.expiring-text) {
    color: #ff9c01;
  }

  :deep(.operation-cell) {
    display: flex;
    align-items: center;
    gap: 12px;
  }
}

</style>

<style lang="scss">
.personal-token-status-filter {

  .t-table__filter-pop-search {
    display: none;
  }
}
</style>
