<template>
  <BkSideslider
    v-model:is-show="isShow"
    class="token-create-drawer"
    :width="960"
    :quick-close="false"
    @closed="handleClosed"
  >
    <template #header>
      {{ drawerTitle }}
    </template>

    <div
      v-bkloading="{ loading: initializing }"
      class="drawer-body"
    >
      <BkForm
        ref="formRef"
        form-type="vertical"
        :model="formData"
        :rules="formRules"
      >
        <BkFormItem
          property="name"
          required
          :label="t('名称')"
        >
          <BkInput
            v-model="formData.name"
            :maxlength="64"
            :placeholder="t('例如：Cursor 日常开发')"
          />
        </BkFormItem>

        <BkFormItem
          property="description"
          :label="t('备注')"
        >
          <BkInput
            v-model="formData.description"
            :maxlength="200"
            :placeholder="t('用于说明令牌用途')"
          />
        </BkFormItem>

        <BkFormItem
          property="expiredAt"
          :label="t('过期时间')"
          :required="!formData.permanent"
        >
          <div class="expired-field">
            <BkDatePicker
              v-model="formData.expiredAt"
              class="expired-picker"
              type="datetime"
              append-to-body
              :clearable="false"
              :disabled="formData.permanent"
              :disabled-date="disabledDate"
              :placeholder="t('请选择过期时间')"
            >
              <template #shortcuts="{ change }">
                <div class="date-shortcuts">
                  <BkButton
                    v-for="item in dateShortcuts"
                    :key="item.days"
                    class="date-shortcut-item"
                    text
                    @click="() => handleDateShortcut(item.days, change)"
                  >
                    {{ item.label }}
                  </BkButton>
                </div>
              </template>
            </BkDatePicker>

            <BkCheckbox
              v-model="formData.permanent"
              v-bk-tooltips="{ content: '暂未支持'}"
              disabled
              @change="handlePermanentChange"
            >
              {{ t('永久有效') }}
            </BkCheckbox>
          </div>
        </BkFormItem>
      </BkForm>

      <section class="scope-section">
        <div class="section-title">
          {{ t('授权范围') }}
        </div>

        <div
          v-if="resourceError"
          class="scope-error"
        >
          {{ t('请选择至少一项授权资源') }}
        </div>

        <div class="scope-card">
          <div class="scope-card-header">
            <div
              class="scope-card-title"
              @click="mcpSectionExpanded = !mcpSectionExpanded"
            >
              <CommonIcon
                class="scope-arrow"
                :class="{ 'is-collapsed': !mcpSectionExpanded }"
                name="down-shape"
                color="#63656E"
                size="10"
              />
              <span>{{ t('MCP（共 {count} 个）', { count: mcpTotalCount }) }}</span>
            </div>

            <div
              v-bk-tooltips="t('勾选后将自动包含后续新增的 MCP')"
              class="scope-select-all"
              @click.stop
            >
              <BkCheckbox
                v-model="mcpSelectAll"
                @change="handleMcpSelectAll"
              >
                <strong class="select-all-text">{{ t('全选') }}</strong>{{ t('，包括后续新增的 MCP 也生效') }}
              </BkCheckbox>
            </div>
          </div>

          <div
            v-show="mcpSectionExpanded"
            class="scope-card-content"
          >
            <div class="resource-browser">
              <div class="resource-list-pane">
                <BkInput
                  v-model="mcpSearchKey"
                  class="resource-search"
                  clearable
                  :placeholder="t('搜索网关、MCP 名称')"
                  type="search"
                />

                <div class="resource-group-list">
                  <CheckboxCollapse
                    v-for="group in pagedMcpGroups"
                    :key="group.id"
                    v-model:collapsed="mcpCollapsed[group.id]"
                    arrow-position="left"
                    compact
                    :auto-expand-on-check="false"
                    :checkbox-disabled="mcpSelectAll"
                    :disabled-tips="t('取消全选后可单独选择网关')"
                    :model-value="isMcpGroupSelected(group)"
                    @update:model-value="value => handleMcpGroupSelect(group, value)"
                  >
                    <template #title>
                      <div class="group-title">
                        <span>{{ group.name }}</span>
                        <span class="group-count">
                          {{ t('共 {total} 个，已选 {selected} 个', {
                            total: group.items.length,
                            selected: getMcpSelectedCount(group),
                          }) }}
                        </span>
                        <BkTag
                          v-if="group.official"
                          size="small"
                          theme="info"
                        >
                          {{ t('官方') }}
                        </BkTag>
                      </div>
                    </template>

                    <BkCheckboxGroup
                      v-model="selectedMcpIds"
                      class="resource-checkbox-list"
                      :disabled="mcpSelectAll || isMcpGroupSelected(group)"
                    >
                      <BkCheckbox
                        v-for="item in group.items"
                        :key="item.id"
                        :label="item.id"
                        size="small"
                      >
                        {{ item.name }}（{{ item.id }}）
                      </BkCheckbox>
                    </BkCheckboxGroup>
                  </CheckboxCollapse>

                  <BkException
                    v-if="!filteredMcpGroups.length"
                    class="resource-empty"
                    type="empty"
                    scene="part"
                    :description="t('暂无匹配资源')"
                  />
                </div>

                <BkPagination
                  v-model="mcpPage"
                  v-model:limit="mcpPageSize"
                  class="resource-pagination"
                  :count="filteredMcpGroups.length"
                  :limit-list="pageSizeOptions"
                  small
                  show-total-count
                />

                <div class="custom-resource-area">
                  <BkButton
                    class="add-custom-button"
                    theme="primary"
                    text
                    @click="handleAddCustomMcp"
                  >
                    <CommonIcon
                      name="plus-circle-shape"
                      class="mr-4px"
                    />
                    {{ t('添加非公开 MCP') }}
                  </BkButton>

                  <div
                    v-for="item in customMcpResources"
                    :key="item.uid"
                    class="custom-resource-row"
                  >
                    <BkInput
                      v-model="item.serverUrl"
                      :placeholder="t('请输入 MCP 服务地址')"
                    />
                    <BkInput
                      v-model="item.name"
                      :placeholder="t('请输入 MCP 名称')"
                    />
                    <BkButton
                      class="delete-custom-button"
                      text
                      @click="() => handleDeleteCustomMcp(item.uid)"
                    >
                      <Del />
                    </BkButton>
                  </div>
                </div>
              </div>

              <div class="selection-preview">
                <div class="preview-header">
                  <span class="font-bold">{{ t('选择结果预览（{count}）', { count: mcpPreviewItems.length }) }}</span>
                  <BkButton
                    v-if="mcpPreviewItems.length"
                    theme="primary"
                    text
                    @click="handleClearMcp"
                  >
                    {{ t('清空') }}
                  </BkButton>
                </div>

                <div class="preview-content">
                  <div
                    v-if="mcpPreviewItems.length"
                    class="preview-group"
                  >
                    <div class="preview-group-title">
                      【MCP】- {{ t('共 {count} 个', { count: mcpPreviewItems.length }) }}
                    </div>
                    <div
                      v-for="item in mcpPreviewItems"
                      :key="item.key"
                      class="preview-item"
                    >
                      <div class="preview-item-main">
                        <BkTag
                          v-for="tag in item.tags"
                          :key="tag.text"
                          size="small"
                          :theme="tag.theme || undefined"
                        >
                          {{ tag.text }}
                        </BkTag>
                        <span class="preview-item-name">{{ item.name }}</span>
                      </div>
                      <BkButton
                        class="remove-preview-button"
                        text
                        @click="() => handleRemoveMcpPreview(item)"
                      >
                        <CloseLine />
                      </BkButton>
                    </div>
                  </div>

                  <BkException
                    v-else
                    class="preview-empty"
                    type="empty"
                    scene="part"
                    :description="t('暂无选择结果')"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="scope-card">
          <div class="scope-card-header">
            <div
              class="scope-card-title"
              @click="apiSectionExpanded = !apiSectionExpanded"
            >
              <CommonIcon
                class="scope-arrow"
                :class="{ 'is-collapsed': !apiSectionExpanded }"
                name="down-shape"
                color="#63656E"
                size="10"
              />
              <span>{{ t('API（共 {count} 个）', { count: apiTotalCount }) }}</span>
            </div>

            <div
              v-bk-tooltips="t('勾选后将自动包含后续新增的 API')"
              class="scope-select-all"
              @click.stop
            >
              <BkCheckbox
                v-model="apiSelectAll"
                @change="handleApiSelectAll"
              >
                <strong class="select-all-text">{{ t('全选') }}</strong>{{ t('，包括后续新增的 API 也生效') }}
              </BkCheckbox>
            </div>
          </div>

          <div
            v-show="apiSectionExpanded"
            class="scope-card-content"
          >
            <div class="resource-browser">
              <div class="resource-list-pane">
                <BkInput
                  v-model="apiSearchKey"
                  class="resource-search"
                  clearable
                  :placeholder="t('搜索网关、API 名称')"
                  type="search"
                />

                <div class="resource-group-list">
                  <CheckboxCollapse
                    v-for="group in pagedApiGroups"
                    :key="group.id"
                    v-model:collapsed="apiCollapsed[group.id]"
                    arrow-position="left"
                    compact
                    :auto-expand-on-check="false"
                    :checkbox-disabled="apiSelectAll"
                    :disabled-tips="t('取消全选后可单独选择网关')"
                    :model-value="selectedGatewayIds.includes(group.id)"
                    @update:model-value="value => handleGatewaySelect(group, value)"
                  >
                    <template #title>
                      <div class="group-title">
                        <span>{{ group.name }}</span>
                        <span class="group-count">
                          {{ t('共 {total} 个，已选 {selected} 个', {
                            total: group.items.length,
                            selected: getApiSelectedCount(group),
                          }) }}
                        </span>
                        <BkTag
                          v-if="group.official"
                          size="small"
                          theme="info"
                        >
                          {{ t('官方') }}
                        </BkTag>
                      </div>
                    </template>

                    <BkCheckboxGroup
                      v-model="selectedApiIds"
                      class="resource-checkbox-list"
                      :disabled="apiSelectAll || selectedGatewayIds.includes(group.id)"
                    >
                      <BkCheckbox
                        v-for="item in group.items"
                        :key="item.id"
                        :label="item.id"
                        size="small"
                      >
                        {{ item.name }}（{{ item.action }}）
                      </BkCheckbox>
                    </BkCheckboxGroup>
                  </CheckboxCollapse>

                  <BkException
                    v-if="!filteredApiGroups.length"
                    class="resource-empty"
                    type="empty"
                    scene="part"
                    :description="t('暂无匹配资源')"
                  />
                </div>

                <BkPagination
                  v-model="apiPage"
                  v-model:limit="apiPageSize"
                  class="resource-pagination"
                  :count="filteredApiGroups.length"
                  :limit-list="pageSizeOptions"
                  small
                  show-total-count
                />

                <div class="custom-resource-area">
                  <BkButton
                    class="add-custom-button"
                    theme="primary"
                    text
                    @click="handleAddCustomApi"
                  >
                    <CommonIcon
                      name="plus-circle-shape"
                      class="mr-4px"
                    />
                    {{ t('添加非公开 API') }}
                  </BkButton>

                  <div
                    v-for="item in customApiResources"
                    :key="item.uid"
                    class="custom-resource-row"
                  >
                    <BkInput
                      v-model="item.gatewayName"
                      :placeholder="t('请输入网关名称')"
                    />
                    <BkInput
                      v-model="item.name"
                      :placeholder="t('请输入 API 资源名称')"
                    />
                    <BkButton
                      class="delete-custom-button"
                      text
                      @click="() => handleDeleteCustomApi(item.uid)"
                    >
                      <Del />
                    </BkButton>
                  </div>
                </div>
              </div>

              <div class="selection-preview">
                <div class="preview-header">
                  <span class="font-bold">{{ t('选择结果预览（{count}）', { count: apiPreviewCount }) }}</span>
                  <BkButton
                    v-if="apiPreviewCount"
                    theme="primary"
                    text
                    @click="handleClearApi"
                  >
                    {{ t('清空') }}
                  </BkButton>
                </div>

                <div class="preview-content">
                  <div
                    v-if="apiSelectAll"
                    class="preview-group"
                  >
                    <div
                      v-for="item in apiAllPreviewItems"
                      :key="item.key"
                      class="preview-item"
                    >
                      <div class="preview-item-main">
                        <BkTag
                          v-for="tag in item.tags"
                          :key="tag.text"
                          size="small"
                          :theme="tag.theme || undefined"
                        >
                          {{ tag.text }}
                        </BkTag>
                        <span class="preview-item-name">{{ item.name }}</span>
                      </div>
                      <BkButton
                        class="remove-preview-button"
                        text
                        @click="() => handleRemoveApiAllPreview(item)"
                      >
                        <CloseLine />
                      </BkButton>
                    </div>
                  </div>

                  <template v-else-if="apiPreviewCount">
                    <div
                      v-if="apiGatewayPreviewItems.length"
                      class="preview-group"
                    >
                      <div class="preview-group-title">
                        【{{ t('网关') }}】- {{ t('共 {count} 个', { count: apiGatewayPreviewItems.length }) }}
                      </div>
                      <div
                        v-for="item in apiGatewayPreviewItems"
                        :key="item.key"
                        class="preview-item"
                      >
                        <div class="preview-item-main">
                          <BkTag
                            v-for="tag in item.tags"
                            :key="tag.text"
                            size="small"
                            :theme="tag.theme || undefined"
                          >
                            {{ tag.text }}
                          </BkTag>
                          <span class="preview-item-name">{{ item.name }}</span>
                        </div>
                        <BkButton
                          class="remove-preview-button"
                          text
                          @click="() => handleRemoveApiPreview(item)"
                        >
                          <CloseLine />
                        </BkButton>
                      </div>
                    </div>

                    <div
                      v-if="apiResourcePreviewItems.length"
                      class="preview-group"
                    >
                      <div class="preview-group-title">
                        【API】- {{ t('共 {count} 个', { count: apiResourcePreviewItems.length }) }}
                      </div>
                      <div
                        v-for="item in apiResourcePreviewItems"
                        :key="item.key"
                        class="preview-item"
                      >
                        <div class="preview-item-main">
                          <BkTag
                            v-for="tag in item.tags"
                            :key="tag.text"
                            size="small"
                            :theme="tag.theme || undefined"
                          >
                            {{ tag.text }}
                          </BkTag>
                          <span class="preview-item-name">{{ item.name }}</span>
                        </div>
                        <BkButton
                          class="remove-preview-button"
                          text
                          @click="() => handleRemoveApiPreview(item)"
                        >
                          <CloseLine />
                        </BkButton>
                      </div>
                    </div>
                  </template>

                  <BkException
                    v-else
                    class="preview-empty"
                    type="empty"
                    scene="part"
                    :description="t('暂无选择结果')"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>

    <template #footer>
      <BkButton
        theme="primary"
        :loading="submitting"
        @click="handleSubmit"
      >
        {{ primaryButtonText }}
      </BkButton>
      <BkButton
        class="ml-8px"
        @click="handleCancel"
      >
        {{ t('取消') }}
      </BkButton>
    </template>
  </BkSideslider>

  <BkDialog
    v-model:is-show="createdDialogShow"
    class="token-created-dialog"
    width="520"
    :title="''"
    :quick-close="false"
    @closed="handleCreatedDialogClosed"
  >
    <div class="created-dialog-body">
      <Success class="created-success-icon" />
      <div class="created-dialog-title">
        {{ t('令牌创建成功') }}
      </div>
      <BkAlert theme="warning">
        <template #title>
          {{ t('请妥善保存令牌，关闭弹窗后将无法再次查看它') }}
        </template>
      </BkAlert>
      <div class="created-token-info">
        <div class="created-info-label">
          {{ t('令牌名称') }}
        </div>
        <div class="created-info-value">
          {{ createdResult?.name || '--' }}
        </div>
        <div class="created-info-label mt-12px">
          {{ t('令牌（access_token）') }}
        </div>
        <div class="created-token-value">
          <span>{{ createdResult?.token || '--' }}</span>
          <BkButton
            v-if="createdResult?.token"
            class="copy-token-button"
            text
            @click="handleCopyToken"
          >
            <Copy />
          </BkButton>
        </div>
      </div>
    </div>

    <template #footer>
      <BkButton @click="createdDialogShow = false">
        {{ t('关闭') }}
      </BkButton>
    </template>
  </BkDialog>
