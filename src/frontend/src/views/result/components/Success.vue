<template>
  <div class="success-page">
    <div class="success-card">
      <!-- 成功图标 -->
      <div class="success-icon">
        <svg
          viewBox="0 0 32 32"
          width="32"
          height="32"
        >
          <circle
            cx="16"
            cy="16"
            r="16"
            fill="#E5F6ED"
          />
          <path
            d="M10 16.5L14 20.5L22 12.5"
            stroke="#2DCB56"
            stroke-width="2.5"
            stroke-linecap="round"
            stroke-linejoin="round"
            fill="none"
          />
        </svg>
      </div>

      <!-- 标题 -->
      <h2 class="success-title">
        授权成功
      </h2>

      <!-- 描述 -->
      <p class="success-desc">
        已授权成功，您现在可关闭此窗口或返回应用。
      </p>

      <!-- 按钮区域 -->
      <div class="success-actions">
        <!-- <BkButton
          v-if="url"
          theme="primary"
          @click="handleBack"
        >
          返回应用
        </BkButton> -->
        <BkButton @click="handleClose">
          关闭窗口
        </BkButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">

const { url = '' } = defineProps<{ url?: string }>();

const handleBack = () => {
  window.location.replace(url);
};

const handleClose = () => {
  window.close();
};
</script>

<style scoped lang="scss">
.success-page {
  display: flex;
  justify-content: center;
  align-items: flex-start;
  width: 100%;
  padding: 40px 0;
  box-sizing: border-box;
}

.success-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: clamp(460px, 25vw, 600px);
  padding: 24px 32px;
  background: #fff;
  border-radius: 16px;
  box-shadow: 0 2px 6px 0 rgb(0 0 0 / 5%);
}

.success-icon {
  margin-bottom: 16px;
}

.success-title {
  margin: 0 0 24px;
  font-size: 20px;
  font-weight: 600;
  color: #313238;
}

.success-desc {
  width: 100%;
  padding: 12px 16px;
  margin: 0 0 32px;
  font-size: 14px;
  color: #4D4F56;
  background-color: #F5F7FA;
}

.success-actions {
  display: flex;
  gap: 8px;
}
</style>
