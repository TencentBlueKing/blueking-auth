<template>
  <div class="code-page">
    <div class="code-card">
      <!-- 标题 -->
      <h2 class="code-title">
        验证您的设备
      </h2>

      <!-- 用户信息 -->
      <div class="code-user">
        <AgIcon
          name="user-circle"
          size="16"
          color="#3A84FF"
          class="mr-4px"
        />
        <span>已登陆为 <strong>{{ userInfoStore.info?.username || '--' }}</strong></span>
      </div>

      <!-- 描述文字 -->
      <p class="code-desc">
        请输入应用或设备上显示的设备码<br>
        切勿使用他人发送给您的验证码
      </p>

      <!-- 验证码输入区域 -->
      <div class="code-inputs">
        <!-- 前半段 -->
        <div class="code-group">
          <input
            v-for="i in FIRST_HALF"
            :key="'first-' + i"
            :ref="(el) => setInputRef(el, i - 1)"
            :value="codes[i - 1]"
            type="text"
            maxlength="1"
            class="code-input"
            :class="{ 'code-input--error': hasError }"
            @input="handleInput(i - 1, $event)"
            @keydown="handleKeydown(i - 1, $event)"
            @paste="handlePaste"
          >
        </div>

        <!-- 分隔符 -->
        <span class="code-separator">-</span>

        <!-- 后半段 -->
        <div class="code-group">
          <input
            v-for="i in (CODE_LENGTH - FIRST_HALF)"
            :key="'second-' + i"
            :ref="(el) => setInputRef(el, FIRST_HALF + i - 1)"
            :value="codes[FIRST_HALF + i - 1]"
            type="text"
            inputmode="numeric"
            maxlength="1"
            class="code-input"
            :class="{ 'code-input--error': hasError }"
            @input="handleInput(FIRST_HALF + i - 1, $event)"
            @keydown="handleKeydown(FIRST_HALF + i - 1, $event)"
            @paste="handlePaste"
          >
        </div>
      </div>

      <!-- 错误提示 -->
      <div
        v-if="hasError"
        class="code-error"
      >
        <AgIcon
          name="exclamation-circle-fill"
          size="14"
          color="#EA3636"
          class="mr-4px"
        />
        验证码错误，请重新输入
      </div>

      <!-- 提交按钮 -->
      <div class="code-actions">
        <BkButton
          class="code-submit-btn"
          :theme="btnTheme"
          :loading="loading"
          :disabled="fullCode.length < CODE_LENGTH"
          @click="handleSubmit"
        >
          继续
        </BkButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">

import { useUserInfo } from '@/stores';
import { verifyDeviceCode } from '@/services/source/oauth2/device.ts';
import { useDevice } from '@/stores/useDevice.ts';

const router = useRouter();
const userInfoStore = useUserInfo();
const deviceStore = useDevice();

/** 验证码总位数 */
const CODE_LENGTH = 8;
/** 前半段位数 */
const FIRST_HALF = 4;

/** 每个格子的值 */
const codes = ref<string[]>(Array.from({ length: CODE_LENGTH }, () => ''));
/** 输入框 ref 数组 */
const inputRefs = ref<HTMLInputElement[]>([]);
/** 是否显示错误提示 */
const hasError = ref(false);
/** 按钮加载状态 */
const loading = ref(false);

/** 完整验证码 */
const fullCode = computed(() => codes.value.join(''));

const btnTheme = computed(() => {
  if (hasError.value) {
    return 'danger';
  }
  return 'primary';
});

/**
 * 设置 ref 回调，用于收集每个 input 的 DOM 引用
 */
function setInputRef(el: any, idx: number) {
  if (el) {
    inputRefs.value[idx] = el as HTMLInputElement;
  }
}

/**
 * 处理输入事件：只允许数字，输入后自动跳到下一格
 */
function handleInput(idx: number, e: Event) {
  const input = e.target as HTMLInputElement;
  // 只保留最后一位数字
  // const val = input.value.replace(/\D/g, '').slice(-1);
  const val = input.value;
  codes.value[idx] = val;
  hasError.value = false;

  if (val && idx < CODE_LENGTH - 1) {
    inputRefs.value[idx + 1]?.focus();
  }

  if (fullCode.value.length === CODE_LENGTH) {
    verify();
  }
}

