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

    <el-card style="margin-top: 16px" shadow="hover">
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
import { ref, onMounted } from 'vue'
import { getMe } from '../api/auth'

const stats = ref({})
const projects = ref([])

onMounted(async () => {
  try {
    const res = await getMe()
    projects.value = res.data.projects || []
    // 简易统计，后续可扩展
    const user = JSON.parse(sessionStorage.getItem('user') || '{}')
    stats.value = {
      namespace_count: new Set(projects.value.flatMap(p => p.namespaces)).size || '-',
      project_count: projects.value.length || '-',
      user_count: user.role === 'admin' ? '...' : '-',
      today_ops: '...',
    }
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
</style>
