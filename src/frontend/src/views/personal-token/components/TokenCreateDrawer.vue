<template>
  <!-- 新建/编辑令牌侧滑抽屉 -->
  <BkSideslider
    v-model:is-show="isShow"
    class="token-create-drawer"
    :width="960"
    quick-close
    @closed="handleClosed"
  >
    <!-- 抽屉标题 -->
    <template #header>
      {{ drawerTitle }}
    </template>

    <!-- 抽屉主体 -->
    <div
      v-bkloading="{ loading: initializing }"
      class="drawer-body"
    >
      <!-- 基础信息表单 -->
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
            :maxlength="255"
            :placeholder="t('用于说明令牌用途')"
          />
        </BkFormItem>
        <BkFormItem
          property="expiredAt"
          required
          :label="t('过期时间')"
        >
          <div class="expired-field">
            <BkDatePicker
              v-model="formData.expiredAt"
              class="expired-picker"
              type="datetime"
              append-to-body
              :clearable="false"
              :disabled-date="disabledDate"
              :placeholder="t('请选择过期时间')"
              @change="handleExpiredAtChange"
            >
              <!-- 过期时间快捷选项 -->
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
            <!--            个人令牌暂不支持永久有效 -->
            <!--            <BkCheckbox -->
            <!--              v-model="formData.permanent" -->
            <!--              disabled -->
            <!--            > -->
            <!--              {{ t('永久有效') }} -->
            <!--            </BkCheckbox> -->
          </div>
        </BkFormItem>
      </BkForm>

      <!-- 授权范围 -->
      <section class="scope-section">
        <div class="section-title">
          {{ t('授权范围') }}
        </div>
        <!-- 授权范围错误提示 -->
        <div
          v-if="resourceError"
          class="scope-error"
        >
          {{ t('请选择至少一项授权资源') }}
        </div>

        <!-- 资源类型分组 -->
        <div
          v-for="card in resourceCards"
          :key="card.resourceType.name"
          class="scope-card"
        >
          <!-- 资源类型标题与全选 -->
          <div class="scope-card-header">
            <div
              class="scope-card-title"
              @click="card.state.expanded = !card.state.expanded"
            >
              <CommonIcon
                class="scope-arrow"
                :class="{ 'is-collapsed': !card.state.expanded }"
                name="down-shape"
                color="#63656E"
                size="10"
              />
              <span>{{ card.resourceType.display_name }}（共 {{ card.state.count }} 个）</span>
            </div>
            <div
              v-if="card.resourceType.audience"
              v-bk-tooltips="getSelectAllTooltip(card.resourceType)"
              class="scope-select-all"
              @click.stop
            >
              <BkCheckbox
                :model-value="isAudienceSelected(card.resourceType.audience)"
                @change="(value: CheckboxValue) => handleResourceTypeAudienceChange(card.resourceType, value)"
              >
                <strong class="select-all-text">{{ t('全选') }}</strong>{{ getSelectAllDescription(card.resourceType) }}
              </BkCheckbox>
            </div>
          </div>

          <div
            v-show="card.state.expanded"
            class="scope-card-content"
          >
            <div class="resource-browser">
              <!-- 左侧资源选择区 -->
              <div class="resource-list-pane">
                <!-- 资源搜索 -->
                <BkInput
                  class="resource-search"
                  clearable
                  :model-value="card.state.keyword"
                  :placeholder="getSearchPlaceholder(card.resourceType)"
                  type="search"
                  @update:model-value="(value: CheckboxValue) => handleKeywordChange(
                    card.resourceType,
                    String(value ?? ''),
                  )"
                  @enter="() => handleSearch(card.resourceType)"
                  @clear="() => handleSearch(card.resourceType)"
                />

                <!-- 分组与平铺资源列表 -->
                <div
                  v-bkloading="{ loading: card.state.loading }"
                  class="resource-group-list"
                >
                  <template
                    v-for="resource in card.state.results"
                    :key="resource.name"
                  >
                    <CheckboxCollapse
                      v-if="resource.items"
                      v-model:collapsed="card.state.collapsed[resource.name]"
                      arrow-position="left"
                      compact
                      :auto-expand-on-check="false"
                      :checkbox-disabled="isResourceLocked(card.resourceType, resource)"
                      :model-value="isGrantableResourceSelected(resource)"
                      :show-checkbox="Boolean(resource.audience)"
                      @update:model-value="(value: boolean) => handleGrantableResourceChange(resource, value)"
                    >
                      <template #title>
                        <div class="group-title">
                          <span>{{ resource.display_name }}</span>
                          <span class="group-count">
                            {{ t('共 {total} 个，已选 {selected} 个', {
                              total: resource.items.length,
                              selected: getGrantableResourceSelectedCount(resource),
                            }) }}
                          </span>
                          <BkTag
                            v-if="resource.extras?.is_official === true"
                            size="small"
                            theme="info"
                          >
                            {{ t('官方') }}
                          </BkTag>
                        </div>
                      </template>
                      <div class="resource-checkbox-list">
                        <template
                          v-for="item in resource.items"
                          :key="item.audience || item.name"
                        >
                          <BkCheckbox
                            v-if="item.audience"
                            :disabled="isAudienceLocked(card.resourceType, item.audience)"
                            :model-value="isAudienceSelected(item.audience)"
                            size="small"
                            @change="(value: CheckboxValue) => handleAudienceChange(item.audience, value)"
                          >
                            {{ item.display_name }}（{{ item.name }}）
                          </BkCheckbox>
                          <span
                            v-else
                            class="resource-item-label"
                          >
                            {{ item.display_name }}（{{ item.name }}）
                          </span>
                        </template>
                      </div>
                    </CheckboxCollapse>
                    <div
                      v-else
                      class="flat-resource-list resource-checkbox-list"
                    >
                      <BkCheckbox
                        :disabled="isAudienceLocked(card.resourceType, resource.audience)"
                        :model-value="isAudienceSelected(resource.audience)"
                        size="small"
                        @change="(value: CheckboxValue) => handleAudienceChange(resource.audience, value)"
                      >
                        {{ resource.display_name }}（{{ resource.name }}）
                      </BkCheckbox>
                    </div>
                  </template>
                  <BkException
                    v-if="!card.state.results.length && !card.state.loading"
                    class="resource-empty"
                    type="empty"
                    scene="part"
                    :description="t('暂无匹配资源')"
                  />
                </div>

                <!-- 资源分页 -->
                <BkPagination
                  class="resource-pagination"
                  :count="card.state.count"
                  :limit="card.state.pageSize"
                  :limit-list="pageSizeOptions"
                  :model-value="card.state.page"
                  small
                  show-total-count
                  @update:limit="(value: CheckboxValue) => handlePageSizeChange(card.resourceType, Number(value))"
                  @update:model-value="(value: CheckboxValue) => handlePageChange(card.resourceType, Number(value))"
                />
                <!-- 非公开 MCP/API 添加区 -->
                <div
                  v-if="realm === 'blueking'
                    && (card.resourceType.name === 'mcp' || card.resourceType.name === 'api')"
                  class="custom-resource-area"
                >
                  <!-- 非公开资源添加浮层 -->
                  <BkPopover
                    :is-show="isCustomResourcePopoverShow
                      && customResourceTypeName === card.resourceType.name"
                    :width="360"
                    :padding="0"
                    ext-cls="custom-resource-popover"
                    placement="top-start"
                    theme="light"
                    trigger="manual"
                    @clickoutside="(payload: IPopoverClickOutsideEvent) => handleCustomResourcePopoverClickOutside(
                      card.resourceType.name,
                      payload,
                    )"
                  >
                    <BkButton
                      class="add-custom-button"
                      :disabled="isTypeSelectAllSelected(card.resourceType)"
                      theme="primary"
                      text
                      @click="() => handleOpenCustomResource(card.resourceType)"
                    >
                      <CommonIcon
                        name="plus-circle-shape"
                        class="mr-4px"
                      />
                      {{ card.resourceType.name === 'mcp' ? t('添加非公开 MCP') : t('添加非公开 API') }}
                    </BkButton>
                    <template #content>
                      <div class="custom-resource-popover-content">
                        <div class="custom-resource-popover-title">
                          {{ customResourcePopoverTitle }}
                        </div>
                        <BkForm
                          ref="customResourceFormRefs"
                          class="custom-resource-form"
                          form-type="vertical"
                          :model="customResourceFormData"
                          :rules="customResourceFormRules"
                        >
                          <BkFormItem
                            property="gatewayName"
                            required
                            :label="t('网关名称')"
                          >
                            <BkInput
                              :maxlength="240"
                              :model-value="customResourceFormData.gatewayName"
                              :placeholder="t('请输入网关名称')"
                              @update:model-value="(value: CheckboxValue) => handleCustomResourceInput(
                                'gatewayName',
                                String(value ?? ''),
                              )"
                            />
                          </BkFormItem>
                          <BkFormItem
                            property="resourceName"
                            required
                            :label="customResourceNameLabel"
                          >
                            <BkInput
                              :maxlength="240"
                              :model-value="customResourceFormData.resourceName"
                              :placeholder="customResourceNamePlaceholder"
                              @enter="handleAddCustomResource"
                              @update:model-value="(value: CheckboxValue) => handleCustomResourceInput(
                                'resourceName',
                                String(value ?? ''),
                              )"
                            />
                          </BkFormItem>
                        </BkForm>
                        <div class="custom-resource-popover-footer">
                          <BkButton
                            theme="primary"
                            :loading="customResourceLoading"
                            @click="handleAddCustomResource"
                          >
                            {{ t('确定') }}
                          </BkButton>
                          <BkButton @click="handleCloseCustomResourcePopover">
                            {{ t('取消') }}
                          </BkButton>
                        </div>
                      </div>
                    </template>
                  </BkPopover>
                  <!-- 已校验的非公开资源 -->
                  <div
                    v-if="getCustomResources(card.resourceType.name).length"
                    class="custom-resource-list"
                  >
                    <div
                      v-for="item in getCustomResources(card.resourceType.name)"
                      :key="item.resource.audience"
                      class="custom-resource-row"
                    >
                      <div class="custom-resource-info">
                        <span
                          v-bk-ellipsis
                          class="custom-resource-name"
                        >
                          {{ getCustomResourceDisplayName(item) }}
                        </span>
                        <BkTag
                          v-if="isNonPublicResource(item.resource)"
                          size="small"
                          theme="danger"
                        >
                          {{ t('非公开') }}
                        </BkTag>
                        <BkTag
                          size="small"
                          theme="success"
                        >
                          {{ t('校验通过') }}
                        </BkTag>
                      </div>
                      <BkButton
                        class="delete-custom-button"
                        text
                        @click="() => handleRemoveAudience(card.resourceType, item.resource.audience)"
                      >
                        <Del />
                      </BkButton>
                    </div>
                  </div>
                </div>
              </div>

              <!-- 已选资源预览 -->
              <div class="selection-preview">
                <div class="preview-header">
                  <span class="font-bold">
                    {{ t('选择结果预览（{count}）', { count: getTypeSelectedCount(card.resourceType) }) }}
                  </span>
                  <BkButton
                    v-if="hasRemovableSelection(card.resourceType)"
                    theme="primary"
                    text
                    @click="() => handleClearType(card.resourceType)"
                  >
                    {{ t('清空') }}
                  </BkButton>
                </div>
                <div class="preview-content">
                  <template v-if="getPreviewGroups(card.resourceType).length">
                    <div
                      v-for="group in getPreviewGroups(card.resourceType)"
                      :key="group.level"
                      class="preview-group"
                    >
                      <div
                        v-if="group.displayName"
                        class="preview-group-title"
                      >
                        【{{ group.displayName }}】- {{ t('共 {count} 个', { count: group.items.length }) }}
                      </div>
                      <div
                        v-for="item in group.items"
                        :key="item.audience"
                        class="preview-item"
                      >
                        <div class="preview-item-main">
                          <BkTag
                            v-if="group.displayName"
                            size="small"
                            :theme="group.level === 'gateway' ? 'warning' : undefined"
                          >
                            {{ group.displayName }}
                          </BkTag>
                          <BkTag
                            v-if="isNonPublicResource(item)"
                            size="small"
                            theme="danger"
                          >
                            {{ t('非公开') }}
                          </BkTag>
                          <span
                            v-bk-ellipsis
                            class="preview-item-name"
                          >{{ getPreviewItemName(item) }}</span>
                        </div>
                        <BkButton
                          class="remove-preview-button"
                          :disabled="isAudienceLocked(card.resourceType, item.audience)"
                          text
                          @click="() => handleRemoveAudience(card.resourceType, item.audience)"
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

    <!-- 抽屉操作栏 -->
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

  <!-- 创建结果中的令牌明文仅展示一次 -->
  <BkDialog
    v-model:is-show="createdDialogShow"
    class="token-created-dialog"
    width="520"
    :title="''"
    :quick-close="false"
    @closed="handleCreatedDialogClosed"
  >
    <div class="created-dialog-body">
      <div class="flex items-center justify-center">
        <Success class="created-success-icon" />
      </div>
      <div class="created-dialog-title">
        {{ t('令牌创建成功') }}
      </div>
      <BkAlert theme="warning">
        <template #title>
          {{ t('请妥善保存令牌，关闭弹窗后将无法再次查看它') }}
        </template>
      </BkAlert>
      <!-- 一次性明文令牌信息 -->
      <div class="created-token-info">
        <div class="created-info-label">
          {{ t('令牌名称') }}
        </div>
        <div class="created-info-value">
          {{ createdTokenName || '--' }}
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
    <!-- 创建结果操作栏 -->
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
import type { PersonalTokenRealm } from '@/constants/personal-token';
import { useEnv } from '@/stores';
import {
  type IGrantableResource,
  type IGrantableResourceType,
  type IPersonalToken,
  type IPersonalTokenCreateResult,
  type IPersonalTokenPayload,
  type IPersonalTokenResource,
  createPersonalToken,
  getGrantableResourceList,
  getGrantableResourceTypes,
  getPersonalTokenDetail,
  lookupGrantableResource,
  updatePersonalToken,
} from '@/services/source/personal-token';
import {
  messageError,
  messageSuccess,
  messageWarn,
} from '@/utils';
import {
  dateToUnixSeconds,
  getEndOfDay,
  getEstimatedMaxExpiresAt,
  unixSecondsToDate,
} from '../utils';