</template>

<script setup lang="ts">
import {
  CloseLine,
  Copy,
  Del,
  Success,
} from 'bkui-vue/lib/icon';

import CheckboxCollapse from '@/components/checkbox-collapse/Index.vue';
import {
  type PersonalTokenCreateResult,
  type PersonalTokenDetail,
  type PersonalTokenItem,
  type PersonalTokenPayload,
  createPersonalToken,
  getPersonalTokenDetail,
  updatePersonalToken,
} from '@/services/source/personal-token';
import {
  messageError,
  messageSuccess,
  messageWarn,
} from '@/utils';

type PreviewKind = 'all' | 'mcp' | 'gateway' | 'api' | 'custom-mcp' | 'custom-api';
type TagTheme = 'danger' | 'info' | 'success' | 'warning';
type DateShortcutChange = (date: Date, visible?: boolean) => void;

interface IProps { token?: PersonalTokenItem | null }

interface IEmits { success: [] }

interface IFormInstance {
  validate: () => Promise<Record<string, unknown>>
  clearValidate: (fields?: string | string[]) => void
}

interface IFormData {
  name: string
  description: string
  expiredAt: Date | string | null
  permanent: boolean
}

interface IMcpResource {
  id: string
  name: string
}

interface IApiResource {
  id: string
  name: string
  action: string
  isPublic: boolean
}

