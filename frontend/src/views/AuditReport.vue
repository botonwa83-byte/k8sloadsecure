<template>
  <div>
    <el-card shadow="never" style="margin-bottom: 16px">
      <el-form :inline="true" :model="query">
        <el-form-item v-if="isAdmin" label="用户">
          <el-select v-model="query.user_id" filterable placeholder="选择用户" style="width: 200px">
            <el-option v-for="u in allUsers" :key="u.id" :label="`${u.username} (${u.display_name})`" :value="u.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="时间范围">
          <el-date-picker v-model="dateRange" type="daterange" range-separator="至" start-placeholder="开始" end-placeholder="结束" value-format="YYYY-MM-DD" style="width: 260px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="DataAnalysis" @click="loadReport">生成报告</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <div v-if="report" v-loading="loading">
      <el-row :gutter="16" style="margin-bottom: 16px">
        <el-col :span="6">
          <el-card shadow="hover">
            <template #header>总操作数</template>
            <div class="stat-value">{{ report.summary.total_operations }}</div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover">
            <template #header>成功</template>
            <div class="stat-value" style="color: #67c23a">{{ report.summary.by_result.success }}</div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover">
            <template #header>失败</template>
            <div class="stat-value" style="color: #f56c6c">{{ report.summary.by_result.failed }}</div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover">
            <template #header>被拒绝</template>
            <div class="stat-value" style="color: #e6a23c">{{ report.summary.by_result.denied }}</div>
          </el-card>
        </el-col>
      </el-row>

      <el-row :gutter="16" style="margin-bottom: 16px">
        <el-col :span="12">
          <el-card shadow="hover">
            <template #header>操作类型分布</template>
            <div v-for="(count, action) in report.summary.by_action" :key="action" style="display: flex; justify-content: space-between; margin-bottom: 8px">
              <span>{{ action }}</span>
              <el-progress :percentage="calcPercent(count, report.summary.total_operations)" :stroke-width="16" style="width: 70%" />
            </div>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card shadow="hover">
            <template #header>活跃信息</template>
            <p>活跃天数：<strong>{{ report.summary.active_days }}</strong> 天</p>
            <p>活跃命名空间：</p>
            <el-tag v-for="ns in report.summary.active_namespaces" :key="ns" style="margin: 2px">{{ ns }}</el-tag>
          </el-card>
        </el-col>
      </el-row>

      <el-card shadow="hover" v-if="report.sensitive_operations?.length">
        <template #header>
          <span style="color: #f56c6c">敏感操作记录</span>
        </template>
        <el-table :data="report.sensitive_operations" stripe>
          <el-table-column prop="created_at" label="时间" width="170">
            <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column prop="action" label="操作" width="80">
            <template #default="{ row }">
              <el-tag type="danger" size="small">{{ row.action }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="resource_type" label="资源类型" width="120" />
          <el-table-column prop="resource_name" label="资源名" />
          <el-table-column prop="namespace" label="命名空间" width="130" />
          <el-table-column prop="status_code" label="状态码" width="80" />
        </el-table>
      </el-card>
    </div>

    <el-empty v-else-if="!loading" description="选择时间范围后点击生成报告" />
  </div>
</template>

<script setup>
import { ref, reactive, watch, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { getAuditReport } from '../api/audit'
import { getUserList } from '../api/user'

const loading = ref(false)
const report = ref(null)
const dateRange = ref([])
const allUsers = ref([])

const user = computed(() => JSON.parse(sessionStorage.getItem('user') || '{}'))
const isAdmin = computed(() => user.value.role === 'admin')

const query = reactive({ user_id: null, start_time: '', end_time: '' })

watch(dateRange, (val) => {
  query.start_time = val?.[0] || ''
  query.end_time = val?.[1] || ''
})

onMounted(async () => {
  if (!isAdmin.value) {
    query.user_id = user.value.user_id
  }
  if (isAdmin.value) {
    try {
      const res = await getUserList({ page: 1, page_size: 1000 })
      allUsers.value = res.data.list || []
    } catch (e) { /* handled */ }
  }
})

async function loadReport() {
  if (!query.start_time || !query.end_time) {
    ElMessage.warning('请选择时间范围')
    return
  }
  if (!query.user_id) {
    ElMessage.warning('请选择用户')
    return
  }
  loading.value = true
  try {
    const res = await getAuditReport(query)
    report.value = res.data
  } finally {
    loading.value = false
  }
}

function formatTime(t) {
  return t ? new Date(t).toLocaleString('zh-CN') : ''
}

function calcPercent(count, total) {
  if (!total) return 0
  return Math.round((count / total) * 100)
}
</script>

<style scoped>
.stat-value {
  font-size: 28px;
  font-weight: bold;
  text-align: center;
  color: #409eff;
}
</style>