type DateShortcutChange = (date: Date, visible?: boolean) => void;
type CheckboxValue = boolean | number | string;
type CustomResourceField = 'gatewayName' | 'resourceName';

interface IProps {
  realm: PersonalTokenRealm
  token?: IPersonalToken | null
}

interface IEmits { success: [] }

interface IFormInstance {
  validate: (fields?: string | string[]) => Promise<Record<string, unknown>>
  clearValidate: (fields?: string | string[]) => void
}

interface IFormData {
  name: string
  description: string
  expiredAt: Date | string | null
  // permanent: boolean
}

interface IResourceState {
  expanded: boolean
  loading: boolean
  keyword: string
  page: number
  pageSize: number
  count: number
  results: IGrantableResource[]
  collapsed: Record<string, boolean>
  requestId: number
}

interface IResourceCard {
  resourceType: IGrantableResourceType
  state: IResourceState
}

interface IPreviewGroup {
  level: string
  displayName: string
  items: IPersonalTokenResource[]
}

interface ICustomResourceFormData {
  gatewayName: string
  resourceName: string
}

interface ICustomResourceItem {
  gatewayName: string
  resource: IPersonalTokenResource
}

interface ILookupErrorResponse {
  error?: {
    code?: string
    message?: string
  }
}

interface IPopoverClickOutsideEvent {
  event: MouseEvent
  isShow: boolean
}

