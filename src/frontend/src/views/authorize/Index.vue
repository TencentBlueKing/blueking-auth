<template>
  <div class="auth-page">
    <div
      v-if="hasError"
      class="auth-card flex items-center justify-center"
    >
      <svg
        viewBox="0 0 32 32"
        width="32"
        height="32"
      >
        <circle
          cx="16"
          cy="16"
          r="16"
          fill="#FEDDDC"
        />
        <path
          d="M11 11L21 21M21 11L11 21"
          stroke="#EA3636"
          stroke-width="2.5"
          stroke-linecap="round"
        />
      </svg>
      <div class="mt-16px">
        出错了
      </div>
    </div>
    <div
      v-else
      class="auth-card"
    >
      <!-- 顶部 Logo 区域 -->
      <div class="auth-header">
        <div class="logo-section">
          <div class="subject-info">
            <div class="logo-circle">
              <img
                :src="consentInfo?.client_logo_uri"
                :alt="consentInfo?.client_name || '--'"
                class="logo-img"
              >
            </div>
            <div class="subject-name">
              {{ consentInfo?.client_name || '--' }}
              <span
                v-if="consentInfo?.client_type === 'public'"
                class="client-type-tag"
              >（公开客户端）</span>
            </div>
          </div>
          <AgIcon
            class="rotate-180"
            name="return-small"
            color="#3A84FF"
            size="48"
          />
          <div class="subject-info">
            <div class="logo-circle">
              <img
                :src="logoImageMap[consentInfo?.realm_name as keyof typeof logoImageMap]"
                :alt="consentInfo?.realm_name || '--'"
                class="logo-img"
              >
            </div>
            <div class="subject-name">
              {{ realmNameMap[consentInfo?.realm_name as keyof typeof realmNameMap] || '--' }}
            </div>
          </div>
        </div>
      </div>

      <!-- 标题 -->
      <h2 class="auth-title">
        应用授权确认
      </h2>

      <!-- 描述 -->
      <p class="auth-desc">
        授权 <span class="highlight">{{ consentInfo?.client_name || '--' }}</span> 访问或操作您在
        <span class="highlight">{{ realmNameMap[consentInfo?.realm_name as keyof typeof realmNameMap] || '--' }}</span> 上的资源
      </p>

      <!-- 警告 -->
      <BkAlert
        class="mb-24px"
        theme="warning"
        title="请确保您信任该设备，因为它将获得您账户的访问权限。"
      />

      <!-- 信息区域 -->
      <div class="auth-info">
        <div class="info-row">
          <span class="info-label">当前用户</span>
          <span class="info-value">
            <AgIcon
              name="user-circle"
              size="16"
              color="#979BA5"
              class="mr-4px"
            />
            {{ userInfoStore.info?.username || '--' }}
          </span>
        </div>

        <div class="info-row">
          <span class="info-label">授权对象</span>
          <span class="info-value">{{ consentInfo?.client_name || '--' }}</span>
        </div>

        <!-- 分割线 -->
        <div class="h-1px my-16px mr-24px bg-[#DCDEE5]" />

        <div class="info-row info-row-resource">
          <span class="info-label">授权资源</span>
          <div
            v-if="consentInfo?.resources?.length"
            class="info-value resource-section"
          >
            <div
              v-for="(resource, index) in consentInfo?.resources"
              :key="index"
              class="resource-group"
            >
              <div class="resource-group-title">
                <span class="dot dot-blue" />
                <strong>{{ resource.display_name }}</strong>
              </div>
              <div class="resource-list">
                <ResourceCollapse
                  v-for="resourceItem in resource.items"
                  :key="resourceItem.name"
                  theme="info"
                  :title="resourceItem.display_name"
                  :tag="resource.type"
                  :items="resourceItem.items"
                  :counter="getResourceCollapseCounter(resourceItem.items)"
                  :collapsible="isResourceCollapsible(resourceItem.items)"
                />
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 底部按钮 -->
      <div class="auth-actions">
        <BkButton
          theme="primary"
          class="action-btn"
          @click="() => handleSubmit('approve')"
        >
          授权
        </BkButton>
        <BkButton
          class="action-btn"
          @click="() => handleSubmit('deny')"
        >
          拒绝
        </BkButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import ResourceCollapse from './components/ResourceCollapse.vue';
import {
  type ConsentResponseData,
  type ResourceItem,
  confirmConsent,
  getConsentInfo,
} from '@/services/source/oauth2/consent.ts';
import { useUserInfo } from '@/stores';
import ApiGatewayLogo from '@/assets/bk_api_gateway_logo.png';
import DevOpsLogo from '@/assets/bk_devops_logo.png';
import { useDevice } from '@/stores/useDevice.ts';
import { cloneDeep } from 'lodash-es';
import { confirmDeviceCode } from '@/services/source/oauth2/device.ts';

const route = useRoute();
const router = useRouter();
const userInfoStore = useUserInfo();
const deviceStore = useDevice();

const consentInfo = ref<ConsentResponseData>();

// 页面来源
const source = ref('');
const consentChallenge = ref('');
const hasError = ref(false);

const logoImageMap = {
  'blueking': ApiGatewayLogo,
  'bk-devops': DevOpsLogo,
};

const realmNameMap = {
  'blueking': '蓝鲸网关 MCP & API',
  'bk-devops': '蓝盾',
};