interface IResourceGroup<T> {
  id: string
  name: string
  official: boolean
  items: T[]
}

interface ICustomMcpResource {
  uid: number
  serverUrl: string
  name: string
}

interface ICustomApiResource {
  uid: number
  gatewayName: string
  name: string
}

interface IPreviewTag {
  text: string
  theme?: TagTheme
}

interface IPreviewItem {
  key: string
  id: string
  kind: PreviewKind
  name: string
  tags: IPreviewTag[]
}

const isShow = defineModel<boolean>('isShow', { default: false });

const { token = null } = defineProps<IProps>();

const emit = defineEmits<IEmits>();

const { t } = useI18n();

const formData = ref<IFormData>({
  name: '',
  description: '',
  expiredAt: null,
  permanent: false,
});
const initializing = ref(false);
const submitting = ref(false);
const createdDialogShow = ref(false);
const createdResult = ref<PersonalTokenCreateResult | null>(null);
const mcpSectionExpanded = ref(true);
const apiSectionExpanded = ref(true);
const mcpSelectAll = ref(false);
const apiSelectAll = ref(false);
const selectedMcpIds = ref<string[]>([]);
const selectedGatewayIds = ref<string[]>([]);
const selectedApiIds = ref<string[]>([]);
const mcpSearchKey = ref('');
const apiSearchKey = ref('');
const mcpPage = ref(1);
const apiPage = ref(1);
const mcpPageSize = ref(10);
const apiPageSize = ref(10);
const mcpCollapsed = ref<Record<string, boolean>>({});
const apiCollapsed = ref<Record<string, boolean>>({});
const customMcpResources = ref<ICustomMcpResource[]>([]);
const customApiResources = ref<ICustomApiResource[]>([]);
const resourceError = ref(false);

