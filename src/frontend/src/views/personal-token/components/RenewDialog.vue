<template>
  <BkDialog
    :is-show="isShow"
    title="续期令牌"
    :width="480"
    :is-loading="submitting"
    @confirm="handleConfirm"
    @closed="handleClose"
  >
    <div
      v-if="token"
      class="renew"
    >
      <div class="renew__field">
        <label>令牌名称</label>
        <div class="renew__value">
          {{ token.name }}
        </div>
      </div>

      <div class="renew__field">
        <label>续期时长</label>
        <BkRadioGroup
          v-model="mode"
          type="capsule"
        >
          <BkRadioButton
            v-for="preset in RENEW_PRESETS"
            :key="preset.seconds"
            :label="String(preset.seconds)"
          >
            {{ preset.label }}
          </BkRadioButton>
          <BkRadioButton label="custom">
            自定义
          </BkRadioButton>
        </BkRadioGroup>
        <BkDatePicker
          v-if="mode === 'custom'"
          v-model="customExpiry"
          class="mt-8px w-full"
          type="datetime"
          :clearable="false"
          placeholder="请选择续期后截止时间"
        />
      </div>

      <div class="renew__preview">
        <div>
          <span class="renew__preview-label">原截止时间</span>
          <span>{{ formatDateTime(token.expires_at) }}</span>
        </div>
        <div>
          <span class="renew__preview-label">续期后截止时间</span>
          <span class="renew__preview-new">{{ previewExpiry }}</span>
        </div>
      </div>
    </div>
  </BkDialog>
</template>

<script setup lang="ts">
import dayjs from 'dayjs';
import { PERSONAL_TOKEN_REALM, type PersonalToken, renewPersonalToken } from '@/services/source/personalToken';
import { DEFAULT_TTL_DAYS, DEFAULT_TTL_SECONDS, RENEW_PRESETS, formatDateTime } from '../utils';
import { messageError, messageSuccess } from '@/utils';

const { isShow, token, realm = PERSONAL_TOKEN_REALM } = defineProps<{
  isShow: boolean
  token: PersonalToken | null
  /** 所属 realm，来自路由 */
  realm?: string
}>();

const emit = defineEmits<{
  'update:isShow': [value: boolean]
  'success': []
}>();

const mode = ref<string>(String(DEFAULT_TTL_SECONDS));
const customExpiry = ref<Date>(dayjs().add(DEFAULT_TTL_DAYS, 'day').toDate());
const submitting = ref(false);

watch(
  () => isShow,
  (show) => {
    if (show) {
      mode.value = String(DEFAULT_TTL_SECONDS);
      customExpiry.value = dayjs().add(DEFAULT_TTL_DAYS, 'day').toDate();
    }
  },
);

const resolveExpiresIn = (): number | null => {
  if (mode.value === 'custom') {
    const seconds = dayjs(customExpiry.value).diff(dayjs(), 'second');
    return seconds > 0 ? seconds : null;
  }
  return Number(mode.value);
};

const previewExpiry = computed(() => {
  const seconds = resolveExpiresIn();
  if (seconds === null) {
    return '--';
  }
  return dayjs().add(seconds, 'second').format('YYYY-MM-DD HH:mm:ss');
});

const handleClose = () => {
  if (submitting.value) {
    return;
  }
  emit('update:isShow', false);
};

const handleConfirm = async () => {
  if (!token) {
    return;
  }
  const expiresIn = resolveExpiresIn();
  if (expiresIn === null) {
    messageError('续期后截止时间必须晚于当前时间');
    return;
  }
  submitting.value = true;
  try {
    await renewPersonalToken(token.id, { expires_in: expiresIn }, realm);
    messageSuccess('续期成功');
    emit('success');
    emit('update:isShow', false);
  }
  catch {
    // 全局响应中间件已弹出错误信息
  }
  finally {
    submitting.value = false;
  }
};
</script>

<style scoped lang="scss">
.renew {
  &__field {
    margin-bottom: 16px;

    label {
      display: block;
      margin-bottom: 6px;
      font-size: 12px;
      color: #63656e;
    }
  }

  &__value {
    color: #313238;
  }

  &__preview {
    padding: 12px 14px;
    font-size: 12px;
    line-height: 22px;
    background: #f5f7fa;
    border-radius: 2px;

    &-label {
      display: inline-block;
      width: 110px;
      color: #979ba5;
    }

    &-new {
      color: #ff9c01;
    }
  }
}
</style>
