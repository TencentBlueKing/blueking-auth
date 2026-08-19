<template>
  <div
    class="checkbox-collapse"
    :class="{ 'is-compact': compact }"
  >
    <div
      class="header"
      :class="{ 'is-disabled': disabled }"
    >
      <div
        v-if="arrowPosition === 'left'"
        class="toggle"
        :class="{ 'is-collapsed': collapsed }"
        @click="handleTitleClicked"
      >
        <CommonIcon
          name="down-shape"
          color="#979BA5"
          size="10"
        />
      </div>
      <div
        v-if="showCheckbox"
        v-bk-tooltips="{
          content: disabledTips || '',
          disabled: !(disabled || checkboxDisabled),
        }"
        class="prefix"
        @click.stop
      >
        <BkCheckbox
          v-model="enabled"
          :disabled="disabled || checkboxDisabled"
          @change="handleCheckboxChanged"
        />
      </div>
      <div
        class="title"
        @click="handleTitleClicked"
      >
        <slot
          name="title"
          :collapsed="collapsed"
          :enabled="enabled"
        >
          <div class="name">
            {{ name }}
          </div>
          <div class="desc">
            <BkOverflowTitle
              resizeable
              type="tips"
            >
              {{ desc }}
            </BkOverflowTitle>
          </div>
        </slot>
      </div>
      <div
        v-if="arrowPosition === 'right'"
        class="toggle is-right"
        :class="{ 'is-collapsed': collapsed }"
        @click="handleTitleClicked"
      >
        <CommonIcon
          name="down-shape"
          color="#979BA5"
          size="10"
        />
      </div>
    </div>
    <div
      v-show="!collapsed"
      class="content"
    >
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
type ArrowPosition = 'left' | 'right';

interface IProps {
  name?: string
  desc?: string
  disabled?: boolean
  disabledTips?: string
  showCheckbox?: boolean
  checkboxDisabled?: boolean
  autoExpandOnCheck?: boolean
  arrowPosition?: ArrowPosition
  compact?: boolean
}

const enabled = defineModel<boolean>({ default: false });

const collapsed = defineModel<boolean>('collapsed', { default: true });

const {
  name = '',
  desc = '',
  disabled = false,
  disabledTips = undefined,
  showCheckbox = true,
  checkboxDisabled = false,
  autoExpandOnCheck = true,
  arrowPosition = 'right',
  compact = false,
} = defineProps<IProps>();

watch(() => disabled, () => {
  if (disabled) {
    collapsed.value = true;
  }
});

const handleCheckboxChanged = (checked: boolean) => {
  if (autoExpandOnCheck) {
    collapsed.value = !checked;
  }
};

const handleTitleClicked = () => {
  if (disabled) {
    collapsed.value = true;
    return;
  }
  collapsed.value = !collapsed.value;
};

</script>

<style scoped lang="scss">
.checkbox-collapse {
  width: 100%;

  .header {
    display: flex;
    align-items: center;
    height: 40px;
    padding-left: 8px;
    background: #FAFBFD;
    border: 1px solid #DCDEE5;
    border-radius: 2px;

    &.is-disabled {

      .title,
      .toggle {
        cursor: not-allowed;
      }
    }

    .prefix {
      display: flex;
      align-items: center;
      justify-content: center;
      padding-right: 8px;
    }

    .title {
      display: flex;
      align-items: center;
      flex: 1;
      min-width: 0;
      cursor: pointer;

      .name {
        margin-right: 12px;
        font-size: 14px;
        font-weight: 700;
        line-height: 20px;
        color: #4D4F56;
        flex-shrink: 0;
      }

      .desc {
        width: calc(100% - 68px);
        font-size: 12px;
        line-height: 20px;
        color: #4D4F56;
      }
    }

    .toggle {
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 8px;
      cursor: pointer;
      transition: transform 0.2s;

      &.is-right {
        margin-left: auto;
      }

      &.is-collapsed {
        transform: rotate(-90deg);
      }
    }
  }

  &.is-compact {

    .header {
      height: 32px;
      padding-left: 8px;
      background-color: #fff;
      border: 0;
      border-bottom: 1px solid #DCDEE5;
      border-radius: 0;

      .prefix {
        padding: 0 8px 0 2px;
      }

      .title {

        .name {
          margin-right: 8px;
          font-size: 12px;
          font-weight: 400;
        }

        .desc {
          font-size: 12px;
          color: #979BA5;
        }
      }

      .toggle {
        padding: 6px 4px;
      }
    }

    .content {
      padding: 8px 12px 8px 28px;
      border: 0;
      border-bottom: 1px solid #DCDEE5;
      border-radius: 0;
    }
  }

  .content {
    padding: 16px 32px;
    background-color: #fff;
    border: 1px solid #DCDEE5;
    border-top: none;
    border-radius: 2px;
  }
}

</style>