const formRef = useTemplateRef<IFormInstance>('formRef');

let customResourceSeed = 0;

const pageSizeOptions = [5, 10, 20];

const mcpGroups: IResourceGroup<IMcpResource>[] = Array.from({ length: 20 }, (_, groupIndex) => {
  const groupNo = String(groupIndex + 1).padStart(2, '0');
  const itemCount = groupIndex % 3 + 2;
  return {
    id: 'mcp-group-' + groupNo,
    name: groupIndex === 0 ? '蓝鲸监控' : '业务网关' + groupNo,
    official: groupIndex < 2,
    items: Array.from({ length: itemCount }, (_, itemIndex) => ({
      id: groupIndex === 0 ? 'mcp-id' + (itemIndex + 1) : 'mcp-' + groupNo + '-' + (itemIndex + 1),
      name: 'MCP名称' + (itemIndex + 1),
    })),
  };
});

const apiGroups: IResourceGroup<IApiResource>[] = Array.from({ length: 20 }, (_, groupIndex) => {
  const groupNo = String(groupIndex + 1).padStart(2, '0');
  const itemCount = groupIndex % 4 + 2;
  return {
    id: 'gateway-' + groupNo,
    name: '业务网关' + groupNo,
    official: groupIndex < 2,
    items: Array.from({ length: itemCount }, (_, itemIndex) => ({
      id: groupIndex === 0 ? 'api-id' + (itemIndex + 1) : 'api-' + groupNo + '-' + (itemIndex + 1),
      name: 'API 名称' + (itemIndex + 1),
      action: '资源操作' + (itemIndex + 1),
      isPublic: itemIndex !== itemCount - 1,
    })),
  };
});

const isEdit = computed(() => Boolean(token?.id));
const drawerTitle = computed(() => (isEdit.value ? t('编辑个人令牌') : t('新增个人令牌')));
const primaryButtonText = computed(() => (isEdit.value ? t('保存') : t('生成令牌')));
const dateShortcuts = computed(() => [
  {
    days: 7,
    label: t('7天后'),
  },
  {
    days: 15,
    label: t('15天后'),
  },
  {
    days: 30,
    label: t('30天后'),
  },
  {
    days: 60,
    label: t('60天后'),
  },
  {
    days: 90,
    label: t('90天后'),
  },
  {
    days: 180,
    label: t('180天后'),
  },
  {
    days: 365,
    label: t('365天后'),
  },
]);
const formRules = computed(() => ({
  name: [
    {
      required: true,
      message: t('请输入令牌名称'),
      trigger: 'blur',
    },
  ],
  expiredAt: [
    {
      validator: (value: Date | string | null) => formData.value.permanent || Boolean(value),
      message: t('请选择过期时间'),
      trigger: 'change',
    },
  ],
}));
const mcpTotalCount = computed(() =>
  mcpGroups.reduce((total, group) => total + group.items.length, 0));
const apiTotalCount = computed(() =>
  apiGroups.reduce((total, group) => total + group.items.length, 0));
const mcpResourceMap = computed(() => new Map(
  mcpGroups.flatMap(group => group.items.map(item => [
    item.id,
    {
      group,
      item,
    },
  ] as const)),
));
const mcpGroupMap = computed(() => new Map(mcpGroups.map(group => [group.id, group])));
const apiGroupMap = computed(() => new Map(apiGroups.map(group => [group.id, group])));
const apiResourceMap = computed(() => new Map(
  apiGroups.flatMap(group => group.items.map(item => [
    item.id,
    {
      group,
      item,
    },
  ] as const)),
));
const filteredMcpGroups = computed(() => {
  const keyword = mcpSearchKey.value.trim().toLowerCase();
  if (!keyword) {
    return mcpGroups;
  }
  return mcpGroups.reduce<IResourceGroup<IMcpResource>[]>((result, group) => {
    const groupMatched = group.name.toLowerCase().includes(keyword);
    const items = groupMatched
      ? group.items
      : group.items.filter(item =>
        item.name.toLowerCase().includes(keyword) || item.id.toLowerCase().includes(keyword));
    if (items.length) {
      result.push({
        ...group,
        items,
      });
    }
    return result;
  }, []);
});
const filteredApiGroups = computed(() => {
  const keyword = apiSearchKey.value.trim().toLowerCase();
  if (!keyword) {
    return apiGroups;
  }
  return apiGroups.reduce<IResourceGroup<IApiResource>[]>((result, group) => {
    const groupMatched = group.name.toLowerCase().includes(keyword);
    const items = groupMatched
      ? group.items
      : group.items.filter(item =>
        item.name.toLowerCase().includes(keyword) || item.action.toLowerCase().includes(keyword));
    if (items.length) {
      result.push({
        ...group,
        items,
      });
    }
    return result;
  }, []);
});
const pagedMcpGroups = computed(() => {
  const start = (mcpPage.value - 1) * mcpPageSize.value;
  return filteredMcpGroups.value.slice(start, start + mcpPageSize.value);
});
const pagedApiGroups = computed(() => {
  const start = (apiPage.value - 1) * apiPageSize.value;
  return filteredApiGroups.value.slice(start, start + apiPageSize.value);
});
const completeCustomMcpResources = computed(() =>
  customMcpResources.value.filter(item => item.serverUrl.trim() && item.name.trim()));
