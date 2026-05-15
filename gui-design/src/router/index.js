import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  { path: '/',           name: 'dashboard', component: () => import('../views/Dashboard.vue'), meta: { title: '仪表盘' } },
  { path: '/providers',  name: 'providers', component: () => import('../views/Providers.vue'), meta: { title: 'Provider 管理' } },
  { path: '/models',     name: 'models',    component: () => import('../views/Models.vue'),    meta: { title: '模型管理' } },
  { path: '/routes',     name: 'routes',    component: () => import('../views/Routes.vue'),    meta: { title: '路由管理' } },
  { path: '/settings',   name: 'settings',  component: () => import('../views/Settings.vue'),  meta: { title: '设置' } },
  { path: '/config',     name: 'config',    component: () => import('../views/Config.vue'),    meta: { title: '配置管理' } },
  { path: '/changes',    name: 'changes',   component: () => import('../views/Changes.vue'),   meta: { title: '变更管理' } },
  { path: '/sessions',   name: 'sessions',  component: () => import('../views/Sessions.vue'),  meta: { title: '会话' } },
  { path: '/stats',      name: 'stats',     component: () => import('../views/Stats.vue'),     meta: { title: '统计分析' } },
  { path: '/logs',       name: 'logs',      component: () => import('../views/Logs.vue'),      meta: { title: '日志' } },
]

export default createRouter({
  history: createWebHashHistory(),
  routes,
})
