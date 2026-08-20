<template>
  <BkDialog
    v-model:is-show="isShow"
    class="renew-dialog"
    width="640"
    :title="''"
    :quick-close="false"
    @closed="handleClosed"
  >
    <template #header>
      <div class="renew-dialog-header">
        <span class="header-title">续期令牌</span>
        <span class="header-divider">|</span>
        <span class="header-name">{{ token?.name }}</span>
      </div>
    </template>

    <div class="renew-dialog-body">
      <!-- 续期方式 -->
      <div class="field-label">
        续期时间
      </div>
      <div class="preset-group">
        <div
          v-for="item in presets"
          :key="item.value"
          class="preset-item"
          :class="{ 'is-active': selectedType === item.value }"
          @click="handleSelectPreset(item.value)"
        >
          {{ item.label }}
        </div>
      </div>

      <!-- 自定义过期时间 -->
      <BkDatePicker
        v-if="selectedType === 'custom'"
        v-model="customDate"
        class="custom-date-picker mt-12px"
        type="datetime"
        placeholder="如：2025-01-30 12:12:21"
        append-to-body
        :clearable="false"
        :disabled-date="disabledDate"
      />

      <!-- 续期前后时间对比 -->
      <div class="expired-info mt-12px">
        <span class="info-label">当前过期时间：</span>
        <span class="info-value">{{ currentExpiredAtText }}</span>
        <span class="info-arrow">→</span>
        <span class="info-label">续期后过期时间：</span>
        <span class="info-value">{{ newExpiredAtText }}</span>
      </div>
    </div>

    <template #footer>
      <BkButton
        theme="primary"
        :loading="submitting"
        :disabled="confirmDisabled"
        @click="handleConfirm"
      >
        确定
      </BkButton>
      <BkButton
        class="ml-8px"
        @click="handleCancel"
      >
        取消
      </BkButton>
    </template>
  </BkDialog>
</template>

<script setup lang="ts">
import { messageSuccess } from '@/utils';
import type { PersonalTokenRealm } from '@/constants/personal-token';
import { useEnv } from '@/stores';
import {
  type IPersonalToken,
  renewPersonalToken,
} from '@/services/source/personal-token';
import {
  dateToUnixSeconds,
  formatUnixSeconds,
  getEstimatedMaxExpiresAt,
  unixSecondsToDate,
} from '../utils';

type PresetValue = 'days30' | 'days90' | 'custom';

interface PresetOption {
  label: string
  value: PresetValue
  /** 续期天数，仅时长类预设有效 */
  days?: number
}

interface IProps {
  realm: PersonalTokenRealm
  token?: IPersonalToken | null
}

const isShow = defineModel<boolean>('isShow', { default: false });

const {
  realm,
  token = null,
} = defineProps<IProps>();

const emit = defineEmits<{ success: [] }>();

const envStore = useEnv();

const presets: PresetOption[] = [
  {
    label: '+ 30 天',
    value: 'days30',
    days: 30,
  },
  {
    label: '+ 90 天',
    value: 'days90',
    days: 90,
  },
  {
    label: '自定义',
    value: 'custom',
  },
];

const selectedType = ref<PresetValue>('days30');
const customDate = ref<Date | string>('');
const submitting = ref(false);

const formatDate = (date: Date) => {
  const pad = (num: number) => String(num).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} `
    + `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
};

// 预设时长，若令牌已过期，从当前服务器时间开始算；若未过期，当前过期时间起算
// 自定义模式直接使用所选时间
const computedExpiredAt = computed<Date | null>(() => {
  if (!token) {
    return null;
  }
  if (selectedType.value === 'custom') {
    return customDate.value ? new Date(dateToUnixSeconds(customDate.value) * 1000) : null;
  }
  const preset = presets.find(item => item.value === selectedType.value);
  const base = unixSecondsToDate(Math.max(token.expires_at, Date.now() / 1000));
  base.setDate(base.getDate() + (preset?.days ?? 0));
  return base;
});

const newExpiredAtText = computed(() => {
  return computedExpiredAt.value ? formatDate(computedExpiredAt.value) : '--';
});
const currentExpiredAtText = computed(() => formatUnixSeconds(token?.expires_at));

// 自定义模式下未选择时间时禁用确定按钮
const confirmDisabled = computed(() => selectedType.value === 'custom' && !customDate.value);

// 自定义时间限制在今天至后端允许的最大时间内
const disabledDate = (date: Date) => {
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  return date.getTime() < today.getTime()
    || date.getTime() > getEstimatedMaxExpiresAt(
      envStore.env.personal_token_policy.max_ttl,
    ).getTime();
};

const handleSelectPreset = (value: PresetValue) => {
  selectedType.value = value;
};

const resetState = () => {
  selectedType.value = 'days30';
  customDate.value = '';
};

const handleConfirm = async () => {
  if (!token || confirmDisabled.value) {
    return;
  }
  const expiresAt = computedExpiredAt.value;
  if (!expiresAt) {
    return;
  }

  submitting.value = true;
  try {
    await renewPersonalToken(realm, token.id, { expires_at: dateToUnixSeconds(expiresAt) });
    messageSuccess('续期成功');
    isShow.value = false;
    emit('success');
  }
  finally {
    submitting.value = false;
  }
};

const handleCancel = () => {
  isShow.value = false;
};

const handleClosed = () => {
  resetState();
};

watch(isShow, (val) => {
  if (val) {
    resetState();
  }
});
</script>

<style scoped lang="scss">
.renew-dialog-header {
  display: flex;
  align-items: center;
  font-size: 20px;
  line-height: 28px;

  .header-title {
    color: #313238;
  }

  .header-divider {
    margin: 0 10px;
    color: #dcdee5;
  }

  .header-name {
    font-size: 16px;
    color: #979ba5;
  }
}

.renew-dialog-body {
  padding-bottom: 8px;

  .field-label {
    margin-bottom: 8px;
    font-size: 14px;
    color: #63656e;
  }

  .preset-group {
    display: flex;
    width: 100%;

    .preset-item {
      display: flex;
      flex: 1;
      align-items: center;
      justify-content: center;
      height: 32px;
      margin-left: -1px;
      font-size: 14px;
      color: #63656e;
      cursor: pointer;
      background-color: #fff;
      border: 1px solid #c4c6cc;

      &:first-child {
        margin-left: 0;
        border-radius: 2px 0 0 2px;
      }

      &:last-child {
        border-radius: 0 2px 2px 0;
      }

      &:hover {
        position: relative;
        z-index: 1;
        color: #3a84ff;
      }

      &.is-active {
        position: relative;
        z-index: 2;
        color: #3a84ff;
        background-color: #f0f5ff;
        border-color: #3a84ff;
      }
    }
  }

  .custom-date-picker {
    width: 100%;
  }

  .expired-info {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    font-size: 14px;
    color: #979ba5;

    .info-value {
      color: #63656e;
    }

    .info-arrow {
      margin: 0 8px;
      color: #979ba5;
    }
  }
}
</style>
