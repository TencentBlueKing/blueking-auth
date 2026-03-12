import { createApp } from 'vue';
import { createPinia } from 'pinia';

import App from './App.vue';
import router from './router';

// 全量引入 bkui-vue
import bkui from 'bkui-vue';
// 全量引入 bkui-vue 样式
import 'bkui-vue/dist/cli.css';
// UnoCSS
import 'virtual:uno.css';
import '@unocss/reset/tailwind-compat.css';

import i18n from './locales';

import directive from '@/directives';
import AgIcon from '@/components/ag-icon/Index.vue';
import IconButton from '@/components/icon-button/Index.vue';

const app = createApp(App);

app.use(createPinia())
  .use(router)
  .use(bkui)
  .use(i18n)
  .use(directive)
  // 全局组件
  .component('AgIcon', AgIcon)
  .component('IconButton', IconButton)
  .mount('#app');
