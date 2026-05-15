<template>
  <el-container style="height: 100vh">
    <el-aside :width="isCollapse ? '64px' : '220px'" style="transition: width 0.3s; background: #1d1e1f">
      <div class="logo">
        <span v-show="!isCollapse">K8sGate</span>
      </div>
      <el-menu
        :default-active="$route.path"
        router
        background-color="#1d1e1f"
        text-color="#bfcbd9"
        active-text-color="#409eff"
        :collapse="isCollapse"
      >
        <el-menu-item index="/dashboard">
          <el-icon><Monitor /></el-icon>
          <template #title>集群概览</template>
        </el-menu-item>
        <el-menu-item index="/k8s-dashboard">
          <el-icon><Platform /></el-icon>
          <template #title>K8s Dashboard</template>
        </el-menu-item>
        <el-sub-menu v-if="user?.role === 'admin'" index="admin">
          <template #title>
            <el-icon><Setting /></el-icon>
            <span>系统管理</span>
          </template>
          <el-menu-item index="/users">
            <el-icon><User /></el-icon>
            <template #title>用户管理</template>
          </el-menu-item>
          <el-menu-item index="/projects">
            <el-icon><Folder /></el-icon>
            <template #title>项目管理</template>
          </el-menu-item>
        </el-sub-menu>
        <el-sub-menu index="audit">
          <template #title>
            <el-icon><Document /></el-icon>
            <span>审计中心</span>
          </template>
          <el-menu-item index="/audit-log">
            <el-icon><List /></el-icon>
            <template #title>操作日志</template>
          </el-menu-item>
          <el-menu-item index="/audit-report">
            <el-icon><DataAnalysis /></el-icon>
            <template #title>操作报告</template>
          </el-menu-item>
          <el-menu-item v-if="user?.role === 'admin'" index="/login-log">
            <el-icon><Unlock /></el-icon>
            <template #title>登录日志</template>
          </el-menu-item>
          <el-menu-item v-if="user?.role === 'admin'" index="/global-stats">
            <el-icon><TrendCharts /></el-icon>
            <template #title>全局统计</template>
          </el-menu-item>
        </el-sub-menu>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header style="display: flex; align-items: center; justify-content: space-between; border-bottom: 1px solid #e4e7ed; background: #fff">
        <el-icon style="cursor: pointer; font-size: 20px" @click="isCollapse = !isCollapse">
          <Fold v-if="!isCollapse" />
          <Expand v-else />
        </el-icon>
        <div style="display: flex; align-items: center; gap: 16px">
          <span style="color: #606266; font-size: 14px">{{ user?.display_name || user?.username }}</span>
          <el-tag size="small" :type="roleTagType">{{ roleLabel }}</el-tag>
          <el-dropdown @command="handleCommand">
            <el-icon style="cursor: pointer; font-size: 18px"><MoreFilled /></el-icon>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="password">修改密码</el-dropdown-item>
                <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <el-main style="background: #f0f2f5; overflow-y: auto">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { logout, getMe } from '../api/auth'

const router = useRouter()
const isCollapse = ref(false)
const user = ref(null)

const roleLabel = computed(() => {
  const map = { admin: '管理员', developer: '开发者', viewer: '只读' }
  return map[user.value?.role] || user.value?.role
})

const roleTagType = computed(() => {
  const map = { admin: 'danger', developer: '', viewer: 'info' }
  return map[user.value?.role] || ''
})

onMounted(async () => {
  try {
    const res = await getMe()
    user.value = res.data
    sessionStorage.setItem('user', JSON.stringify(res.data))
  } catch (e) {
    router.push('/login')
  }
})

async function handleCommand(cmd) {
  if (cmd === 'logout') {
    await logout()
    sessionStorage.removeItem('user')
    ElMessage.success('已退出')
    router.push('/login')
  } else if (cmd === 'password') {
    router.push('/change-password')
  }
}
</script>

<style scoped>
.logo {
  height: 50px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #409eff;
  font-size: 20px;
  font-weight: bold;
  border-bottom: 1px solid #333;
}
</style>