watch(
  () => route.query,
  async () => {
    hasError.value = false;
    // 从设备验证码授权过来的
    if (route.query?.source === 'device' && deviceStore.consentInfo && deviceStore.code) {
      source.value = 'device';
      consentInfo.value = cloneDeep(deviceStore.consentInfo);
    }
    else if (route.query?.consent_challenge) {
      consentChallenge.value = route.query.consent_challenge as string;
      try {
        consentInfo.value = await getConsentInfo({ consent_challenge: consentChallenge.value });
      }
      catch {
        hasError.value = true;
      }
    }
    else {
      consentChallenge.value = '';
      consentInfo.value = undefined;
      hasError.value = true;
    }
  },
  {
    immediate: true,
    deep: true,
  },
);

const handleSubmit = async (action: string) => {
  if (source.value === 'device') {
    await confirmDeviceCode({
      user_code: deviceStore.code,
      action,
    });
    router.replace({
      name: 'Result',
      query: { action },
    });
  }
  else {
    const { redirect_url } = await confirmConsent({
      consent_challenge: consentChallenge.value,
      action,
    });
    if (redirect_url) {
      router.replace({
        name: 'Result',
        query: {
          action,
          redirect: redirect_url,
        },
      });
    }
  }
};

const getResourceCollapseCounter = (items: ResourceItem['items'] = []) => {
  if (items.length) {
    if (items[0]!.name === '*') {
      return '所有API';
    }
    return `${items.length}个API`;
  }
  return undefined;
};

const isResourceCollapsible = (items: ResourceItem['items'] = []) => {
  if (items?.length) {
    return items[0]!.name === '*';
  }
  return false;
};

</script>

<style lang="scss" scoped>
.auth-page {
  display: flex;
  width: 100%;
  height: calc(100vh - 48px);
  min-height: 905.6px;
  padding: 40px 0;
  box-sizing: border-box;
  justify-content: center;
  align-items: flex-start;
}

.auth-card {
  display: flex;
  width: clamp(516px, 25vw, 700px);
  height: clamp(825.6px, 40vw, 1120px);
  padding: 24px 32px 32px;
  background: #fff;
  border-radius: 16px;
  box-shadow: 0 2px 12px 0 rgb(0 0 0 / 6%);
  box-sizing: border-box;
  flex-direction: column;
}

/* 顶部 Logo */

.auth-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 20px;
}

.logo-section {
  display: flex;
   align-items: flex-start;
  gap: 16px;

  .subject-info {
    display: flex;
    width: 30%;
    flex-direction: column;
    gap: 8px;
    align-items: center;

    .subject-name {
      font-size: 12px;
      color: #979ba5;
      white-space: nowrap;
    }
  }
}

.client-type-tag {
  color: #ff9c01;
}

.logo-circle {
  display: flex;
  width: 48px;
  height: 48px;
  overflow: hidden;
  font-size: 18px;
  font-weight: 600;
  color: #fff;
  background: #3a84ff;
  border-radius: 8px;
  box-shadow: 0 0 6px 0 #0003;
  align-items: center;
  justify-content: center;
}

.logo-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

/* 标题 & 描述 */

.auth-title {
  margin: 0 0 12px;
  font-size: 20px;
  font-weight: 600;
  color: #313238;
  text-align: center;
}

.auth-desc {
  margin: 0 0 24px;
  font-size: 14px;
  line-height: 1.6;
  color: #63656e;
  text-align: center;

  .highlight {
    font-weight: 500;
    color: #3a84ff;
  }
}

/* 信息区域 */

.auth-info {
  max-height: calc(100% - 318.4px);
  padding: 16px 0 24px 24px;
  overflow: hidden;
  background: #f5f7fa;
  border-radius: 10px;
  flex: 1;
}

.info-row {
  display: flex;
  padding-right: 24px;
  margin-bottom: 16px;
  font-size: 14px;
  line-height: 22px;
  align-items: flex-start;
}

.info-label {
  width: 70px;
  margin-right: 16px;
  color: #979ba5;
  flex-shrink: 0;
}

.info-value {
  color: #313238;
}

.info-row-resource {
  height: calc(100% - 91px);
  padding-right: 0;
  margin-bottom: 0;
  align-items: flex-start;
  overflow: hidden;

  .info-label {
    padding-top: 2px;
  }
}

/* 授权资源 */

.resource-section {
  height: 100%;
  padding-right: 24px;
  padding-bottom: 12px;
   overflow-y: auto;
  flex: 1;
}

.resource-group {
  margin-bottom: 16px;
}

.resource-group-title {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 10px;
  font-size: 14px;
  color: #313238;
}

.dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.dot-blue {
  background: #3a84ff;
}

.dot-green {
  background: #2dcb56;
}

.resource-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.collapse-header-item {
  cursor: pointer;
}

/* API 子列表 */

.api-sub-list {
  padding: 4px 0 0 20px;
  margin: 0;
  list-style: none;
}

.api-sub-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  font-size: 14px;
  color: #63656e;
}

/* 底部按钮 */

.auth-actions {
  display: flex;
  padding-top: 16px;
  margin-top: 24px;
  gap: 12px;

  .action-btn {
    flex: 1;
    height: 40px;
    font-size: 14px;
  }
}
</style>