const completeCustomApiResources = computed(() =>
  customApiResources.value.filter(item => item.gatewayName.trim() && item.name.trim()));
const mcpPreviewItems = computed<IPreviewItem[]>(() => {
  const customItems: IPreviewItem[] = completeCustomMcpResources.value.map(item => ({
    key: 'custom-mcp-' + item.uid,
    id: String(item.uid),
    kind: 'custom-mcp',
    name: item.name + '（' + item.serverUrl + '）',
    tags: [
      { text: 'MCP' },
      {
        text: t('非公开'),
        theme: 'danger',
      },
    ],
  }));
  if (mcpSelectAll.value) {
    return [
      {
        key: 'mcp-all',
        id: 'mcp-all',
        kind: 'all',
        name: t('全部 MCP（包括后续新增）'),
        tags: [
          {
            text: 'ALL',
            theme: 'info',
          },
        ],
      },
      ...customItems,
    ];
  }
  const selectedItems = selectedMcpIds.value.reduce<IPreviewItem[]>((result, id) => {
    const matched = mcpResourceMap.value.get(id);
    if (matched) {
      result.push({
        key: 'mcp-' + id,
        id,
        kind: 'mcp',
        name: matched.item.name + '（' + matched.item.id + '）',
        tags: [{ text: 'MCP' }],
      });
    }
    return result;
  }, []);
  return [...selectedItems, ...customItems];
});
const apiAllPreviewItems = computed<IPreviewItem[]>(() => {
  const customItems: IPreviewItem[] = completeCustomApiResources.value.map(item => ({
    key: 'custom-api-' + item.uid,
    id: String(item.uid),
    kind: 'custom-api',
    name: item.gatewayName + ' / ' + item.name,
    tags: [
      {
        text: 'API',
        theme: 'info',
      },
      {
        text: t('非公开'),
        theme: 'danger',
      },
    ],
  }));
  return [
    {
      key: 'api-all',
      id: 'api-all',
      kind: 'all',
      name: t('全部 API（包括后续新增）'),
      tags: [
        {
          text: 'ALL',
          theme: 'info',
        },
      ],
    },
    ...customItems,
  ];
});
const apiGatewayPreviewItems = computed<IPreviewItem[]>(() =>
  selectedGatewayIds.value.reduce<IPreviewItem[]>((result, id) => {
    const group = apiGroupMap.value.get(id);
    if (group) {
      result.push({
        key: 'gateway-' + id,
        id,
        kind: 'gateway',
        name: group.name,
        tags: [
          {
            text: t('网关'),
            theme: 'warning' as const,
          },
        ],
      });
    }
    return result;
  }, []));
const apiResourcePreviewItems = computed<IPreviewItem[]>(() => {
  const selectedItems = selectedApiIds.value.reduce<IPreviewItem[]>((result, id) => {
    const matched = apiResourceMap.value.get(id);
    if (matched) {
      const tags: IPreviewTag[] = [
        {
          text: 'API',
          theme: 'info',
        },
      ];
      if (!matched.item.isPublic) {
        tags.push({
          text: t('非公开'),
          theme: 'danger',
        });
      }
      result.push({
        key: 'api-' + id,
        id,
        kind: 'api',
        name: matched.item.name + '（' + matched.item.action + '）',
        tags,
      });
    }
    return result;
  }, []);
  const customItems: IPreviewItem[] = completeCustomApiResources.value.map(item => ({
    key: 'custom-api-' + item.uid,
    id: String(item.uid),
    kind: 'custom-api' as const,
    name: item.gatewayName + ' / ' + item.name,
    tags: [
      {
        text: 'API',
        theme: 'info' as const,
      },
      {
        text: t('非公开'),
        theme: 'danger' as const,
      },
    ],
  }));
  return [...selectedItems, ...customItems];
});
const apiPreviewCount = computed(() => (apiSelectAll.value
  ? apiAllPreviewItems.value.length
  : apiGatewayPreviewItems.value.length + apiResourcePreviewItems.value.length));
const hasSelectedResource = computed(() =>
  mcpSelectAll.value
  || apiSelectAll.value
  || selectedMcpIds.value.length > 0
  || selectedGatewayIds.value.length > 0
  || selectedApiIds.value.length > 0
  || completeCustomMcpResources.value.length > 0
  || completeCustomApiResources.value.length > 0);

watch(mcpSearchKey, () => {
  mcpPage.value = 1;
  if (mcpSearchKey.value.trim()) {
    filteredMcpGroups.value.forEach((group) => {
      mcpCollapsed.value[group.id] = false;
    });
  }
});

watch(apiSearchKey, () => {
  apiPage.value = 1;
  if (apiSearchKey.value.trim()) {
    filteredApiGroups.value.forEach((group) => {
      apiCollapsed.value[group.id] = false;
    });
  }
});

watch(mcpPageSize, () => {
  mcpPage.value = 1;
});

watch(apiPageSize, () => {
  apiPage.value = 1;
});

watch(hasSelectedResource, (value) => {
  if (value) {
    resourceError.value = false;
  }
});

const disabledDate = (value: Date | number) => {
  const date = value instanceof Date ? value : new Date(value);
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  return date.getTime() < today.getTime();
};

