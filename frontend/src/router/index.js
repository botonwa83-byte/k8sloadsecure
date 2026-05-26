import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
  },
  {
    path: '/change-password',
    name: 'ChangePassword',
    component: () => import('../views/ChangePassword.vue'),
  },
  {
    path: '/',
    component: () => import('../views/Layout.vue'),
    redirect: '/overview',
    children: [
      {
        path: 'overview',
        name: 'Dashboard',
        component: () => import('../views/Dashboard.vue'),
        meta: { title: '集群概览' },
      },
      {
        path: 'k8s-dashboard',
        name: 'K8sDashboard',
        component: () => import('../views/K8sDashboard.vue'),
        meta: { title: 'K8s Dashboard' },
      },
      {
        path: 'my-permissions',
        name: 'MyPermissions',
        component: () => import('../views/MyPermissions.vue'),
        meta: { title: '我的权限' },
      },
      {
        path: 'users',
        name: 'UserManage',
        component: () => import('../views/UserManage.vue'),
        meta: { title: '用户管理', admin: true },
      },
      {
        path: 'projects',
        name: 'ProjectManage',
        component: () => import('../views/ProjectManage.vue'),
        meta: { title: '项目管理', admin: true },
      },
      {
        path: 'audit-log',
        name: 'AuditLog',
        component: () => import('../views/AuditLog.vue'),
        meta: { title: '操作日志' },
      },
      {
        path: 'audit-report',
        name: 'AuditReport',
        component: () => import('../views/AuditReport.vue'),
        meta: { title: '操作报告' },
      },
      {
        path: 'login-log',
        name: 'LoginLog',
        component: () => import('../views/LoginLog.vue'),
        meta: { title: '登录日志', admin: true },
      },
      {
        path: 'global-stats',
        name: 'GlobalStats',
        component: () => import('../views/GlobalStats.vue'),
        meta: { title: '全局统计', admin: true },
      },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to, from, next) => {
  if (to.path !== '/login' && to.path !== '/change-password') {
    const user = sessionStorage.getItem('user')
    if (!user) {
      return next('/login')
    }
    if (to.meta.admin) {
      const userObj = JSON.parse(user)
      if (userObj.role !== 'admin') {
        return next('/overview')
      }
    }
  }
  next()
})

export default router
