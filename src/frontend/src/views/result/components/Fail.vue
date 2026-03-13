<script setup lang="ts">
const handleClose = () => {
  window.close();
};
</script>

<template>
  <div class="fail-page">
    <div class="fail-card">
      <!-- 失败图标 -->
      <div class="fail-icon">
        <svg viewBox="0 0 32 32" width="32" height="32">
          <circle cx="16" cy="16" r="16" fill="#FEDDDC" />
          <path
            d="M11 11L21 21M21 11L11 21"
            stroke="#EA3636"
            stroke-width="2.5"
            stroke-linecap="round"
          />
        </svg>
      </div>

      <!-- 标题 -->
      <h2 class="fail-title">授权已拒绝</h2>

      <!-- 描述 -->
      <p class="fail-desc">提示：您已选择拒绝此次授权请求</p>

      <!-- 按钮区域 -->
      <div class="fail-actions">
        <BkButton @click="handleClose">关闭窗口</BkButton>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.fail-page {
  display: flex;
  justify-content: center;
  align-items: flex-start;
  width: 100%;
  padding: 40px 0;
  box-sizing: border-box;
}

.fail-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: clamp(460px, 25vw, 600px);
  padding: 24px 32px;
  background: #fff;
  border-radius: 16px;
  box-shadow: 0 2px 6px 0 rgb(0 0 0 / 5%);
}

.fail-icon {
  margin-bottom: 16px;
}

.fail-title {
  margin: 0 0 24px;
  font-size: 20px;
  font-weight: 600;
  color: #313238;
}

.fail-desc {
  width: 100%;
  padding: 12px 16px;
  margin: 0 0 32px;
  font-size: 14px;
  color: #4d4f56;
  background-color: #f5f7fa;
}

.fail-actions {
  display: flex;
  gap: 8px;
}
</style>