const handleDateShortcut = (days: number, change: DateShortcutChange) => {
  const date = new Date();
  date.setDate(date.getDate() + days);
  date.setSeconds(0, 0);
  formData.value.expiredAt = date;
  change(date);
  formRef.value?.clearValidate('expiredAt');
};

const handlePermanentChange = (value: boolean | number | string) => {
  formData.value.permanent = Boolean(value);
  if (formData.value.permanent) {
    formData.value.expiredAt = null;
    formRef.value?.clearValidate('expiredAt');
  }
};

const getMcpSelectedCount = (group: IResourceGroup<IMcpResource>) => {
  if (mcpSelectAll.value) {
    return group.items.length;
  }
  return group.items.filter(item => selectedMcpIds.value.includes(item.id)).length;
};

const isMcpGroupSelected = (group: IResourceGroup<IMcpResource>) => {
  const sourceGroup = mcpGroupMap.value.get(group.id) ?? group;
  return sourceGroup.items.length > 0
    && sourceGroup.items.every(item => selectedMcpIds.value.includes(item.id));
};

const getApiSelectedCount = (group: IResourceGroup<IApiResource>) => {
  if (apiSelectAll.value || selectedGatewayIds.value.includes(group.id)) {
    return group.items.length;
  }
  return group.items.filter(item => selectedApiIds.value.includes(item.id)).length;
};

const handleMcpSelectAll = (value: boolean | number | string) => {
  mcpSelectAll.value = Boolean(value);
  if (mcpSelectAll.value) {
    selectedMcpIds.value = [];
  }
};

const handleMcpGroupSelect = (group: IResourceGroup<IMcpResource>, value: boolean) => {
  const sourceGroup = mcpGroupMap.value.get(group.id) ?? group;
  const groupMcpIds = new Set(sourceGroup.items.map(item => item.id));
  if (value) {
    selectedMcpIds.value = [...new Set([...selectedMcpIds.value, ...groupMcpIds])];
    return;
  }
  selectedMcpIds.value = selectedMcpIds.value.filter(id => !groupMcpIds.has(id));
};

const handleApiSelectAll = (value: boolean | number | string) => {
  apiSelectAll.value = Boolean(value);
  if (apiSelectAll.value) {
    selectedGatewayIds.value = [];
    selectedApiIds.value = [];
  }
};

const handleGatewaySelect = (group: IResourceGroup<IApiResource>, value: boolean) => {
  const sourceGroup = apiGroupMap.value.get(group.id) ?? group;
  if (value) {
    if (!selectedGatewayIds.value.includes(group.id)) {
      selectedGatewayIds.value = [...selectedGatewayIds.value, group.id];
    }
    const groupApiIds = new Set(sourceGroup.items.map(item => item.id));
    selectedApiIds.value = selectedApiIds.value.filter(id => !groupApiIds.has(id));
    return;
  }
  selectedGatewayIds.value = selectedGatewayIds.value.filter(id => id !== group.id);
};

const handleAddCustomMcp = () => {
  customResourceSeed += 1;
  customMcpResources.value.push({
    uid: customResourceSeed,
    serverUrl: '',
    name: '',
  });
};

const handleDeleteCustomMcp = (uid: number) => {
  customMcpResources.value = customMcpResources.value.filter(item => item.uid !== uid);
};

const handleAddCustomApi = () => {
  customResourceSeed += 1;
  customApiResources.value.push({
    uid: customResourceSeed,
    gatewayName: '',
    name: '',
  });
};

const handleDeleteCustomApi = (uid: number) => {
  customApiResources.value = customApiResources.value.filter(item => item.uid !== uid);
};

const handleRemoveMcpPreview = (item: IPreviewItem) => {
  if (item.kind === 'all') {
    mcpSelectAll.value = false;
    return;
  }
  if (item.kind === 'custom-mcp') {
    handleDeleteCustomMcp(Number(item.id));
    return;
  }
  selectedMcpIds.value = selectedMcpIds.value.filter(id => id !== item.id);
};

const handleRemoveApiPreview = (item: IPreviewItem) => {
  if (item.kind === 'gateway') {
    selectedGatewayIds.value = selectedGatewayIds.value.filter(id => id !== item.id);
    return;
  }
  if (item.kind === 'custom-api') {
    handleDeleteCustomApi(Number(item.id));
    return;
  }
  selectedApiIds.value = selectedApiIds.value.filter(id => id !== item.id);
};

const handleRemoveApiAllPreview = (item: IPreviewItem) => {
  if (item.kind === 'all') {
    apiSelectAll.value = false;
    return;
  }
  if (item.kind === 'custom-api') {
    handleDeleteCustomApi(Number(item.id));
  }
};

const handleClearMcp = () => {
  mcpSelectAll.value = false;
  selectedMcpIds.value = [];
  customMcpResources.value = [];
};

const handleClearApi = () => {
  apiSelectAll.value = false;
  selectedGatewayIds.value = [];
  selectedApiIds.value = [];
  customApiResources.value = [];
};

const parseDate = (value: string) => new Date(value.replace(/-/g, '/'));

const formatDate = (value: Date | string) => {
  const date = value instanceof Date ? value : parseDate(value);
  const pad = (number: number) => String(number).padStart(2, '0');
  return date.getFullYear()
    + '-' + pad(date.getMonth() + 1)
    + '-' + pad(date.getDate())
    + ' ' + pad(date.getHours())
    + ':' + pad(date.getMinutes())
    + ':' + pad(date.getSeconds());
};

const hasIncompleteCustomResource = () =>
  customMcpResources.value.some(item => !item.serverUrl.trim() || !item.name.trim())
  || customApiResources.value.some(item => !item.gatewayName.trim() || !item.name.trim());

const validateResources = () => {
  if (hasIncompleteCustomResource()) {
    messageWarn(t('请完整填写非公开资源信息'));
    return false;
  }
  if (!hasSelectedResource.value) {
    resourceError.value = true;
    messageWarn(t('请选择至少一项授权资源'));
    return false;
  }
  return true;
};

