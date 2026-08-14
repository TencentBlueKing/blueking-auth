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
      <BkButton
        theme="primary"
        @click="handleCreate"
      >
        新增个人令牌
      </BkButton>

      <div class="toolbar-right">
        <BkSelect
          v-model="filterStatus"
          class="status-select"
          placeholder="状态"
          clearable
          @change="handleSearch"
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
          placeholder="搜索令牌名称、备注、MCP 名称、API 名称、网关"
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
      :api-method="getPersonalTokenList"
      :columns="columns"
    />

    <!-- 新建/编辑令牌侧滑 -->
    <TokenCreateDrawer
      v-model:is-show="createDrawerShow"
      :token="currentRow"
      @success="handleTokenUpdated"
    />

    <!-- 令牌详情侧滑 -->
    <TokenDetailDrawer
      v-model:is-show="detailDrawerShow"
      :token="currentRow"
      @edit="handleEdit"
      @updated="handleTokenUpdated"
    />

    <!-- 续期弹窗 -->
    <RenewDialog
      v-model:is-show="renewDialogShow"
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
import { messageSuccess } from '@/utils';
import {
  type PersonalTokenItem,
  type TokenStatus,
  getPersonalTokenList,
  revokePersonalToken,
} from '@/services/source/personal-token';
import { usePopInfoBox } from '@/hooks';

type TableInstance = InstanceType<typeof CommonTable>;

const tableRef = useTemplateRef<TableInstance>('tableRef');

const searchKey = ref('');
const filterStatus = ref<TokenStatus | ''>('');

// 新建/编辑侧滑
const createDrawerShow = ref(false);

// 详情侧滑
const detailDrawerShow = ref(false);

// 续期弹窗
const renewDialogShow = ref(false);
const currentRow = ref<PersonalTokenItem | null>(null);

const handleTokenUpdated = () => {
  tableRef.value?.fetchData(
    {
      status: filterStatus.value,
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

// 计算距离过期的天数
const getRemainDays = (expiredAt: string) => {
  const diff = new Date(expiredAt).getTime() - Date.now();
  return Math.ceil(diff / (24 * 60 * 60 * 1000));
};

// 渲染授权资源单元格
const renderResource = (resource: PersonalTokenItem['resource']) => {
  const mcpTag = resource?.mcp?.all
    ? <BkTag class="resource-count-tag" theme="info">全部 MCP</BkTag>
    : <BkTag class="resource-count-tag">{ `MCP ( ${resource?.mcp?.count ?? 0} )` }</BkTag>;

  return (
    <div class="resource-cell">
      <div class="resource-row">
        <span class="resource-label">MCP</span>
        <span class="resource-colon">：</span>
        { mcpTag }
      </div>
      <div class="resource-row">
        <span class="resource-label">API</span>
        <span class="resource-colon">：</span>
        <BkTag theme="warning" class="resource-count-tag mr-4px">{ `网关 ( ${resource?.api?.gateway_count ?? 0} )` }</BkTag>
        <BkTag class="resource-count-tag">{ `API ( ${resource?.api?.api_count ?? 0} )` }</BkTag>
      </div>
    </div>
  );
};

// 渲染过期时间单元格
const renderExpiredAt = (row: PersonalTokenItem) => {
  if (row.status === 'expired') {
    return <span class="expired-text">{ row.expired_at }</span>;
  }
  if (row.status === 'valid') {
    const remainDays = getRemainDays(row.expired_at);
    if (remainDays >= 0 && remainDays <= EXPIRING_THRESHOLD_DAYS) {
      return (
        <span class="expiring-text">{ `${row.expired_at}（${remainDays}天后过期）` }</span>
      );
    }
  }
  return <span>{ row.expired_at }</span>;
};

const columns: any[] = [
  {
    title: '名称/备注',
    colKey: 'name',
    minWidth: 160,
    cell: (_h: any, { row }: { row: PersonalTokenItem }) => (
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
    colKey: 'resource',
    minWidth: 200,
    cell: (_h: any, { row }: { row: PersonalTokenItem }) => renderResource(row.resource),
  },
  {
    title: 'Token',
    colKey: 'token',
    minWidth: 160,
  },
  {
    title: '过期时间',
    colKey: 'expired_at',
    minWidth: 200,
    sorter: true,
    cell: (_h: any, { row }: { row: PersonalTokenItem }) => renderExpiredAt(row),
  },
  {
    title: '状态',
    colKey: 'status',
    width: 120,
    filter: {
      type: 'single',
      list: statusOptions.map(item => ({
        label: item.label,
        value: item.value,
      })),
    },
    cell: (_h: any, { row }: { row: PersonalTokenItem }) => {
      const config = statusConfig[row.status];
      return <BkTag theme={config.theme || undefined}>{ config.text }</BkTag>;
    },
  },
  {
    title: '操作',
    colKey: 'operation',
    width: 160,
    cell: (_h: any, { row }: { row: PersonalTokenItem }) => {
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
      // 已过期：编辑、续期置灰
      const disabled = row.status === 'expired';
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
            disabled={disabled}
            onClick={() => handleEdit(row)}
          >
            编辑
          </BkButton>
          <BkButton
            theme="primary"
            text
            disabled={disabled}
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
      status: filterStatus.value,
      keyword: searchKey.value,
    },
    { resetPage: options.resetPage ?? true },
  );
};

const handleSearch = () => {
  fetchList({ resetPage: true });
};

const handleCreate = () => {
  currentRow.value = null;
  createDrawerShow.value = true;
};

const handleView = (row: PersonalTokenItem) => {
  currentRow.value = row;
  detailDrawerShow.value = true;
};

const handleEdit = (row: PersonalTokenItem) => {
  currentRow.value = row;
  detailDrawerShow.value = false;
  createDrawerShow.value = true;
};

const handleRenew = (row: PersonalTokenItem) => {
  currentRow.value = row;
  renewDialogShow.value = true;
};

const handleRevoke = (row: PersonalTokenItem) => {
  usePopInfoBox({
    type: 'warning',
    isShow: true,
    title: '确认撤销该个人令牌？',
    subTitle: `撤销后令牌 ${row.name} 将立即失效且无法恢复。`,
    confirmText: '撤销',
    cancelText: '取消',
    onConfirm: async () => {
      await revokePersonalToken(row.id);
      messageSuccess('撤销成功');
      fetchList({ resetPage: false });
    },
  });
  // InfoBox({
  //   title: '确认撤销该个人令牌？',
  //   subTitle: `撤销后令牌 ${row.name} 将立即失效且无法恢复。`,
  //   confirmText: '撤销',
  //   // theme: 'warning',
  //   onConfirm: async () => {
  //     await revokePersonalToken(row.id);
  //     messageSuccess('撤销成功');
  //     fetchList({ resetPage: false });
  //   },
  // });
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
