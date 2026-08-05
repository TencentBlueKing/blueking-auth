<template>
  <div class="pt-list">
    <div class="pt-list__header">
      <div class="pt-list__title">
        个人令牌
      </div>
      <BkButton
        theme="primary"
        @click="openCreate"
      >
        新建个人令牌
      </BkButton>
    </div>

    <BkAlert
      class="pt-list__tip"
      theme="info"
      title="个人令牌用于以你的身份调用受支持的资源。令牌明文仅在创建时展示一次，请妥善保管；如发生泄露请立即撤销。"
    />

    <div class="pt-list__toolbar">
      <BkRadioGroup
        v-model="stateFilter"
        type="capsule"
        @change="fetchList"
      >
        <BkRadioButton label="">
          全部
        </BkRadioButton>
        <BkRadioButton label="active">
          有效
        </BkRadioButton>
        <BkRadioButton label="inactive">
          失效
        </BkRadioButton>
      </BkRadioGroup>
      <BkInput
        v-model="keyword"
        class="pt-list__search"
        type="search"
        clearable
        placeholder="搜索令牌名称"
      />
    </div>

    <BkLoading :loading="loading">
      <BkTable
        :data="displayData"
        :columns="columns"
        :max-height="560"
        row-hover="auto"
        :empty-text="loading ? '加载中…' : '暂无个人令牌'"
      />
    </BkLoading>

    <TokenFormSlider
      v-model:is-show="formVisible"
      :token="editingToken"
      :realm="realm"
      @created="onCreated"
      @updated="onUpdated"
    />
    <CreatedTokenDialog
      v-model:is-show="createdVisible"
      :token="createdToken"
    />
    <RenewDialog
      v-model:is-show="renewVisible"
      :token="activeToken"
      :realm="realm"
      @success="fetchList"
    />
    <TokenDetailSlider
      v-model:is-show="detailVisible"
      :token="activeToken"
    />

    <BkDialog
      :is-show="revokeVisible"
      title="确认撤销令牌？"
      theme="danger"
      :is-loading="revoking"
      confirm-text="撤销"
      @confirm="confirmRevoke"
      @closed="revokeVisible = false"
    >
      <div class="pt-revoke">
        <p>
          令牌：<strong>{{ activeToken?.name }}</strong>
        </p>
        <p class="pt-revoke__desc">
          撤销后该令牌立即失效且不可恢复，使用它的脚本或服务将无法继续调用。记录将保留用于审计。
        </p>
      </div>
    </BkDialog>
  </div>
</template>

<script setup lang="ts">
import { h } from 'vue';
import TokenFormSlider from './components/TokenFormSlider.vue';
import CreatedTokenDialog from './components/CreatedTokenDialog.vue';
import RenewDialog from './components/RenewDialog.vue';
import TokenDetailSlider from './components/TokenDetailSlider.vue';
import {
  type CreatedPersonalToken,
  PERSONAL_TOKEN_REALM,
  type PersonalToken,
  type PersonalTokenState,
  getPersonalTokens,
  revokePersonalToken,
} from '@/services/source/personalToken';
import { STATUS_META, formatDateTime, getDisplayStatus, isTokenActionable } from './utils';
import { messageSuccess } from '@/utils';

const route = useRoute();
// realm 来自路由参数，与后端 per-realm API 对齐；缺省回退到本期 demo realm。
const realm = computed(() => String(route.params.realm || PERSONAL_TOKEN_REALM));

const list = ref<PersonalToken[]>([]);
const loading = ref(false);
const stateFilter = ref<'' | PersonalTokenState>('');
const keyword = ref('');

const formVisible = ref(false);
const editingToken = ref<PersonalToken | null>(null);
const createdVisible = ref(false);
const createdToken = ref<CreatedPersonalToken | null>(null);
const detailVisible = ref(false);
const renewVisible = ref(false);
const revokeVisible = ref(false);
const revoking = ref(false);
const activeToken = ref<PersonalToken | null>(null);

const displayData = computed(() => {
  const kw = keyword.value.trim().toLowerCase();
  if (!kw) {
    return list.value;
  }
  return list.value.filter(item => item.name.toLowerCase().includes(kw));
});

const fetchList = async () => {
  loading.value = true;
  try {
    const params = stateFilter.value ? { state: stateFilter.value } : {};
    list.value = await getPersonalTokens(params, realm.value);
  }
  finally {
    loading.value = false;
  }
};