const isShow = defineModel<boolean>('isShow', { default: false });

const {
  realm,
  token = null,
} = defineProps<IProps>();

const emit = defineEmits<IEmits>();

const envStore = useEnv();

const { t } = useI18n();

const formData = ref<IFormData>({
  name: '',
  description: '',
  expiredAt: null,
  // permanent: false,
});
const initializing = ref(false);
const submitting = ref(false);
const createdDialogShow = ref(false);
const createdResult = ref<IPersonalTokenCreateResult | null>(null);
const createdTokenName = ref('');
const resourceTypes = ref<IGrantableResourceType[]>([]);
// 各资源类型独立维护搜索、分页、折叠和请求状态
const resourceStates = ref<Record<string, IResourceState>>({});
// audience 是提交标识，映射表用于补齐已选资源的预览信息
const selectedAudience = ref<string[]>([]);
const selectedResourceMap = ref<Record<string, IPersonalTokenResource>>({});
// 精确查询成功的非公开资源按类型保留，用于左侧只读回显
const customResources = ref<Record<string, ICustomResourceItem[]>>({});
const customResourceTypeName = ref('');
const customResourceFormData = ref<ICustomResourceFormData>({
  gatewayName: '',
  resourceName: '',
});
const customResourceErrors = ref<ICustomResourceFormData>({
  gatewayName: '',
  resourceName: '',
});
const customResourceLoading = ref(false);
// 编辑时保留原始秒级时间，避免无改动时重复转换
const originalExpiresAt = ref<number | null>(null);
const resourceError = ref(false);

