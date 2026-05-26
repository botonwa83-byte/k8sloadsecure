<template>
  <div>
    <h3 style="margin-top: 0">我的权限</h3>

    <!-- 角色信息 -->
    <el-card shadow="hover">
      <template #header>基本信息</template>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="用户名">{{ user.username }}</el-descriptions-item>
        <el-descriptions-item label="姓名">{{ user.display_name || '-' }}</el-descriptions-item>
        <el-descriptions-item label="角色">
          <el-tag :type="roleTagType" size="small">{{ roleLabel }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="角色说明">{{ roleDesc }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <!-- 命名空间权限汇总 -->
    <el-card style="margin-top: 16px" shadow="hover">
      <template #header>
        <div style="display: flex; align-items: center; justify-content: space-between">
          <span>命名空间访问权限</span>
          <el-tag size="small" type="info">共 {{ allNamespaces.length }} 个命名空间</el-tag>
        </div>
      </template>

      <el-alert
        v-if="allNamespaces.length === 0"
        type="warning"
        :closable="false"
        title="您尚未被分配任何命名空间权限"
        description="请联系管理员将您分配到相关项目中，分配后即可在 K8s Dashboard 中查看和管理对应命名空间的资源。"
      />

      <div v-else>
        <el-table :data="nsPermList" stripe>
          <el-table-column prop="namespace" label="命名空间" min-width="150" />
          <el-table-column prop="project_name" label="所属项目" min-width="120" />
          <el-table-column prop="permission" label="权限" width="100">
            <template #default="{ row }">
              <el-tag :type="row.permission === 'readwrite' ? 'success' : 'info'" size="small">
                {{ row.permission === 'readwrite' ? '读写' : '只读' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="可执行操作" min-width="200">
            <template #default="{ row }">
              <div style="font-size: 12px; color: #606266">
                <template v-if="row.permission === 'readwrite'">
                  <el-tag size="small" type="success" style="margin: 1px">查看</el-tag>
                  <el-tag size="small" type="success" style="margin: 1px">创建</el-tag>
                  <el-tag size="small" type="success" style="margin: 1px">修改</el-tag>
                  <el-tag size="small" type="success" style="margin: 1px">删除</el-tag>
                </template>
                <template v-else>
                  <el-tag size="small" type="success" style="margin: 1px">查看</el-tag>
                  <el-tag size="small" type="danger" style="margin: 1px" effect="plain">创建</el-tag>
                  <el-tag size="small" type="danger" style="margin: 1px" effect="plain">修改</el-tag>
                  <el-tag size="small" type="danger" style="margin: 1px" effect="plain">删除</el-tag>
                </template>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-card>

    <!-- 项目权限明细 -->
    <el-card style="margin-top: 16px" shadow="hover">
      <template #header>项目权限明细</template>
      <el-table :data="projects" stripe>
        <el-table-column prop="project_name" label="项目名称" min-width="120" />
        <el-table-column prop="permission" label="权限等级" width="100">
          <template #default="{ row }">
            <el-tag :type="row.permission === 'readwrite' ? 'success' : 'info'" size="small">
              {{ row.permission === 'readwrite' ? '读写' : '只读' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="关联命名空间" min-width="250">
          <template #default="{ row }">
            <el-tag v-for="ns in row.namespaces" :key="ns" size="small" style="margin: 2px">{{ ns }}</el-tag>
            <span v-if="!row.namespaces || row.namespaces.length === 0" style="color: #909399; font-size: 12px">无</span>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="projects.length === 0" description="暂无项目权限" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getMe } from '../api/auth'

const user = ref({})
const projects = ref([])

const roleLabel = computed(() => {
  const map = { admin: '管理员', developer: '开发者', global_viewer: '全局只读', viewer: '只读' }
  return map[user.value.role] || user.value.role
})

const roleTagType = computed(() => {
  const map = { admin: 'danger', developer: '', global_viewer: 'warning', viewer: 'info' }
  return map[user.value.role] || ''
})

const roleDesc = computed(() => {
  const map = {
    admin: '拥有系统所有权限，可管理用户、项目和所有命名空间',
    developer: '只能访问被分配项目中的命名空间，权限由管理员控制',
    global_viewer: '可查看所有命名空间资源，但不可进行任何写操作',
    viewer: '只读用户'
  }
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

const nsPermList = computed(() => {
  const nsMap = {}
  for (const p of projects.value) {
    for (const ns of (p.namespaces || [])) {
      // 如果同一命名空间存在多个项目权限，取最高权限
      if (!nsMap[ns] || (p.permission === 'readwrite' && nsMap[ns].permission !== 'readwrite')) {
        nsMap[ns] = {
          namespace: ns,
          project_name: p.project_name,
          permission: p.permission,
        }
      }
    }
  }
  return Object.values(nsMap).sort((a, b) => a.namespace.localeCompare(b.namespace))
})

onMounted(async () => {
  try {
    const res = await getMe()
    user.value = res.data
    projects.value = res.data.projects || []
  } catch {
    // handled
  }
})
</script>
