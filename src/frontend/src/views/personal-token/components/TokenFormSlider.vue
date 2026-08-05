<template>
  <BkSideslider
    :is-show="isShow"
    :title="isEdit ? '编辑个人令牌' : '新建个人令牌'"
    :width="640"
    quick-close
    @update:is-show="handleClose"
  >
    <template #default>
      <div class="token-form">
        <BkForm
          :model="formData"
          :label-width="90"
        >
          <BkFormItem
            label="名称"
            required
            property="name"
          >
            <BkInput
              v-model="formData.name"
              :maxlength="64"
              placeholder="请输入令牌名称"
            />
          </BkFormItem>

          <BkFormItem
            label="描述"
            property="description"
          >
            <BkInput
              v-model="formData.description"
              type="textarea"
              :maxlength="255"
              :rows="3"
              placeholder="请输入描述（选填）"
            />
          </BkFormItem>

          <BkFormItem
            v-if="!isEdit"
            label="过期时间"
            required
            property="expiry"
          >
            <BkRadioGroup
              v-model="expiryMode"
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
              v-if="expiryMode === 'custom'"
              v-model="customExpiry"
              class="mt-8px w-full"
              type="datetime"
              :clearable="false"
              placeholder="请选择过期时间"
            />
            <p class="token-form__hint">
              到期后令牌将失效，可在到期前续期。最长有效期由管理员配置。
            </p>
          </BkFormItem>

          <BkFormItem
            label="授权资源"
            required
            property="resource"
          >
            <ResourceSelector
              v-model="formData.resource"
              :realm="realm"
            />
          </BkFormItem>
        </BkForm>
      </div>
    </template>
    <template #footer>
      <div class="token-form__footer">
        <BkButton
          theme="primary"
          :loading="submitting"
          @click="handleSubmit"
        >
          {{ isEdit ? '保存' : '生成令牌' }}
        </BkButton>
        <BkButton
          class="ml-8px"
          :disabled="submitting"
          @click="handleClose(false)"
        >
          取消
        </BkButton>
      </div>
    </template>
  </BkSideslider>
</template>

<script setup lang="ts">
import dayjs from 'dayjs';
import ResourceSelector from './ResourceSelector.vue';
import {
  type CreatedPersonalToken,
  PERSONAL_TOKEN_REALM,
  type PersonalToken,
  createPersonalToken,
  updatePersonalToken,
} from '@/services/source/personalToken';
import { DEFAULT_TTL_DAYS, DEFAULT_TTL_SECONDS, RENEW_PRESETS } from '../utils';
import { messageError } from '@/utils';

const { isShow, token = null, realm = PERSONAL_TOKEN_REALM } = defineProps<{
  isShow: boolean
  /** 有值表示编辑模式 */
  token?: PersonalToken | null
  /** 所属 realm，来自路由 */
  realm?: string
}>();

const emit = defineEmits<{
  'update:isShow': [value: boolean]
  'created': [token: CreatedPersonalToken]
  'updated': []
}>();

const isEdit = computed(() => !!token);

const formData = reactive<{
  name: string
  description: string
  resource: string[]
}>({
  name: '',
  description: '',
  resource: [],
});

// 默认预选 90 天
const expiryMode = ref<string>(String(DEFAULT_TTL_SECONDS));
const customExpiry = ref<Date>(dayjs().add(DEFAULT_TTL_DAYS, 'day').toDate());
const submitting = ref(false);

const resetForm = () => {
  if (token) {
    formData.name = token.name;
    formData.description = token.description;
    formData.resource = [...token.audience];
  }
  else {
    formData.name = '';
    formData.description = '';
    formData.resource = [];
    expiryMode.value = String(DEFAULT_TTL_SECONDS);
    customExpiry.value = dayjs().add(DEFAULT_TTL_DAYS, 'day').toDate();
  }
};

watch(
  () => isShow,
  (show) => {
    if (show) {
      resetForm();
    }
  },
);

const resolveExpiresIn = (): number | null => {
  if (expiryMode.value === 'custom') {
    const seconds = dayjs(customExpiry.value).diff(dayjs(), 'second');
    return seconds > 0 ? seconds : null;
  }
  return Number(expiryMode.value);
};

const validate = (): boolean => {
  if (!formData.name.trim()) {
    messageError('请输入令牌名称');
    return false;
  }
  if (!formData.resource.length) {
    messageError('请至少选择一个授权资源');
    return false;
  }
  if (!isEdit.value && resolveExpiresIn() === null) {
    messageError('过期时间必须晚于当前时间');
    return false;
  }
  return true;
};

const handleClose = (value: boolean) => {
  if (submitting.value) {
    return;
  }
  emit('update:isShow', value);
};

const handleSubmit = async () => {
  if (!validate()) {
    return;
  }
  submitting.value = true;
  try {
    const resource = formData.resource.join(',');
    if (isEdit.value && token) {
      await updatePersonalToken(token.id, {
        name: formData.name.trim(),
        description: formData.description,
        resource,
      }, realm);
      emit('updated');
    }
    else {
      const created = await createPersonalToken({
        name: formData.name.trim(),
        description: formData.description,
        resource,
        expires_in: resolveExpiresIn() ?? 0,
      }, realm);
      emit('created', created);
    }
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
.token-form {
  padding: 20px 24px;

  &__hint {
    margin-top: 6px;
    font-size: 12px;
    color: #979ba5;
  }

  &__footer {
    padding-left: 24px;
  }
}
</style>