const formRef = useTemplateRef<IFormInstance>('formRef');
const customResourceFormRefs = useTemplateRef<IFormInstance[]>('customResourceFormRefs');

const pageSizeOptions = [5, 10, 20];
// 每种资源类型分别维护搜索防抖计时器
const searchTimers = new Map<string, ReturnType<typeof setTimeout>>();
// 标识最新抽屉初始化请求，防止切换令牌后旧响应回写
let drawerRequestId = 0;
// 标识最新非公开资源查询，关闭气泡后忽略旧响应
let customResourceRequestId = 0;

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
      validator: (value: Date | string | null) => Boolean(value),
      message: t('请选择过期时间'),
      trigger: 'change',
    },
    {
      validator: (value: Date | string | null) => !value
        || dateToUnixSeconds(value) > Math.floor(Date.now() / 1000),
      message: t('过期时间不能早于当前时间'),
      trigger: 'change',
    },
  ],
}));

const resourceCards = computed<IResourceCard[]>(() => resourceTypes.value
  .map(resourceType => ({
    resourceType,
    state: resourceStates.value[resourceType.name],
  }))
  .filter((card): card is IResourceCard => Boolean(card.state)));

const hasSelectedResource = computed(() => selectedAudience.value.some(Boolean));

const activeCustomResourceType = computed(() => resourceTypes.value
  .find(resourceType => resourceType.name === customResourceTypeName.value));

const isCustomResourcePopoverShow = computed(() => (
  realm === 'blueking' && Boolean(activeCustomResourceType.value)
));

const customResourcePopoverTitle = computed(() => (
  activeCustomResourceType.value?.name === 'mcp'
    ? t('添加非公开 MCP')
    : t('添加非公开 API')
));

const customResourceNameLabel = computed(() => (
  activeCustomResourceType.value?.name === 'mcp'
    ? t('MCP服务名称')
    : t('API名称')
));

const customResourceNamePlaceholder = computed(() => (
  activeCustomResourceType.value?.name === 'mcp'
    ? t('请输入 MCP 服务名称')
    : t('请输入 API 名称')
));

const customResourceNotFoundMessage = computed(() => (
  activeCustomResourceType.value?.name === 'mcp'
    ? t('未找到该 MCP 服务，请检查名称是否正确')
    : t('未找到该 API，请检查名称是否正确')
));

const customResourceFormRules = computed(() => ({
  gatewayName: [
    {
      validator: (value: string) => Boolean(value?.trim()) && !customResourceErrors.value.gatewayName,
      message: customResourceErrors.value.gatewayName || t('请输入网关名称'),
      trigger: 'blur',
    },
  ],
  resourceName: [
    {
      validator: (value: string) => Boolean(value?.trim()) && !customResourceErrors.value.resourceName,
      message: customResourceErrors.value.resourceName || customResourceNamePlaceholder.value,
      trigger: 'blur',
    },
  ],
}));

// 有效期限制在今天至后端允许的最大时间内
const disabledDate = (date: Date) => {
  const startOfToday = new Date();
  startOfToday.setHours(0, 0, 0, 0);
  return date.getTime() < startOfToday.getTime()
    || date.getTime() > getEstimatedMaxExpiresAt(
      envStore.env.personal_token_policy.max_ttl,
    ).getTime();
};

// 选择日期时统一使用当天最后一秒
const handleExpiredAtChange = (_value: string, selectionType?: string) => {
  if (selectionType !== 'date' || !formData.value.expiredAt) {
    return;
  }
  const date = getEndOfDay(formData.value.expiredAt);
  if (date) {
    formData.value.expiredAt = date;
  }
};

const handleDateShortcut = (days: number, change: DateShortcutChange) => {
  const date = new Date();
  date.setDate(date.getDate() + days);
  const endOfDay = getEndOfDay(date);
  if (!endOfDay) {
    return;
  }
  formData.value.expiredAt = endOfDay;
  change(endOfDay, false);
};

const createResourceState = (): IResourceState => ({
  expanded: true,
  loading: false,
  keyword: '',
  page: 1,
  pageSize: 10,
  count: 0,
  results: [],
  collapsed: {},
  requestId: 0,
});

const registerSelectedResource = (resource: IPersonalTokenResource) => {
  selectedResourceMap.value[resource.audience] = resource;
};

const registerGrantableResource = (
  resourceType: IGrantableResourceType,
  resource: IGrantableResource,
  level: string,
) => {
  if (resource.audience) {
    registerSelectedResource({
      type: resourceType.name,
      level,
      name: resource.name,
      display_name: resource.display_name,
      audience: resource.audience,
      extras: resource.extras,
    });
  }
};

// 将接口返回的分层资源摊平为 audience 到预览信息的映射
const registerGrantableResourceTree = (
  resourceType: IGrantableResourceType,
  resources: IGrantableResource[],
) => {
  const outerLevel = resourceType.levels[0]?.name ?? '';
  const innerLevel = resourceType.levels[resourceType.levels.length - 1]?.name ?? '';
  resources.forEach((resource) => {
    registerGrantableResource(resourceType, resource, resource.items ? outerLevel : innerLevel);
    resource.items?.forEach((item) => {
      registerGrantableResource(resourceType, item, innerLevel);
    });
  });
};