/**
 * 处理键盘事件：Backspace 删除当前格并回退到上一格
 */
function handleKeydown(idx: number, e: KeyboardEvent) {
  if (e.key === 'Backspace') {
    if (!codes.value[idx] && idx > 0) {
      codes.value[idx - 1] = '';
      inputRefs.value[idx - 1]?.focus();
      e.preventDefault();
    }
  }
}

/**
 * 处理粘贴事件：支持一次性粘贴完整验证码
 */
function handlePaste(e: ClipboardEvent) {
  e.preventDefault();
  const paste = (e.clipboardData?.getData('text') ?? '').replace('-', '').slice(0, CODE_LENGTH);
  if (!paste) return;

  for (let i = 0; i < CODE_LENGTH; i++) {
    codes.value[i] = paste[i] ?? '';
  }
  // 聚焦到最后一个已填的格子或末尾
  const focusIdx = Math.min(paste.length, CODE_LENGTH - 1);
  inputRefs.value[focusIdx]?.focus();
  hasError.value = false;

  if (fullCode.value.length === CODE_LENGTH) {
    verify();
  }
}

async function verify() {
  try {
    const info = await verifyDeviceCode({ user_code: fullCode.value });
    deviceStore.setConsentInfo(info);
    deviceStore.setCode(fullCode.value);
    hasError.value = false;
  }
  catch {
    hasError.value = true;
  }
}

/**
 * 点击"继续"按钮
 */
async function handleSubmit() {
  if (fullCode.value.length < CODE_LENGTH) return;

  try {
    loading.value = true;
    await verify();
    router.replace({
      name: 'Authorize',
      query: { source: 'device' },
    });
  }
  catch {
    hasError.value = true;
  }
  finally {
    loading.value = false;
  }
}
</script>

<style scoped lang="scss">
.code-page {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 100%;
  height: calc(100vh - 48px);
  box-sizing: border-box;
}

.code-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 490px;
  padding: 24px 32px;
  background: #fff;
  border-radius: 16px;
  box-shadow: 0 2px 12px 0 rgb(0 0 0 / 6%);
  box-sizing: border-box;
}

.code-title {
  margin: 0 0 16px;
  font-size: 20px;
  font-weight: 600;
  color: #313238;
}

.code-user {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
  font-size: 14px;
  color: #313238;

  strong {
    font-weight: 600;
  }
}

.code-desc {
  margin: 0 0 24px;
  font-size: 13px;
  line-height: 1.6;
  color: #979ba5;
  text-align: center;
}

/* 验证码输入区域 */

.code-inputs {
  display: flex;
  align-items: center;
   gap: 6px;

  // justify-content: space-between;
  margin-bottom: 8px;
}

.code-group {
  display: flex;
  gap: 6px;
}

.code-input {
  width: 46px;
  height: 56px;
  font-size: 24px;
  font-weight: 600;
  color: #313238;
  text-align: center;
  background: #fff;
  border: 1px solid #c4c6cc;
  border-radius: 4px;
  outline: none;
  transition: border-color 0.2s;
  caret-color: #3a84ff;

  &:focus {
    border-color: #3a84ff;
    box-shadow: 0 0 0 2px rgb(58 132 255 / 15%);
  }

  &--error {
    background: #fff1f1;
    border-color: #ea3636;

    &:focus {
      border-color: #ea3636;
      box-shadow: 0 0 0 2px rgb(234 54 54 / 15%);
    }
  }
}

.code-separator {
  margin: 0 2px;
  font-size: 20px;
  font-weight: 500;
  color: #979ba5;
  user-select: none;
}

/* 错误提示 */

.code-error {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
  font-size: 12px;
  color: #ea3636;
}

/* 提交按钮 */

.code-actions {
  width: 100%;
  margin-top: 16px;

  .code-submit-btn {
    width: 100%;
    height: 40px;
    font-size: 14px;
  }
}
</style>
