import {
  type RouteRecordRaw,
  createRouter,
  createWebHistory,
} from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    path: '/oauth2',
    name: 'OAuth2',
    component: () => import('@/layouts/oauth2/Index.vue'),
    redirect: { name: 'Authorize' },
    children: [
      {
        path: 'authorize',
        name: 'Authorize',
        component: () => import('@/views/authorize/Index.vue'),
      },
      {
        path: 'result',
        name: 'Result',
        component: () => import('@/views/result/Index.vue'),
      },
      {
        path: 'device',
        name: 'DeviceAuthorize',
        component: () => import('@/views/device/Index.vue'),
      },
    ],
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: () => import('@/layouts/dashboard/Index.vue'),
    redirect: { name: 'Token' },
    children: [
      {
        path: 'token',
        name: 'Token',
        component: () => import('@/views/token-mgmt/Index.vue'),
        redirect: { name: 'PersonalToken' },
        children: [
          {
            path: 'personal-token',
            name: 'PersonalToken',
            component: () => import('@/views/personal-token/Index.vue'),
            meta: {
              matchRoute: 'PersonalToken',
              title: '个人令牌',
            },
          },
        ],
      },
    ],
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
