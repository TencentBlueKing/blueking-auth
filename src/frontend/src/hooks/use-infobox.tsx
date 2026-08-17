/** 基于infoBox二次封装  */
import { InfoBox } from 'bkui-vue';
import { isFunction } from 'lodash-es';

type Align = 'center' | 'left' | 'right';
type Theme = 'danger' | 'success' | 'warning' | 'primary';
type InfoType = 'danger' | 'success' | 'warning' | 'loading';

export interface IProps {
  isShow?: boolean
  width?: number | string
  class?: string | string[]
  type?: InfoType
  title?: (() => VNode | string) | VNode | string
  subTitle?: (() => VNode) | VNode | string
  content?: (() => VNode) | VNode | string
  footer?: (() => VNode) | VNode | string
  headerAlign?: Align
  footerAlign?: Align
  contentAlign?: Align
  showContentBgColor?: boolean
  showMask?: boolean
  quickClose?: boolean
  escClose?: boolean
  closeIcon?: boolean
  confirmText?: (() => VNode) | VNode | string
  cancelText?: (() => VNode) | VNode | string
  confirmButtonTheme?: Theme
  beforeClose?: (v: string) => Promise<boolean> | boolean
  onConfirm?: () => void
  onCancel?: () => void
}

export class InfoModel implements IProps {
  isShow?: boolean = false;
  width?: number | string = 480;
  contentAlign?: Align = 'left';
  showContentBgColor?: boolean = true;
}

export function usePopInfoBox(props: Partial<IProps>) {
  const infoBoxInstance = InfoBox(new InfoModel());

  const renderTitle = () => {
    return <div class="break-all info-box-title">{ isFunction(props.title) ? props.title() : props.title }</div>;
  };

  const renderContent = () => {
    const displayContent = props.subTitle ?? props.content;
    return <div class="break-all info-box-content">{isFunction(displayContent) ? displayContent() : displayContent }</div>;
  };

  const renderInfoBox = () => {
    const subTitle = props.content ? 'content' : 'subTitle';
    infoBoxInstance.update({
      ...props,
      title: renderTitle(),
      [subTitle]: renderContent(),
    });
    infoBoxInstance.show();
  };

  // 是否初始化的时候展示infoBox
  if (props?.isShow) {
    renderInfoBox();
  }
}
