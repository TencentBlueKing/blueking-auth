import { useWindowSize } from '@vueuse/core';
import { locale } from '@/locales';
import type { ReturnRecordType } from '@/types/common';
import router from '@/router';

type ITableLimit = {
  allocatedHeight: number
  customLineHeight: number
  mode?: string | undefined
  className: string
  hasPagination: boolean
};

/**
 * @description 根据表格设置大小获取表格行高
 * @param {String} className  获取缓存中每个table单独的class
 * @returns lineH行高  topHead 表头高度
 */
function getTableSizeLineHeight(className: string, mode = 'bkui'): Record<string, number> {
  const curSetting = localStorage.getItem(`table-setting-${locale.value}-${className}`);
  let tableSize = curSetting ? JSON.parse(curSetting)?.size : 'small';
  if (['tdesign'].includes(mode)) {
    tableSize = curSetting ? JSON.parse(curSetting)?.rowSize : 'medium';
  }
  // 后续其他表格也可以适配，默认先以bkui-vue表格为例
  const sizeMap: ReturnRecordType<string, Record<string, number>> = {
    mini: () => {
      if (['bkui'].includes(mode)) {
        return {
          lineH: 42,
          topHead: 42,
        };
      }
      return {
        lineH: 32,
        topHead: 32,
      };
    },
    small: () => {
      if (['bkui'].includes(mode)) {
        return {
          lineH: 42,
          topHead: 42,
        };
      }
      if (['tdesign'].includes(mode)) {
        return {
          lineH: 36,
          topHead: 36,
        };
      }
      return {
        lineH: 42,
        topHead: 42,
      };
    },
    medium: () => {
      if (['bkui'].includes(mode)) {
        return {
          lineH: 60,
          topHead: 42,
        };
      }
      if (['tdesign'].includes(mode)) {
        return {
          lineH: 42,
          topHead: 42,
        };
      }
      return {
        lineH: 60,
        topHead: 42,
      };
    },
    large: () => {
      if (['bkui'].includes(mode)) {
        return {
          lineH: 78,
          topHead: 42,
        };
      }
      if (['tdesign'].includes(mode)) {
        return {
          lineH: 56,
          topHead: 56,
        };
      }
      return {
        lineH: 44,
        topHead: 47,
      };
    },
  };
  return sizeMap[tableSize || 'mini']?.();
}

/**
 * @description 获取表格最大显示行数
 * @param {ITableLimit} payload
 * allocatedHeight 已占用不能用来展示表格行的高度
 * customLineHeight 自定义行高，覆盖默认表格设置行高
 * hasPagination 表格是否有分页
 * className 获取缓存中每个table唯一标识的class，处理表格设置大小
 * @returns maxTableLimit → 最大展示条数   clientHeight → 剩余可视化最大高度
 */
export function useMaxTableLimit(payload?: Partial<ITableLimit>) {
  const viewportHeight = toValue(useWindowSize().height);
  // 默认已占位高度
  const hasAllocatedHeight = payload?.allocatedHeight ?? 186;
  // 默认分页器高度
  let paginationH = payload?.hasPagination || typeof payload?.hasPagination === 'undefined' ? 60 : 0;
  if (['tdesign'].includes(payload?.mode as string)) {
    paginationH = 64;
  }
  // 获取表格的最大可视化区域
  const clientHeight = viewportHeight - hasAllocatedHeight;
  const name = payload?.className ?? router?.currentRoute?.value?.name;
  // topHead是指vxe-table根据不同表格大小动态设置了距离表头top
  const { lineH, topHead } = getTableSizeLineHeight(name as string, payload?.mode);
  // 优先获取自定义传入行高，默认设置不同大小表格的固定行高
  const rowHeight = payload?.customLineHeight ?? lineH;
  // 为了防止body区域出现滚动条，需要减去表头和分页器的高度
  const maxTableLimit = Math.max(Math.floor((clientHeight - paginationH - topHead) / rowHeight), 1);
  return {
    maxTableLimit,
    clientHeight,
  };
}
