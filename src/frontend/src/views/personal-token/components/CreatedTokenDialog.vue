<template>
  <BkDialog
    :is-show="isShow"
    :width="480"
    :quick-close="false"
    show-mask
    header-align="center"
    dialog-type="show"
    @closed="handleClose"
  >
    <div class="created-token">
      <div class="created-token__icon">
        <AgIcon
          name="check-circle-shape"
          size="42"
          color="#2dcb56"
        />
      </div>
      <h3 class="created-token__title">
        令牌创建成功
      </h3>
      <BkAlert
        class="created-token__warn"
        theme="warning"
        title="请立即复制并妥善保存，令牌明文仅展示这一次，关闭后无法再次查看。"
      />

      <div class="created-token__field">
        <label>令牌名称</label>
        <div class="created-token__value">
          {{ token?.name || '--' }}
        </div>
      </div>

      <div class="created-token__field">
        <label>令牌（access_token）</label>
        <div class="created-token__token">
          <span class="created-token__token-text">{{ token?.token }}</span>
          <BkButton
            text
            theme="primary"
            @click="handleCopy"
          >
            <AgIcon
              name="copy"
              size="14"
              class="mr-2px"
            />
            复制
          </BkButton>
        </div>
      </div>
    </div>

    <template #footer>
      <BkButton
        theme="primary"
        @click="handleClose"
      >
        我已保存，关闭
      </BkButton>
    </template>
  </BkDialog>
</template>

<script setup lang="ts">
import type { CreatedPersonalToken } from '@/services/source/personalToken';
import { messageError, messageSuccess } from '@/utils';

const { token } = defineProps<{
  isShow: boolean
  token: CreatedPersonalToken | null
}>();

const emit = defineEmits<{ 'update:isShow': [value: boolean] }>();

const handleCopy = async () => {
  const value = token?.token;
  if (!value) {
    return;
  }
  try {
    await navigator.clipboard.writeText(value);
    messageSuccess('已复制到剪贴板');
  }
  catch {
    messageError('复制失败，请手动选择并复制');
  }
};

const handleClose = () => {
  emit('update:isShow', false);
};
</script>

<style scoped lang="scss">
.created-token {
  padding: 8px 8px 0;
  text-align: center;

  &__icon {
    margin-bottom: 12px;
  }

  &__title {
    margin-bottom: 16px;
    font-size: 20px;
    color: #313238;
  }

  &__warn {
    margin-bottom: 16px;
    text-align: left;
  }

  &__field {
    margin-bottom: 14px;
    text-align: left;

    label {
      display: block;
      margin-bottom: 4px;
      font-size: 12px;
      color: #979ba5;
    }
  }

  &__value {
    color: #313238;
  }

  &__token {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 10px;
    background: #f5f7fa;
    border-radius: 2px;
  }

  &__token-text {
    margin-right: 8px;
    font-family: monospace;
    color: #313238;
    word-break: break-all;
  }
}
</style>