// 注册类型级 audience，用于包含后续新增资源的全选项
const registerResourceTypeAudience = (resourceType: IGrantableResourceType) => {
  if (!resourceType.audience) {
    return;
  }
  registerSelectedResource({
    type: resourceType.name,
    level: '',
    name: resourceType.name,
    display_name: '全部 ' + resourceType.display_name,
    audience: resourceType.audience,
  });
};

const getCustomResources = (typeName: string) => customResources.value[typeName] ?? [];

const registerCustomResource = (
  gatewayName: string,
  resource: IPersonalTokenResource,
) => {
  const typeResources = getCustomResources(resource.type);
  if (typeResources.some(item => item.resource.audience === resource.audience)) {
    return;
  }
  customResources.value[resource.type] = [
    ...typeResources,
    {
      gatewayName,
      resource,
    },
  ];
};

const removeCustomResource = (typeName: string, audience: string) => {
  customResources.value[typeName] = getCustomResources(typeName)
    .filter(item => item.resource.audience !== audience);
};

const getCustomResourceDisplayName = (item: ICustomResourceItem) => [
  item.gatewayName,
  item.resource.display_name,
].filter(Boolean).join(' / ');

const getGatewayNameFromAudience = (audience: string) => {
  const gatewayAudience = audience.split('/')[0] ?? '';
  return gatewayAudience.startsWith('gateway:')
    ? gatewayAudience.slice('gateway:'.length)
    : '';
};

const getResourceState = (typeName: string) => resourceStates.value[typeName];

// 后端使用“末级层级:关键字”的格式搜索资源
const buildKeyword = (resourceType: IGrantableResourceType, keyword: string) => {
  const levelName = resourceType.levels[resourceType.levels.length - 1]?.name;
  return levelName && keyword ? levelName + ':' + keyword : undefined;
};

// 加载单类可授权资源，并丢弃过期请求的结果
const fetchGrantableResources = async (
  resourceType: IGrantableResourceType,
  currentRealm = realm,
) => {
  const state = getResourceState(resourceType.name);
  if (!state) {
    return;
  }
  const requestId = state.requestId + 1;
  state.requestId = requestId;
  state.loading = true;
  try {
    const result = await getGrantableResourceList(currentRealm, {
      type: resourceType.name,
      keyword: buildKeyword(resourceType, state.keyword),
      page: state.page,
      page_size: state.pageSize,
    });
    if (
      state.requestId !== requestId
      || resourceStates.value[resourceType.name] !== state
      || realm !== currentRealm
      || !isShow.value
    ) {
      return;
    }
    state.count = result.count;
    state.results = result.results;
    result.results.forEach((resource) => {
      if (resource.items) {
        state.collapsed[resource.name] = state.keyword
          ? false
          : (state.collapsed[resource.name] ?? true);
      }
    });
    registerGrantableResourceTree(resourceType, result.results);
  }
  finally {
    if (state.requestId === requestId) {
      state.loading = false;
    }
  }
};

const getSelectableResources = (resources: IGrantableResource[]): IGrantableResource[] =>
  resources.flatMap(resource => [
    ...(resource.audience ? [resource] : []),
    ...getSelectableResources(resource.items ?? []),
  ]);

// 新建时仅有一种类型和一个可选资源则默认选中
const selectOnlyGrantableResource = () => {
  if (isEdit.value || hasSelectedResource.value || !resourceTypes.value.length || resourceTypes.value.length > 1) {
    return;
  }
  const resourceType = resourceTypes.value[0];
  const state = getResourceState(resourceType!.name);
  if (!state || state.count !== 1) {
    return;
  }
  const selectableResources = getSelectableResources(state.results);
  if (selectableResources.length === 1) {
    selectedAudience.value = [selectableResources[0]!.audience];
    resourceError.value = false;
  }
};

// 初始化资源类型及各类型的首屏资源
const loadResourceTypes = async (currentRealm: PersonalTokenRealm, requestId: number) => {
  const types = await getGrantableResourceTypes(currentRealm);
  if (drawerRequestId !== requestId || realm !== currentRealm || !isShow.value) {
    return;
  }
  resourceStates.value = Object.fromEntries(types.map(type => [type.name, createResourceState()]));
  resourceTypes.value = types;
  types.forEach(registerResourceTypeAudience);
  await Promise.all(types.map(type => fetchGrantableResources(type, currentRealm)));
  if (drawerRequestId === requestId && realm === currentRealm && isShow.value) {
    selectOnlyGrantableResource();
  }
};

// 清洗搜索关键字，并按资源类型分别防抖查询
const handleKeywordChange = (resourceType: IGrantableResourceType, value: string) => {
  const state = getResourceState(resourceType.name);
  if (!state) {
    return;
  }
  state.keyword = value.replace(/[,:]/g, '');
  const currentTimer = searchTimers.get(resourceType.name);
  if (currentTimer) {
    clearTimeout(currentTimer);
  }
  searchTimers.set(resourceType.name, setTimeout(() => {
    state.page = 1;
    fetchGrantableResources(resourceType);
  }, 300));
};

const handleSearch = (resourceType: IGrantableResourceType) => {
  const state = getResourceState(resourceType.name);
  if (!state) {
    return;
  }
  state.page = 1;
  fetchGrantableResources(resourceType);
};

const handlePageChange = (resourceType: IGrantableResourceType, page: number) => {
  const state = getResourceState(resourceType.name);
  if (!state || state.page === page) {
    return;
  }
  state.page = page;
  fetchGrantableResources(resourceType);
};

const handlePageSizeChange = (resourceType: IGrantableResourceType, pageSize: number) => {
  const state = getResourceState(resourceType.name);
  if (!state || state.pageSize === pageSize) {
    return;
  }
  state.page = 1;
  state.pageSize = Math.min(pageSize, 20);
  fetchGrantableResources(resourceType);
};

