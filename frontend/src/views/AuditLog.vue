<template>
  <div>
    <el-card shadow="never" style="margin-bottom: 16px">
      <el-form :inline="true" :model="query">
        <el-form-item label="操作类型">
          <el-select v-model="query.action" clearable placeholder="全部" style="width: 120px">
            <el-option label="GET" value="GET" />
            <el-option label="POST" value="POST" />
            <el-option label="PUT" value="PUT" />
            <el-option label="PATCH" value="PATCH" />
            <el-option label="DELETE" value="DELETE" />
          </el-select>
        </el-form-item>
        <el-form-item label="命名空间">
          <el-input v-model="query.namespace" clearable placeholder="命名空间" style="width: 150px" />
        </el-form-item>
        <el-form-item label="时间范围">
          <el-date-picker v-model="dateRange" type="daterange" range-separator="至" start-placeholder="开始" end-placeholder="结束" value-format="YYYY-MM-DD" style="width: 260px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="Search" @click="loadData">查询</el-button>
          <el-button icon="Download" @click="handleExport">导出CSV</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-table :data="logs" stripe v-loading="loading">
      <el-table-column prop="created_at" label="时间" width="170">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column prop="username" label="用户" width="100" />
      <el-table-column prop="action" label="操作" width="80">
        <template #default="{ row }">
          <el-tag :type="actionType(row.action)" size="small">{{ row.action }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="resource_type" label="资源类型" width="120" />
      <el-table-column prop="resource_name" label="资源名" />
      <el-table-column prop="namespace" label="命名空间" width="130" />
      <el-table-column prop="status_code" label="状态码" width="80">
        <template #default="{ row }">
          <el-tag :type="row.status_code < 400 ? 'success' : 'danger'" size="small">{{ row.status_code }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="detail" label="操作摘要" />
      <el-table-column prop="client_ip" label="IP" width="130" />
    </el-table>

    <el-pagination
      style="margin-top: 16px; justify-content: flex-end"
      v-model:current-page="query.page"
      v-model:page-size="query.page_size"
      :total="total"
      :page-sizes="[50, 100, 200]"
      layout="total, sizes, prev, pager, next"
      @change="loadData"
    />
  </div>
</template>

<script setup>
import { ref, reactive, watch, onMounted } from 'vue'
import { getAuditLogs, exportAuditCSV } from '../api/audit'

const loading = ref(false)
const logs = ref([])
const total = ref(0)
const dateRange = ref([])
const query = reactive({ page: 1, page_size: 50, action: '', namespace: '', start_time: '', end_time: '' })

watch(dateRange, (val) => {
  query.start_time = val?.[0] || ''
  query.end_time = val?.[1] || ''
})

onMounted(() => loadData())

async function loadData() {
  loading.value = true
  try {
    const res = await getAuditLogs(query)
    logs.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

function handleExport() {
  exportAuditCSV(query)
}

function formatTime(t) {
  if (!t) return ''
  return new Date(t).toLocaleString('zh-CN')
}

function actionType(action) {
  const map = { GET: 'info', POST: 'success', PUT: '', PATCH: 'warning', DELETE: 'danger' }
  return map[action] || ''
}
</script>