const openCreate = () => {
  editingToken.value = null;
  formVisible.value = true;
};

const openEdit = (token: PersonalToken) => {
  editingToken.value = token;
  formVisible.value = true;
};

const openDetail = (token: PersonalToken) => {
  activeToken.value = token;
  detailVisible.value = true;
};

const openRenew = (token: PersonalToken) => {
  activeToken.value = token;
  renewVisible.value = true;
};

const openRevoke = (token: PersonalToken) => {
  activeToken.value = token;
  revokeVisible.value = true;
};

const onCreated = (token: CreatedPersonalToken) => {
  createdToken.value = token;
  createdVisible.value = true;
  fetchList();
};

const onUpdated = () => {
  messageSuccess('保存成功');
  fetchList();
};

const confirmRevoke = async () => {
  if (!activeToken.value) {
    return;
  }
  revoking.value = true;
  try {
    await revokePersonalToken(activeToken.value.id, realm.value);
    messageSuccess('已撤销');
    revokeVisible.value = false;
    fetchList();
  }
  finally {
    revoking.value = false;
  }
};

const renderStatus = (token: PersonalToken) => {
  const meta = STATUS_META[getDisplayStatus(token)];
  return h('span', { class: ['pt-status', `pt-status--${getDisplayStatus(token)}`] }, meta.label);
};

const renderActions = (token: PersonalToken) => {
  const actionable = isTokenActionable(token);
  const link = (label: string, onClick: () => void, disabled = false) =>
    h(
      'span',
      {
        class: ['pt-op', disabled ? 'pt-op--disabled' : ''],
        onClick: () => !disabled && onClick(),
      },
      label,
    );
  return h('div', { class: 'pt-op-group' }, [
    link('查看', () => openDetail(token)),
    link('编辑', () => openEdit(token), !actionable),
    link('续期', () => openRenew(token), !actionable),
    link('撤销', () => openRevoke(token), !actionable),
  ]);
};

const columns = [
  {
    label: '名称',
    field: 'name',
    render: ({ row }: { row: PersonalToken }) =>
      h('span', {
        class: 'pt-name',
        onClick: () => openDetail(row),
      }, row.name),
  },
  {
    label: 'Token',
    field: 'token_mask',
    render: ({ row }: { row: PersonalToken }) => h('span', { class: 'pt-mask' }, row.token_mask),
  },
  {
    label: '授权资源',
    render: ({ row }: { row: PersonalToken }) => h('span', {}, `${row.audience.length} 项`),
  },
  {
    label: '过期时间',
    render: ({ row }: { row: PersonalToken }) => h('span', {}, formatDateTime(row.expires_at)),
  },
  {
    label: '状态',
    width: 100,
    render: ({ row }: { row: PersonalToken }) => renderStatus(row),
  },
  {
    label: '操作',
    width: 220,
    render: ({ row }: { row: PersonalToken }) => renderActions(row),
  },
];

onMounted(fetchList);
</script>

<style scoped lang="scss">
.pt-list {
  padding: 20px 24px;

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
  }

  &__title {
    font-size: 16px;
    font-weight: 700;
    color: #313238;
  }

  &__tip {
    margin-bottom: 16px;
  }

  &__toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
  }

  &__search {
    width: 320px;
  }
}

:deep(.pt-name) {
  color: #3a84ff;
  cursor: pointer;
}

:deep(.pt-mask) {
  font-family: monospace;
  color: #63656e;
}

:deep(.pt-status) {
  display: inline-block;
  padding: 0 8px;
  font-size: 12px;
  line-height: 20px;
  border-radius: 2px;

  &--normal {
    color: #14a568;
    background: #e4faf0;
  }

  &--expiring {
    color: #ff9c01;
    background: #fff3e1;
  }

  &--expired {
    color: #ea3636;
    background: #ffe6e6;
  }

  &--revoked {
    color: #979ba5;
    background: #f0f1f5;
  }
}

:deep(.pt-op-group) {
  display: flex;
  gap: 12px;
}

:deep(.pt-op) {
  color: #3a84ff;
  cursor: pointer;

  &--disabled {
    color: #c4c6cc;
    cursor: not-allowed;
  }
}
</style>