const isAudienceSelected = (audience: string) => selectedAudience.value.includes(audience);

const isTypeSelectAllSelected = (resourceType: IGrantableResourceType) =>
  Boolean(resourceType.audience && isAudienceSelected(resourceType.audience));

const handleAudienceChange = (audience: string, value: CheckboxValue) => {
  if (!audience) {
    return;
  }
  const checked = Boolean(value);
  if (checked && !isAudienceSelected(audience)) {
    selectedAudience.value = [...selectedAudience.value, audience];
  }
  if (!checked) {
    selectedAudience.value = selectedAudience.value.filter(item => item !== audience);
  }
  resourceError.value = false;
};

const resetCustomResourceForm = () => {
  customResourceFormData.value = {
    gatewayName: '',
    resourceName: '',
  };
  customResourceErrors.value = {
    gatewayName: '',
    resourceName: '',
  };
  nextTick(() => {
    customResourceFormRefs.value?.[0]?.clearValidate();
  });
};

const handleCloseCustomResourcePopover = () => {
  customResourceRequestId += 1;
  customResourceTypeName.value = '';
  customResourceLoading.value = false;
  resetCustomResourceForm();
};

const handleOpenCustomResource = (resourceType: IGrantableResourceType) => {
  if (
    realm !== 'blueking'
    || isTypeSelectAllSelected(resourceType)
  ) {
    return;
  }
  if (customResourceTypeName.value === resourceType.name) {
    handleCloseCustomResourcePopover();
    return;
  }
  customResourceRequestId += 1;
  resetCustomResourceForm();
  customResourceTypeName.value = resourceType.name;
};

const handleCustomResourcePopoverClickOutside = (
  typeName: string,
  {
    event,
    isShow,
  }: IPopoverClickOutsideEvent,
) => {
  if (!isShow || customResourceTypeName.value !== typeName) {
    return;
  }
  if ((event.target as HTMLElement | null)?.closest('.add-custom-button')) {
    return;
  }
  handleCloseCustomResourcePopover();
};

const handleCustomResourceInput = (
  field: CustomResourceField,
  value: string,
) => {
  customResourceFormData.value[field] = value.replace(/[,:]/g, '');
  customResourceErrors.value[field] = '';
  customResourceFormRefs.value?.[0]?.clearValidate(field);
};

const getLookupError = (error: unknown) => (error as ILookupErrorResponse)?.error;

const showCustomResourceFieldError = async (
  field: CustomResourceField,
  message: string,
) => {
  customResourceErrors.value[field] = message;
  await nextTick();
  try {
    await customResourceFormRefs.value?.[0]?.validate(field);
  }
  catch {
    // 校验失败即由 BkFormItem 展示当前错误
  }
  messageError(t('添加失败，{message}', { message }));
};

// 按 blueking 两级名称精确查询，并复用已有 audience 选中与预览逻辑
const handleAddCustomResource = async () => {
  const resourceType = activeCustomResourceType.value;
  if (
    realm !== 'blueking'
    || !resourceType
    || isTypeSelectAllSelected(resourceType)
  ) {
    return;
  }

  customResourceErrors.value = {
    gatewayName: '',
    resourceName: '',
  };
  try {
    await customResourceFormRefs.value?.[0]?.validate();
  }
  catch {
    return;
  }

  const gatewayName = customResourceFormData.value.gatewayName.trim();
  const resourceName = customResourceFormData.value.resourceName.trim();
  if (!gatewayName || !resourceName) {
    return;
  }

  const requestId = customResourceRequestId + 1;
  customResourceRequestId = requestId;
  customResourceLoading.value = true;
  try {
    const resource = await lookupGrantableResource(realm, {
      type: resourceType.name,
      name: `gateway:${gatewayName},${resourceType.name}:${resourceName}`,
    });
    if (
      customResourceRequestId !== requestId
      || activeCustomResourceType.value?.name !== resourceType.name
      || !isShow.value
    ) {
      return;
    }
    if (!resource.audience) {
      messageError(t('可授予资源查询失败，请稍后重试'));
      return;
    }

    const selectedResource: IPersonalTokenResource = {
      type: resourceType.name,
      level: resourceType.levels[resourceType.levels.length - 1]?.name ?? resourceType.name,
      name: resource.name,
      display_name: resource.display_name,
      audience: resource.audience,
      extras: resource.extras,
    };
    registerSelectedResource(selectedResource);
    registerCustomResource(gatewayName, selectedResource);
    handleAudienceChange(selectedResource.audience, true);
    messageSuccess(t('添加成功'));
    handleCloseCustomResourcePopover();
  }
  catch (error) {
    if (customResourceRequestId !== requestId) {
      return;
    }
    const requestError = getLookupError(error);
    if (requestError?.code === 'NOT_FOUND') {
      const gatewayNotFound = requestError.message?.includes('no gateway named') === true;
      await showCustomResourceFieldError(
        gatewayNotFound ? 'gatewayName' : 'resourceName',
        gatewayNotFound
          ? t('未找到该网关，请检查名称是否正确')
          : customResourceNotFoundMessage.value,
      );
      return;
    }
    if (requestError?.code === 'INVALID_ARGUMENT') {
      await showCustomResourceFieldError(
        'resourceName',
        t('资源信息校验失败，请检查名称是否正确'),
      );
      return;
    }
    messageError(t('可授予资源查询失败，请稍后重试'));
  }
  finally {
    if (customResourceRequestId === requestId) {
      customResourceLoading.value = false;
    }
  }
};

// 分组资源优先使用自身 audience，否则汇总所有子资源 audience
const getGrantableResourceAudiences = (resource: IGrantableResource) => {
  if (resource.audience) {
    return [resource.audience];
  }
  return resource.items?.map(item => item.audience).filter(Boolean) ?? [];
};