const buildPayload = (): PersonalTokenPayload => ({
  name: formData.value.name.trim(),
  description: formData.value.description.trim(),
  permanent: formData.value.permanent,
  expired_at: formData.value.permanent || !formData.value.expiredAt
    ? null
    : formatDate(formData.value.expiredAt),
  resource: {
    mcp: {
      all: mcpSelectAll.value,
      ids: mcpSelectAll.value ? [] : selectedMcpIds.value,
      custom_resources: completeCustomMcpResources.value.map(item => ({
        server_url: item.serverUrl.trim(),
        name: item.name.trim(),
      })),
    },
    api: {
      all: apiSelectAll.value,
      gateway_ids: apiSelectAll.value ? [] : selectedGatewayIds.value,
      ids: apiSelectAll.value ? [] : selectedApiIds.value,
      custom_resources: completeCustomApiResources.value.map(item => ({
        gateway_name: item.gatewayName.trim(),
        name: item.name.trim(),
      })),
    },
  },
});

const handleSubmit = async () => {
  try {
    await formRef.value?.validate();
  }
  catch {
    return;
  }
  if (!validateResources()) {
    return;
  }

  submitting.value = true;
  try {
    const payload = buildPayload();
    if (isEdit.value && token) {
      await updatePersonalToken(token.id, payload);
      messageSuccess(t('编辑成功'));
      isShow.value = false;
      emit('success');
      return;
    }
    createdResult.value = await createPersonalToken(payload);
    isShow.value = false;
    emit('success');
    await nextTick();
    createdDialogShow.value = true;
  }
  finally {
    submitting.value = false;
  }
};

const handleCancel = () => {
  isShow.value = false;
};

const handleCopyToken = async () => {
  if (!createdResult.value?.token) {
    return;
  }
  try {
    await navigator.clipboard.writeText(createdResult.value.token);
    messageSuccess(t('复制成功'));
  }
  catch {
    messageError(t('复制失败'));
  }
};

const resetState = () => {
  formData.value = {
    name: '',
    description: '',
    expiredAt: null,
    permanent: false,
  };
  mcpSectionExpanded.value = true;
  apiSectionExpanded.value = true;
  mcpSelectAll.value = false;
  apiSelectAll.value = false;
  selectedMcpIds.value = [];
  selectedGatewayIds.value = [];
  selectedApiIds.value = [];
  mcpSearchKey.value = '';
  apiSearchKey.value = '';
  mcpPage.value = 1;
  apiPage.value = 1;
  mcpPageSize.value = 10;
  apiPageSize.value = 10;
  mcpCollapsed.value = Object.fromEntries(mcpGroups.map(group => [group.id, true]));
  apiCollapsed.value = Object.fromEntries(apiGroups.map(group => [group.id, true]));
  customMcpResources.value = [];
  customApiResources.value = [];
  resourceError.value = false;
  nextTick(() => {
    formRef.value?.clearValidate();
  });
};

const resolveMcpResourceId = (resource: PersonalTokenDetail['mcp_resources'][number]) => {
  if (mcpResourceMap.value.has(resource.id)) {
    return resource.id;
  }
  const matched = [...mcpResourceMap.value.values()].find(item => item.item.name === resource.name);
  return matched?.item.id;
};

const resolveGatewayId = (resource: PersonalTokenDetail['gateway_resources'][number]) => {
  if (resource.id && apiGroupMap.value.has(resource.id)) {
    return resource.id;
  }
  return apiGroups.find(group => group.name === resource.name)?.id;
};

const resolveApiId = (resource: PersonalTokenDetail['api_resources'][number]) => {
  if (resource.id && apiResourceMap.value.has(resource.id)) {
    return resource.id;
  }
  const matched = [...apiResourceMap.value.values()].find(item => item.item.name === resource.name);
  return matched?.item.id;
};

const applyDetail = (detail: PersonalTokenDetail) => {
  formData.value = {
    name: detail.name,
    description: detail.description,
    expiredAt: detail.permanent ? null : parseDate(detail.expired_at),
    permanent: Boolean(detail.permanent),
  };
  mcpSelectAll.value = Boolean(detail.resource.mcp?.all);
  apiSelectAll.value = Boolean(detail.resource.api?.all);
  if (!mcpSelectAll.value) {
    selectedMcpIds.value = detail.mcp_resources
      .map(resolveMcpResourceId)
      .filter((id): id is string => Boolean(id));
  }
  if (!apiSelectAll.value) {
    selectedGatewayIds.value = detail.gateway_resources
      .map(resolveGatewayId)
      .filter((id): id is string => Boolean(id));
    selectedApiIds.value = detail.api_resources
      .map(resolveApiId)
      .filter((id): id is string => Boolean(id));
    const selectedGatewayApiIds = new Set(selectedGatewayIds.value.flatMap(id =>
      apiGroupMap.value.get(id)?.items.map(item => item.id) ?? []));
    selectedApiIds.value = selectedApiIds.value.filter(id => !selectedGatewayApiIds.has(id));
  }
  mcpGroups.forEach((group) => {
    if (group.items.some(item => selectedMcpIds.value.includes(item.id))) {
      mcpCollapsed.value[group.id] = false;
    }
  });
  apiGroups.forEach((group) => {
    if (selectedGatewayIds.value.includes(group.id)
      || group.items.some(item => selectedApiIds.value.includes(item.id))) {
      apiCollapsed.value[group.id] = false;
    }
  });
};

const initializeDrawer = async () => {
  resetState();
  if (!token?.id) {
    return;
  }
  const tokenId = token.id;
  initializing.value = true;
  try {
    const detail = await getPersonalTokenDetail(tokenId);
    if (isShow.value && token?.id === tokenId) {
      applyDetail(detail);
    }
  }
  finally {
    initializing.value = false;
  }
};

const handleClosed = () => {
  resetState();
};

const handleCreatedDialogClosed = () => {
  createdResult.value = null;
};

watch(isShow, (value) => {
  if (value) {
    initializeDrawer();
  }
}, { immediate: true });

</script>

