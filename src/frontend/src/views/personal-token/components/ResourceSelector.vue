<template>
  <div class="resource-selector">
    <BkSelect
      :model-value="modelValue"
      multiple
      multiple-mode="tag"
      filterable
      collapse-tags
      placeholder="请选择授权资源"
      @update:model-value="handleChange"
    >
      <BkOptionGroup
        v-for="group in groups"
        :key="group.type"
        :label="group.display_name"
        collapsible
      >
        <BkOption
          v-for="item in group.items"
          :id="`${group.type}:${item.name}`"
          :key="`${group.type}:${item.name}`"
          :name="item.display_name || item.name"
        />
      </BkOptionGroup>
    </BkSelect>
    <p
      v-if="!hasOptions"
      class="resource-selector__empty"
    >
      当前 Realm 暂无可选授权资源，请联系管理员在网关侧配置。
    </p>
  </div>
</template>

<script setup lang="ts">
import { PERSONAL_TOKEN_REALM, getSelectableResources } from '@/services/source/personalToken';

const { modelValue, realm = PERSONAL_TOKEN_REALM } = defineProps<{
  /** 已选资源 token 列表，形如 ["service:codecc"] */
  modelValue: string[]
  /** 所属 realm，来自路由 */
  realm?: string
}>();

const emit = defineEmits<{ 'update:modelValue': [value: string[]] }>();

// Demo 阶段资源目录在前端固定（见 services/source/personalToken）。
const groups = computed(() => getSelectableResources(realm));

const hasOptions = computed(() =>
  groups.value.some(group => (group.items?.length ?? 0) > 0));

const handleChange = (value: string[]) => {
  emit('update:modelValue', value);
};
</script>

<style scoped lang="scss">
.resource-selector {
  &__empty {
    margin-top: 6px;
    font-size: 12px;
    color: #ea3636;
  }
}
</style>
