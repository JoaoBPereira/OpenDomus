import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: () => import('@/layouts/LoginLayout.vue')
  },
  {
    path: '/main',
    component: () => import('@/layouts/MainLayout.vue'),
    children: [{ path: '', component: () => import('@/pages/IndexPage.vue') }],
  },
  {
    path: '/devices',
    component: () => import('@/layouts/MainLayout.vue'),
    children: [{ path: '', component: () => import('@/pages/DevicesPage.vue') }],
  },
  {
    path: '/logs',
    component: () => import('@/layouts/MainLayout.vue'),
    children: [{ path: '', component: () => import('@/pages/LogsPage.vue') }],
  },
  {
    path: '/tags',
    component: () => import('@/layouts/MainLayout.vue'),
    children: [{ path: '', component: () => import('@/pages/TagsPage.vue') }],
  },
  {
    path: '/tag-value-float',
    component: () => import('@/layouts/MainLayout.vue'),
    children: [{ path: '', component: () => import('@/pages/TagValueFloatPage.vue') }],
  },

  {
    path: '/automations-time',
    component: () => import('@/layouts/MainLayout.vue'),
    children: [{ path: '', component: () => import('@/pages/AutomationsPage.vue'), props: { trigger: 'timer' } }],
  },
  {
    path: '/automations-values',
    component: () => import('@/layouts/MainLayout.vue'),
    children: [{ path: '', component: () => import('@/pages/AutomationsPage.vue'), props: { trigger: 'tagValueFloat' } }],
  },
  {
    path: '/automations/:id',
    component: () => import('@/layouts/MainLayout.vue'),
    children: [{ path: '', component: () => import('@/pages/AutomationEditorPage.vue') }],
  },

  // Always leave this as last one,
  // but you can also remove it
  {
    path: '/:catchAll(.*)*',
    component: () => import('@/pages/ErrorNotFound.vue'),
  },
];

export default routes;