<style scoped lang="scss">
.drawer-body {
  min-height: 720px;
  padding: 16px 24px 32px;

  :deep(.bk-form-label) {
    color: #63656e;
  }
}

.expired-field {
  display: flex;
  align-items: center;
  gap: 12px;

  .expired-picker {
    flex: 1;
    min-width: 0;
  }
}

.date-shortcuts {
  display: flex;
  flex-direction: column;
  width: 88px;
  padding: 8px 0;

  .date-shortcut-item {
    justify-content: flex-start;
    height: 32px;
    padding: 0 16px;
    font-size: 12px;
    color: #63656e;

    &:hover {
      color: #3a84ff;
      background-color: #f0f5ff;
    }
  }
}

.scope-section {
  margin-top: 4px;

  .section-title {
    font-size: 14px;
    color: #63656e;

    // height: 36px;
    // padding: 0 12px;
    // font-weight: 700;
    // line-height: 36px;
    // color: #4d4f56;
    // background-color: #f0f1f5;
    // border-bottom: 1px solid #dcdee5;
  }

  .scope-error {
    padding: 6px 12px;
    font-size: 12px;
    color: #ea3636;
    background-color: #feebea;
  }
}

.scope-card {
  margin-top: 12px;
  overflow: hidden;
  border: 1px solid #dcdee5;

  .scope-card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 40px;
    padding: 0 12px;
    background-color: #f0f1f5;
  }

  .scope-card-title {
    display: flex;
    align-items: center;
    min-width: 0;
    font-size: 12px;
    font-weight: 700;
    color: #4d4f56;
    cursor: pointer;
    gap: 8px;
  }

  .scope-arrow {
    transition: transform 0.2s;

    &.is-collapsed {
      transform: rotate(-90deg);
    }
  }

  .scope-select-all {
    flex-shrink: 0;
    font-size: 12px;

    :deep(.bk-checkbox-label) {
      font-size: 12px;
    }

    .select-all-text {
      font-weight: 700;
    }
  }
}

.resource-browser {
  display: grid;
  min-height: 392px;
  grid-template-columns: minmax(0, 2fr) minmax(280px, 1fr);

  .resource-list-pane {
    min-width: 0;
    background-color: #fff;
    border-right: 1px solid #dcdee5;
  }

  .resource-search {
    width: calc(100% - 16px);
    margin: 8px;
  }

  .search-icon {
    font-size: 14px;
    color: #979ba5;
  }

  .resource-group-list {
    height: 256px;
    overflow-y: auto;
    border-top: 1px solid #dcdee5;
  }

  .group-title {
    display: flex;
    align-items: center;
    min-width: 0;
    font-size: 12px;
    color: #4d4f56;
    gap: 6px;

    .group-count {
      color: #979ba5;
    }
  }

  .resource-checkbox-list {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  :deep(.resource-checkbox-list .bk-checkbox) {
    margin-left: 0;
    font-size: 12px;
  }

  .resource-empty {
    height: 220px;
    padding-top: 42px;
  }

  .resource-pagination {
    padding: 8px 12px;
    border-top: 1px solid #dcdee5;
  }

  :deep(.resource-pagination .bk-pagination) {
    font-size: 12px;
  }
}

.custom-resource-area {
  min-height: 48px;
  padding: 8px 12px;
  border-top: 1px solid #dcdee5;

  .add-custom-button {
    display: inline-flex;
    align-items: center;
    font-size: 12px;
    gap: 4px;
  }

  .custom-resource-row {
    display: grid;
    align-items: center;
    margin-top: 8px;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) 24px;
    gap: 8px;
  }

  .delete-custom-button {
    display: flex;
    align-items: center;
    justify-content: center;
    color: #979ba5;

    &:hover {
      color: #ea3636;
    }
  }
}

.selection-preview {
  min-width: 0;
  background-color: #f5f7fa;

  .preview-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 40px;
    padding: 0 12px;
    font-size: 12px;
    color: #4d4f56;
    background-color: #fff;
    border-bottom: 1px solid #dcdee5;
  }

  .preview-content {
    height: 352px;
    padding: 12px;
    overflow-y: auto;
  }

  .preview-group {

    & + .preview-group {
      margin-top: 16px;
    }
  }

  .preview-group-title {
    margin-bottom: 8px;
    font-size: 12px;
    color: #63656e;
  }

  .preview-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    min-height: 32px;
    padding: 4px 8px;
    margin-bottom: 6px;
    background-color: #fff;

    &:hover {
      background-color: #e1ecff;

      .remove-preview-button {
        visibility: visible;
      }
    }
  }

  .preview-item-main {
    display: flex;
    align-items: center;
    min-width: 0;
    gap: 4px;
  }

  .preview-item-name {
    overflow: hidden;
    font-size: 12px;
    color: #63656e;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .remove-preview-button {
    flex-shrink: 0;
    color: #979ba5;
    visibility: hidden;

    &:hover {
      color: #3a84ff;
    }
  }

  .preview-empty {
    padding-top: 88px;
  }
}

.created-dialog-body {
  padding: 4px 20px 8px;

  .created-success-icon {
    display: block;
    margin: 0 auto;
    font-size: 48px;
    color: #2dcb56;
  }

  .created-dialog-title {
    margin: 12px 0 20px;
    font-size: 20px;
    line-height: 28px;
    color: #313238;
    text-align: center;
  }

  .created-token-info {
    padding: 16px;
    margin-top: 12px;
    background-color: #f5f7fa;
  }

  .created-info-label {
    font-size: 12px;
    color: #979ba5;
  }

  .created-info-value,
  .created-token-value {
    margin-top: 4px;
    font-size: 14px;
    color: #313238;
  }

  .created-token-value {
    display: flex;
    align-items: flex-start;
    word-break: break-all;
    gap: 8px;

    span {
      flex: 1;
      min-width: 0;
    }
  }

  .copy-token-button {
    flex-shrink: 0;
    color: #3a84ff;
  }
}

:deep(.token-create-drawer .bk-sideslider-footer) {
  padding: 12px 24px;
}

:deep(.token-created-dialog .bk-dialog-footer) {
  text-align: center;
}

</style>
