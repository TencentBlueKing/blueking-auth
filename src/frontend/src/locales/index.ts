import { createI18n } from 'vue-i18n';
import cn from './cn.json';
import en from './en.json';
import Cookie from 'js-cookie';

const cookieLocale = Cookie.get('blueking_language') || 'zh-cn';
const i18n = createI18n({
  legacy: false,
  locale: cookieLocale || 'zh-cn',
  fallbackLocale: 'zh-cn',
  messages: {
    'zh-cn': cn,
    'en': en,
  },
});
export const { t, locale } = i18n.global;

export default i18n;
