import {
  type RouteRecordRaw,
  createRouter,
  createWebHistory,
} from 'vue-router';
import {
  DEFAULT_PERSONAL_TOKEN_REALM,
  isPersonalTokenRealm,
} from '@/constants/personal-token';

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
        redirect: {
          name: 'PersonalToken',
          params: { realm: DEFAULT_PERSONAL_TOKEN_REALM },
        },
        children: [
          {
            path: 'personal-token',
            redirect: {
              name: 'PersonalToken',
              params: { realm: DEFAULT_PERSONAL_TOKEN_REALM },
            },
          },
          {
            path: 'realms/:realm/personal-tokens',
            name: 'PersonalToken',
            component: () => import('@/views/personal-token/Index.vue'),
            beforeEnter: (to) => {
              if (!isPersonalTokenRealm(to.params.realm)) {
                return { name: '404' };
              }
              return true;
            },
            meta: {
              matchRoute: 'PersonalToken',
              title: '个人令牌',
            },
          },
        ],
      },
    ],
  },
  {
    path: '/404',
    name: '404',
    component: () => import('@/views/404.vue'),
  },
];

const router = createRouter({
  history: createWebHistory(window.BK_SITE_PATH),
  routes,
});

export default router;