const isGrantableResourceSelected = (resource: IGrantableResource) => {
  const audiences = getGrantableResourceAudiences(resource);
  return audiences.length > 0 && audiences.every(isAudienceSelected);
};

const handleGrantableResourceChange = (resource: IGrantableResource, value: boolean) => {
  getGrantableResourceAudiences(resource).forEach((audience) => {
    handleAudienceChange(audience, value);
  });
};

const getGrantableResourceSelectedCount = (resource: IGrantableResource) => {
  if (resource.audience && isAudienceSelected(resource.audience)) {
    return resource.items?.length ?? 1;
  }
  return resource.items?.filter(item => isAudienceSelected(item.audience)).length ?? 0;
};

const isAudienceLocked = (resourceType: IGrantableResourceType, audience: string) =>
  isTypeSelectAllSelected(resourceType) && audience !== resourceType.audience;

const isResourceLocked = (
  resourceType: IGrantableResourceType,
  resource: IGrantableResource,
) => getGrantableResourceAudiences(resource)
  .some(audience => isAudienceLocked(resourceType, audience));

const getSelectAllDescription = (resourceType: IGrantableResourceType) => {
  if (resourceType.name === 'mcp') {
    return t('，包括后续新增的 MCP 也生效');
  }
  return t('，包括后续新增的 API 也生效');
};

const getSelectAllTooltip = (resourceType: IGrantableResourceType) => (
  resourceType.name === 'mcp'
    ? t('勾选后将自动包含后续新增的 MCP')
    : t('勾选后将自动包含后续新增的 API')
);

const getSearchPlaceholder = (resourceType: IGrantableResourceType) => {
  if (resourceType.name === 'mcp') {
    return t('搜索网关、MCP 名称');
  }
  if (resourceType.name === 'api') {
    return t('搜索网关、API 名称');
  }
  return resourceType.display_name;
};

// 按类型收集预览资源，无法识别的历史 audience 归入首个分组
const getTypePreviewResources = (resourceType: IGrantableResourceType) => {
  const firstTypeName = resourceTypes.value[0]?.name;
  return selectedAudience.value.reduce<IPersonalTokenResource[]>((result, audience) => {
    const resource = selectedResourceMap.value[audience];
    if (resource?.type === resourceType.name) {
      result.push(resource);
    }
    else if (!resource && resourceType.name === firstTypeName) {
      result.push({
        type: '',
        level: '__raw__',
        name: audience,
        display_name: audience,
        audience,
      });
    }
    return result;
  }, []);
};

// 类型级全选与该类型的具体资源互斥
const handleResourceTypeAudienceChange = (
  resourceType: IGrantableResourceType,
  value: CheckboxValue,
) => {
  if (!resourceType.audience) {
    return;
  }
  const typeAudiences = new Set(
    getTypePreviewResources(resourceType).map(resource => resource.audience),
  );
  selectedAudience.value = selectedAudience.value.filter(audience => !typeAudiences.has(audience));
  if (value) {
    customResources.value[resourceType.name] = [];
    selectedAudience.value = [...selectedAudience.value, resourceType.audience];
  }
  resourceError.value = false;
};

// 按资源层级生成稳定顺序的预览分组
const getPreviewGroups = (resourceType: IGrantableResourceType): IPreviewGroup[] => {
  const resources = getTypePreviewResources(resourceType);
  const levels = [
    {
      name: '',
      display_name: 'ALL',
    },
    ...resourceType.levels,
    {
      name: '__raw__',
      display_name: '',
    },
  ];
  return levels
    .map(level => ({
      level: level.name,
      displayName: level.display_name,
      items: resources.filter(resource => resource.level === level.name),
    }))
    .filter(group => group.items.length > 0);
};

const getTypeSelectedCount = (resourceType: IGrantableResourceType) =>
  getTypePreviewResources(resourceType).length;

const hasRemovableSelection = (resourceType: IGrantableResourceType) =>
  getTypePreviewResources(resourceType)
    .some(resource => !isAudienceLocked(resourceType, resource.audience));

const handleRemoveAudience = (resourceType: IGrantableResourceType, audience: string) => {
  if (!isAudienceLocked(resourceType, audience)) {
    handleAudienceChange(audience, false);
    removeCustomResource(resourceType.name, audience);
  }
};

const handleClearType = (resourceType: IGrantableResourceType) => {
  const removableAudiences = new Set(
    getTypePreviewResources(resourceType)
      .filter(resource => !isAudienceLocked(resourceType, resource.audience))
      .map(resource => resource.audience),
  );
  selectedAudience.value = selectedAudience.value.filter(audience => !removableAudiences.has(audience));
  customResources.value[resourceType.name] = [];
};

const isNonPublicResource = (resource: IPersonalTokenResource) =>
  Object.prototype.hasOwnProperty.call(resource.extras ?? {}, 'is_public')
  && resource.extras?.is_public === false;

const getPreviewItemName = (resource: IPersonalTokenResource) => (
  resource.name === resource.display_name
    ? resource.display_name
    : resource.display_name + '（' + resource.name + '）'
);

const validateResources = () => {
  if (!hasSelectedResource.value) {
    resourceError.value = true;
    messageWarn(t('请选择至少一项授权资源'));
    return false;
  }
  return true;
};

// 编辑时复用原始过期时间，其他字段统一转换为接口格式
const buildPayload = (): IPersonalTokenPayload => {
  const selectedExpiresAt = formData.value.expiredAt
    ? dateToUnixSeconds(formData.value.expiredAt)
    : 0;
  const expiresAt = originalExpiresAt.value !== null
    && selectedExpiresAt === originalExpiresAt.value
    ? originalExpiresAt.value
    : selectedExpiresAt;
  return {
    name: formData.value.name.trim(),
    description: formData.value.description.trim(),
    audience: selectedAudience.value.filter(Boolean),
    expires_at: expiresAt,
  };
};

