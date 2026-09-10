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
                class="resource-tag"
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
            <div class="collapse-toggle">
              <CommonIcon
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
      class="resource-tag"
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
  border: 1px solid #dcdee5;
  border-radius: 8px;
}

.collapse-panel-header {
  display: flex;
  min-height: 36px;
  padding: 8px 12px;
  cursor: pointer;
  align-items: center;
  gap: 8px;

  .panel-title {
    display: flex;
    min-width: 0;
    flex: 1;
    align-items: center;
    gap: 8px;
  }

  .active-icon {
    transform: rotateZ(-180deg);
    transition: all 0.5s;
  }
}

.collapse-toggle {
  display: flex;
  width: 18px;
  height: 18px;
  margin-left: auto;
  background: #f5f7fa;
  border-radius: 50%;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
}

.static-resource-item {
  display: flex;
  min-height: 36px;
  padding: 8px 12px;
  color: #313238;
  background-color: #fff;
  border: 1px solid #dcdee5;
  border-radius: 8px;
  align-items: center;
  gap: 8px;
}

.resource-name {
  min-width: 0;
  font-size: 12px;
  line-height: 20px;
  overflow-wrap: anywhere;
}

.resource-tag,
.counter {
  flex-shrink: 0;
  white-space: nowrap;
}

.counter {
  display: flex;
  height: 16px;
  padding: 0 6px;
  font-size: 10px;
  line-height: 16px;
  color: #4d4f56;
  background: #f0f1f5;
  border-radius: 8px;
  align-items: center;
}

:deep(.bk-collapse-item) {
  margin-bottom: 0;
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
  background: #fafbfd;
  border-top: 1px solid #dcdee5;

  .api-sub-list {
    max-height: 100px;
    padding: 4px 12px 0 20px;
    margin: 0;
    overflow-y: auto;
    list-style: none;

    .api-sub-item {
      display: flex;
      padding: 6px 0;
      font-size: 12px;
      line-height: 20px;
      color: #4d4f56;
      align-items: flex-start;
      gap: 4px;
      overflow-wrap: anywhere;

      .dot {
        display: inline-block;
        width: 6px;
        height: 6px;
        margin-top: 7px;
        border-radius: 50%;
        flex-shrink: 0;
      }

      .dot-green {
        background: #a1e3ba;
      }
    }
  }
}
</style>
