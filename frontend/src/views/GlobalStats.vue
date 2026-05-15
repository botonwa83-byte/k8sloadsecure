<template>
  <div>
    <el-card shadow="never" style="margin-bottom: 16px">
      <el-form :inline="true">
        <el-form-item label="时间范围">
          <el-date-picker v-model="dateRange" type="daterange" range-separator="至" start-placeholder="开始" end-placeholder="结束" value-format="YYYY-MM-DD" style="width: 260px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="DataAnalysis" @click="loadStats">查询统计</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <div v-if="stats" v-loading="loading">
      <el-row :gutter="16" style="margin-bottom: 16px">
        <el-col :span="6">
          <el-card shadow="hover">
            <template #header>总操作数</template>
            <div class="stat-value">{{ stats.total_operations }}</div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover">
            <template #header>活跃用户数</template>
            <div class="stat-value">{{ stats.active_users }}</div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover">
            <template #header>被拒绝操作</template>
            <div class="stat-value" style="color: #e6a23c">{{ stats.denied_operations }}</div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover">
            <template #header>操作类型</template>
            <div v-for="(count, action) in stats.by_action" :key="action" style="display: flex; justify-content: space-between; font-size: 13px">
              <span>{{ action }}</span>
              <span style="font-weight: bold">{{ count }}</span>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <el-row :gutter="16">
        <el-col :span="12">
          <el-card shadow="hover">
            <template #header>用户活跃度 TOP 20</template>
            <el-table :data="stats.top_users" stripe size="small">
              <el-table-column type="index" label="#" width="50" />
              <el-table-column prop="username" label="用户名" />
              <el-table-column prop="count" label="操作次数" width="100" />
            </el-table>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card shadow="hover">
            <template #header>命名空间操作热度 TOP 20</template>
            <el-table :data="stats.top_namespaces" stripe size="small">
              <el-table-column type="index" label="#" width="50" />
              <el-table-column prop="namespace" label="命名空间" />
              <el-table-column prop="count" label="操作次数" width="100" />
            </el-table>
          </el-card>
        </el-col>
      </el-row>
    </div>

    <el-empty v-else-if="!loading" description="选择时间范围后点击查询" />
  </div>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { getGlobalStats } from '../api/audit'

const loading = ref(false)
const stats = ref(null)
const dateRange = ref([])
const query = reactive({ start_time: '', end_time: '' })

watch(dateRange, (val) => {
  query.start_time = val?.[0] || ''
  query.end_time = val?.[1] || ''
})

async function loadStats() {
  if (!query.start_time || !query.end_time) {
    ElMessage.warning('请选择时间范围')
    return
  }
  loading.value = true
  try {
    const res = await getGlobalStats(query)
    stats.value = res.data
  } finally {
    loading.value = false
  }
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