// 根据抽屉模式分别执行新建或编辑流程
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
      await updatePersonalToken(realm, token.id, payload);
      messageSuccess(t('编辑成功'));
      isShow.value = false;
      emit('success');
      return;
    }
    createdTokenName.value = payload.name;
    createdResult.value = await createPersonalToken(realm, payload);
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
  searchTimers.forEach(timer => clearTimeout(timer));
  searchTimers.clear();
  formData.value = {
    name: '',
    description: '',
    expiredAt: null,
    // permanent: false,
  };
  resourceTypes.value = [];
  resourceStates.value = {};
  selectedAudience.value = [];
  selectedResourceMap.value = {};
  customResources.value = {};
  customResourceRequestId += 1;
  customResourceTypeName.value = '';
  customResourceLoading.value = false;
  customResourceFormData.value = {
    gatewayName: '',
    resourceName: '',
  };
  customResourceErrors.value = {
    gatewayName: '',
    resourceName: '',
  };
  originalExpiresAt.value = null;
  resourceError.value = false;
  nextTick(() => {
    formRef.value?.clearValidate();
    customResourceFormRefs.value?.[0]?.clearValidate();
  });
};

// 回填编辑数据，并展开包含已选资源的分组
const applyDetail = (detail: IPersonalToken) => {
  formData.value = {
    name: detail.name,
    description: detail.description,
    expiredAt: unixSecondsToDate(detail.expires_at),
    // permanent: false,
  };
  originalExpiresAt.value = detail.expires_at;
  selectedAudience.value = detail.audience.filter(Boolean);
  detail.resources?.forEach((resource) => {
    registerSelectedResource(resource);
    if (
      realm === 'blueking'
      && (resource.type === 'mcp' || resource.type === 'api')
      && isNonPublicResource(resource)
    ) {
      registerCustomResource(getGatewayNameFromAudience(resource.audience), resource);
    }
  });
  if (detail.resources) {
    resourceTypes.value
      .filter(resourceType => isTypeSelectAllSelected(resourceType))
      .forEach(resourceType => handleResourceTypeAudienceChange(resourceType, true));
  }
  resourceCards.value.forEach(({ state }) => {
    state.results.forEach((resource) => {
      if (
        resource.items
        && getGrantableResourceAudiences(resource).some(isAudienceSelected)
      ) {
        state.collapsed[resource.name] = false;
      }
    });
  });
};

// 并行加载资源和令牌详情，仅应用当前抽屉的初始化结果
const initializeDrawer = async () => {
  resetState();
  const requestId = drawerRequestId + 1;
  drawerRequestId = requestId;
  const tokenId = token?.id;
  const currentRealm = realm;
  initializing.value = true;
  try {
    const detailPromise = tokenId
      ? getPersonalTokenDetail(currentRealm, tokenId)
      : Promise.resolve(null);
    await loadResourceTypes(currentRealm, requestId);
    const detail = await detailPromise;
    if (
      drawerRequestId === requestId
      && isShow.value
      && realm === currentRealm
      && token?.id === tokenId
      && detail
    ) {
      applyDetail(detail);
    }
  }
  finally {
    if (drawerRequestId === requestId) {
      initializing.value = false;
    }
  }
};

const handleClosed = () => {
  // 使抽屉关闭前尚未完成的初始化请求失效
  drawerRequestId += 1;
  initializing.value = false;
  resetState();
};

const handleCreatedDialogClosed = () => {
  createdResult.value = null;
  createdTokenName.value = '';
};

watch(
  [isShow, () => realm],
  ([visible]) => {
    if (visible) {
      initializeDrawer();
    }
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  searchTimers.forEach(timer => clearTimeout(timer));
  searchTimers.clear();
});
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

  .resource-item-label {
    font-size: 12px;
    line-height: 20px;
    color: #63656e;
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

  .custom-resource-list {
    margin-top: 8px;
  }

  .custom-resource-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    min-height: 40px;
    padding: 0 8px;
    margin-top: 8px;
    border: 1px solid #dcdee5;

    &:first-child {
      margin-top: 0;
    }
  }

  .custom-resource-info {
    display: flex;
    align-items: center;
    min-width: 0;
    gap: 6px;
  }

  .custom-resource-name {
    min-width: 0;
    overflow: hidden;
    font-size: 12px;
    color: #4d4f56;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  :deep(.custom-resource-row .bk-tag) {
    flex-shrink: 0;
  }

  .delete-custom-button {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    width: 24px;
    height: 24px;
    color: #979ba5;

    &:hover {
      color: #ea3636;
    }
  }
}

.custom-resource-popover-content {
  color: #4d4f56;
  background-color: #fff;

  .custom-resource-popover-title {
    padding: 18px 20px 8px;
    font-size: 14px;
    font-weight: 700;
    line-height: 24px;
  }

  .custom-resource-form {
    padding: 8px 20px 4px;

    :deep(.bk-form-label) {
      font-size: 12px;
      color: #63656e;
    }
  }

  .custom-resource-popover-footer {
    display: flex;
    padding: 10px 20px;
    background:#FAFBFD;
    border-top: 1px solid #dcdee5;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
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
    margin-bottom: 2px;
    background-color: #fff;
    border-radius: 2px;

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
    cursor: pointer;
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

.flat-resource-list {
  padding: 12px 28px;
}

</style>

<style lang="scss">
.custom-resource-popover {
  padding: 0 !important;
}
</style>
