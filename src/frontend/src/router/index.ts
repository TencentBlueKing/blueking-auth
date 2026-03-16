import {
  type RouteRecordRaw,
  createRouter,
  createWebHistory,
} from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    path: '/oauth2/authorize',
    name: 'Authorize',
    component: () => import('@/views/authorize/Index.vue'),
  },
  {
    path: '/oauth2/result',
    name: 'Result',
    component: () => import('@/views/result/Index.vue'),
  },
  {
    path: '/oauth2/device',
    name: 'DeviceAuthorize',
    component: () => import('@/views/device/Index.vue'),
  },
  // {
  //   path: '/,
  //   name: '404',
  //   component: () => import('@/views/404.vue'),
  // },
];

const router = createRouter({
  history: createWebHistory(window.BK_SITE_PATH),
  routes,
});

export default router;
