<template>
  <div style="height: calc(100vh - 130px)">
    <!-- 加载中 -->
    <div v-if="loading" class="status-container">
      <el-icon class="is-loading" :size="40" color="#409eff"><Loading /></el-icon>
      <p style="margin-top: 16px; color: #909399">正在检测 K8s Dashboard 服务...</p>
    </div>

    <!-- 不可用 -->
    <div v-else-if="!available" class="status-container">
      <el-result icon="warning" title="K8s Dashboard 不可用" :sub-title="reason">
        <template #extra>
          <div style="text-align: left; background: #f5f7fa; padding: 16px; border-radius: 8px; max-width: 600px; margin: 0 auto">
            <p style="font-weight: bold; margin-bottom: 8px">可能的原因：</p>
            <ul style="margin: 0; padding-left: 20px; line-height: 2">
              <li>K8s Dashboard 服务未安装或未启动</li>
              <li>Dashboard 服务地址配置不正确</li>
              <li>网络不通（后端无法访问 Dashboard 服务���</li>
              <li>kubectl proxy 未运行（如使用 proxy 方式）</li>
            </ul>
          </div>
          <el-button type="primary" style="margin-top: 16px" @click="checkStatus">
            <el-icon style="margin-right: 4px"><RefreshRight /></el-icon>重新检测
          </el-button>
        </template>
      </el-result>
    </div>

    <!-- 正常 -->
    <template v-else>
      <!-- 命名空间权限提示 -->
      <div v-if="!isAllNamespaces && namespaces.length > 0" class="ns-info-bar">
        <el-icon><InfoFilled /></el-icon>
        <span>您可访问的命名空间：</span>
        <el-tag v-for="ns in namespaces" :key="ns" size="small" style="margin-left: 4px">{{ ns }}</el-tag>
      </div>
      <div v-else-if="!isAllNamespaces && namespaces.length === 0" class="ns-info-bar ns-warning">
        <el-icon><WarningFilled /></el-icon>
        <span>您尚未被分配任何命名空间权限，请联系管理员分配项目权限。</span>
      </div>
      <iframe
        src="/dashboard/"
        :style="{ width: '100%', height: iframeHeight, border: 'none', borderRadius: '4px' }"
        allow="clipboard-read; clipboard-write"
      />
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import request from '../utils/request'
import { getMyNamespaces } from '../api/auth'

const loading = ref(true)
const available = ref(false)
const reason = ref('')
const namespaces = ref([])
const isAllNamespaces = ref(true)

const iframeHeight = computed(() => {
  if (!isAllNamespaces.value && namespaces.value.length > 0) {
    return 'calc(100% - 40px)'
  }
  if (!isAllNamespaces.value && namespaces.value.length === 0) {
    return 'calc(100% - 40px)'
  }
  return '100%'
})

async function checkStatus() {
  loading.value = true
  available.value = false
  reason.value = ''
  try {
    const res = await request.get('/dashboard/status')
    available.value = res.data.available
    if (!res.data.available) {
      reason.value = res.data.reason || 'Dashboard 服务无响应'
    }
  } catch {
    available.value = false
    reason.value = '状态检测请求失败，请确认后端服务是否正常'
  } finally {
    loading.value = false
  }
}

async function loadNamespaces() {
  try {
    const res = await getMyNamespaces()
    namespaces.value = res.data.namespaces || []
    isAllNamespaces.value = res.data.all
  } catch {
    // 获取失败不影响主流��
  }
}

onMounted(() => {
  checkStatus()
  loadNamespaces()
})
</script>

<style scoped>
.status-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.ns-info-bar {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  background: #ecf5ff;
  border-radius: 4px;
  margin-bottom: 8px;
  font-size: 13px;
  color: #409eff;
  flex-wrap: wrap;
  gap: 2px;
}

.ns-info-bar.ns-warning {
  background: #fdf6ec;
  color: #e6a23c;
}
</style>
