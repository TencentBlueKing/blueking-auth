<template>
  <div
    v-if="collapsible"
    class="wrapper"
  >
    <BkCollapse v-model="activeIndex">
      <BkCollapsePanel :name="name">
        <template #header>
          <div class="collapse-panel-header">
            <slot
              v-if="slots.header"
              name="header"
            />
            <div
              v-else
              class="panel-title"
            >
              <BkTag
                v-if="tag"
                :theme="theme"
                type="stroke"
                size="small"
              >
                {{ tag }}
              </BkTag>
              <span class="resource-name">{{ title }}</span>
              <div
                v-if="counter"
                class="counter"
              >
                {{ counter }}
              </div>
            </div>
            <div class="ml-auto h-18px w-18px rounded-full bg-[#F5F7FA] flex items-center justify-center">
              <AgIcon
                size="14"
                color="#C4C6CC"
                :class="{ 'active-icon': isPanelActive }"
                name="down-small"
              />
            </div>
          </div>
        </template>
        <template #content>
          <div class="content-wrapper">
            <slot name="default">
              <ul class="api-sub-list">
                <li
                  v-for="api in items"
                  :key="api.name"
                  class="api-sub-item"
                >
                  <span class="dot dot-green" />
                  <span>{{ api.display_name }}</span>
                </li>
              </ul>
            </slot>
          </div>
        </template>
      </BkCollapsePanel>
    </BkCollapse>
  </div>
  <div
    v-else
    class="static-resource-item"
  >
    <BkTag
      :theme="theme"
      type="stroke"
      size="small"
    >
      {{ tag }}
    </BkTag>
    <span class="resource-name">{{ title }}</span>
    <div
      v-if="counter"
      class="counter"
    >
      {{ counter }}
    </div>
  </div>
</template>

<script setup lang="ts">

import type { ResourceItem } from '@/services/source/oauth2/consent.ts';

interface Props {
  collapsible?: boolean
  title?: string
  name?: string
  tag?: string
  counter?: string
  theme?: string
  items?: ResourceItem[]
}

interface Emits { (e: 'toggle', value: boolean): void }

interface Slots {
  default: any
  header: any
}

interface Exposes {
  show: () => void
  hide: () => void
}

const {
  collapsible = false,
  title = '',
  name = 'default',
  tag = '',
  theme = 'success',
  counter = '',
  items = [],
} = defineProps<Props>();

const emits = defineEmits<Emits>();

const slots = defineSlots<Slots>();

const activeIndex = ref<string[]>([]);

const isPanelActive = computed(() => !activeIndex.value.includes(name));

watch(isPanelActive, () => {
  emits('toggle', isPanelActive.value);
});

defineExpose<Exposes>({
  show: () => {
    activeIndex.value = [name];
  },
  hide: () => {
    activeIndex.value = [];
  },
});
</script>

<style lang="scss" scoped>

.wrapper {
  overflow: hidden;
  background-color: #fff;
  border: 1px solid #DCDEE5;
  border-radius: 8px;

  :deep(.bk-collapse-item) {
    margin-bottom: 0;
  }
}

.collapse-panel-header {
  position: relative;
  display: flex;
  height: 36px;
  margin-right: 12px;
  cursor: pointer;
  align-items: center;

  :deep(.iamcenter-down-shape) {
    color: #313238;
    transform: rotateZ(0deg);
    transition: all 0.5s;
  }

  .panel-title {
    display: flex;
    padding-left: 12px;
    align-items: center;
    gap: 8px;

    .resource-name {
      font-size: 12px;
    }

    .counter {
      display: flex;
      height: 16px;
      padding: 0 6px;
      font-size: 10px;
      color:#4D4F56;
      background:  #F0F1F5;
      border-radius: 8px;
      align-items: center;
      align-content: center;
      gap: 0 4px;
      flex-wrap: wrap;
    }
  }

  .active-icon {
    transform: rotateZ(-180deg);
    transition: all 0.5s;
  }
}

:deep(.bk-collapse-item) {
  margin-bottom: 8px;
  border: none;

  .bk-collapse-header {
    min-height: auto;
    padding: 0;
    line-height: normal;
    background: transparent;
    border: none;
  }

  .bk-collapse-content {
    padding: 0;
    background: transparent;
    border: none;
  }
}

.content-wrapper {
  background: #FAFBFD;
  border-top: 1px solid #DCDEE5;

  .api-sub-list {
    max-height: 100px;
    padding: 4px 0 0 20px;
    margin: 0;
    overflow-y: auto;
    list-style: none;

    .api-sub-item {
      display: flex;
      padding: 6px 0;
      font-size: 12px;
      line-height: 20px;
      color: #4D4F56;
      align-items: center;
      gap: 4px;

      .dot {
        display: inline-block;
        width: 6px;
        height: 6px;
        border-radius: 50%;
      }

      .dot-blue {
        background: #3a84ff;
      }

      .dot-green {
        background: #A1E3BA;
      }
    }
  }
}

.static-resource-item {
  display: flex;
  height: 36px;
  padding: 9px 12px;
  color: #313238;
  background-color: #fff;
  border: 1px solid #DCDEE5;
  border-radius: 8px;
  align-items: center;
  gap: 8px;

  .resource-name {
    font-size: 12px;
  }

  .counter {
    display: flex;
    height: 16px;
    padding: 0 6px;
    font-size: 10px;
    color:#4D4F56;
    background:  #F0F1F5;
    border-radius: 8px;
    align-items: center;
    align-content: center;
    gap: 0 4px;
    flex-wrap: wrap;
  }
}
</style>
