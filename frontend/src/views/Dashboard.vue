<template>
  <div>
    <h3 style="margin-top: 0">集群概览</h3>
    <el-row :gutter="16">
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>用户总数</template>
          <div class="stat-value">{{ stats.user_count || '-' }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>项目总数</template>
          <div class="stat-value">{{ stats.project_count || '-' }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>命名空间</template>
          <div class="stat-value">{{ stats.namespace_count || '-' }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>今日操作</template>
          <div class="stat-value">{{ stats.today_ops || '-' }}</div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 非管理员用户：权限列表 -->
    <el-card v-if="!isAdmin" style="margin-top: 16px" shadow="hover">
      <template #header>
        <div style="display: flex; align-items: center; justify-content: space-between">
          <span>我的权限</span>
          <el-tag :type="roleTagType" size="small">{{ roleLabel }}</el-tag>
        </div>
      </template>

      <!-- 权限汇总 -->
      <el-row :gutter="16" style="margin-bottom: 16px">
        <el-col :span="8">
          <div class="perm-summary">
            <div class="perm-summary-label">已分配项目</div>
            <div class="perm-summary-value">{{ projects.length }}</div>
          </div>
        </el-col>
        <el-col :span="8">
          <div class="perm-summary">
            <div class="perm-summary-label">可访问命名空间</div>
            <div class="perm-summary-value">{{ allNamespaces.length }}</div>
          </div>
        </el-col>
        <el-col :span="8">
          <div class="perm-summary">
            <div class="perm-summary-label">读写权限项目</div>
            <div class="perm-summary-value">{{ rwCount }}</div>
          </div>
        </el-col>
      </el-row>

      <!-- 可访问的命名空间汇总 -->
      <div v-if="allNamespaces.length > 0" style="margin-bottom: 16px">
        <div style="font-size: 13px; color: #909399; margin-bottom: 8px">可访问的命名空间：</div>
        <div>
          <el-tag v-for="ns in allNamespaces" :key="ns" size="small" style="margin: 2px">{{ ns }}</el-tag>
        </div>
      </div>
      <el-alert
        v-else
        type="warning"
        :closable="false"
        title="您尚未被分配任何项目权限，请联系管理员。"
        style="margin-bottom: 16px"
      />

      <!-- 权限明细表 -->
      <el-table :data="projects" stripe>
        <el-table-column prop="project_name" label="项目名" min-width="120" />
        <el-table-column prop="permission" label="权限" width="100">
          <template #default="{ row }">
            <el-tag :type="row.permission === 'readwrite' ? 'success' : 'info'" size="small">
              {{ row.permission === 'readwrite' ? '读写' : '只读' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="可操作范围" min-width="150">
          <template #default="{ row }">
            <span style="font-size: 12px; color: #909399">
              {{ row.permission === 'readwrite' ? '可查看、创建、修改、删除资源' : '仅可查看资源，不可修改' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="namespaces" label="关联命名空间" min-width="200">
          <template #default="{ row }">
            <el-tag v-for="ns in row.namespaces" :key="ns" size="small" style="margin: 2px">{{ ns }}</el-tag>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 管理员用户：简化的项目列表 -->
    <el-card v-else style="margin-top: 16px" shadow="hover">
      <template #header>我的项目</template>
      <el-table :data="projects" stripe>
        <el-table-column prop="project_name" label="项目名" />
        <el-table-column prop="permission" label="权限">
          <template #default="{ row }">
            <el-tag :type="row.permission === 'readwrite' ? '' : 'info'" size="small">
              {{ row.permission === 'readwrite' ? '读写' : '只读' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="namespaces" label="关联命名空间">
          <template #default="{ row }">
            <el-tag v-for="ns in row.namespaces" :key="ns" size="small" style="margin: 2px">{{ ns }}</el-tag>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="projects.length === 0" description="暂无项目分配" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getDashboardStats } from '../api/audit'

const stats = ref({})
const projects = ref([])

const user = computed(() => {
  try {
    return JSON.parse(sessionStorage.getItem('user') || '{}')
  } catch {
    return {}
  }
})

const isAdmin = computed(() => user.value.role === 'admin')

const roleLabel = computed(() => {
  const map = { admin: '管理员', developer: '开发者', global_viewer: '全局只读', viewer: '只读' }
  return map[user.value.role] || user.value.role
})

const roleTagType = computed(() => {
  const map = { admin: 'danger', developer: '', global_viewer: 'warning', viewer: 'info' }
  return map[user.value.role] || ''
})

const allNamespaces = computed(() => {
  const nsSet = new Set()
  for (const p of projects.value) {
    for (const ns of (p.namespaces || [])) {
      nsSet.add(ns)
    }
  }
  return [...nsSet].sort()
})

const rwCount = computed(() => {
  return projects.value.filter(p => p.permission === 'readwrite').length
})

onMounted(async () => {
  try {
    const res = await getDashboardStats()
    stats.value = res.data
    projects.value = res.data.projects || []
  } catch (e) {
    // handled
  }
})
</script>

<style scoped>
.stat-value {
  font-size: 28px;
  font-weight: bold;
  color: #409eff;
  text-align: center;
}

.perm-summary {
  text-align: center;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 6px;
}

.perm-summary-label {
  font-size: 13px;
  color: #909399;
  margin-bottom: 4px;
}

.perm-summary-value {
  font-size: 22px;
  font-weight: bold;
  color: #303133;
}
</style>
