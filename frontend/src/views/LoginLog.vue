<template>
  <div>
    <el-card shadow="never" style="margin-bottom: 16px">
      <el-form :inline="true" :model="query">
        <el-form-item label="用户名">
          <el-input v-model="query.username" clearable placeholder="用户名" style="width: 150px" />
        </el-form-item>
        <el-form-item label="结果">
          <el-select v-model="query.result" clearable placeholder="全部" style="width: 120px">
            <el-option label="成功" value="success" />
            <el-option label="失败" value="failed" />
            <el-option label="锁定" value="locked" />
          </el-select>
        </el-form-item>
        <el-form-item label="时间">
          <el-date-picker v-model="dateRange" type="daterange" range-separator="至" start-placeholder="开始" end-placeholder="结束" value-format="YYYY-MM-DD" style="width: 260px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="Search" @click="loadData">查询</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-table :data="logs" stripe v-loading="loading">
      <el-table-column prop="created_at" label="时间" width="170">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column prop="username" label="用户名" width="120" />
      <el-table-column prop="client_ip" label="IP" width="140" />
      <el-table-column prop="result" label="结果" width="100">
        <template #default="{ row }">
          <el-tag :type="{ success: 'success', failed: 'danger', locked: 'warning' }[row.result]" size="small">
            {{ { success: '成功', failed: '失败', locked: '锁定' }[row.result] }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="reason" label="原因" />
    </el-table>

    <el-pagination
      style="margin-top: 16px; justify-content: flex-end"
      v-model:current-page="query.page"
      v-model:page-size="query.page_size"
      :total="total"
      :page-sizes="[50, 100]"
      layout="total, sizes, prev, pager, next"
      @change="loadData"
    />
  </div>
</template>

<script setup>
import { ref, reactive, watch, onMounted } from 'vue'
import { getLoginLogs } from '../api/audit'

const loading = ref(false)
const logs = ref([])
const total = ref(0)
const dateRange = ref([])
const query = reactive({ page: 1, page_size: 50, username: '', result: '', start_time: '', end_time: '' })

watch(dateRange, (val) => {
  query.start_time = val?.[0] || ''
  query.end_time = val?.[1] || ''
})

onMounted(() => loadData())

async function loadData() {
  loading.value = true
  try {
    const res = await getLoginLogs(query)
    logs.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

function formatTime(t) {
  return t ? new Date(t).toLocaleString('zh-CN') : ''
}
</script>
