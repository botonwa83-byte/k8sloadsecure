<template>
  <div>
    <div style="display: flex; justify-content: space-between; margin-bottom: 16px">
      <el-input v-model="query.keyword" placeholder="搜索用户名/姓名/邮箱" style="width: 300px" clearable @clear="loadData" @keyup.enter="loadData">
        <template #append>
          <el-button icon="Search" @click="loadData" />
        </template>
      </el-input>
      <el-button type="primary" icon="Plus" @click="showDialog()">新建用户</el-button>
    </div>

    <el-table :data="users" stripe v-loading="loading">
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="username" label="用户名" width="120" />
      <el-table-column prop="display_name" label="姓名" width="120" />
      <el-table-column prop="email" label="邮箱" />
      <el-table-column prop="role" label="角色" width="100">
        <template #default="{ row }">
          <el-tag :type="{ admin: 'danger', developer: '', global_viewer: 'info' }[row.role]" size="small">
            {{ { admin: '管理员', developer: '开发者', global_viewer: '只读' }[row.role] }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="80">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : 'danger'" size="small">
            {{ row.status === 'active' ? '正常' : '禁用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="password_expires_at" label="密码过期" width="170">
        <template #default="{ row }">
          {{ row.password_expires_at ? new Date(row.password_expires_at).toLocaleDateString() : '-' }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="showDialog(row)">编辑</el-button>
          <el-button size="small" type="warning" @click="showResetDialog(row)">重置密码</el-button>
          <el-popconfirm title="确定删除该用户？" @confirm="handleDelete(row.id)">
            <template #reference>
              <el-button size="small" type="danger">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      style="margin-top: 16px; justify-content: flex-end"
      v-model:current-page="query.page"
      v-model:page-size="query.page_size"
      :total="total"
      :page-sizes="[20, 50, 100]"
      layout="total, sizes, prev, pager, next"
      @change="loadData"
    />

    <!-- 新建/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="editingUser ? '编辑用户' : '新建用户'" width="500px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" :disabled="!!editingUser" />
        </el-form-item>
        <el-form-item v-if="!editingUser" label="密码" prop="password">
          <el-input v-model="form.password" type="password" show-password />
        </el-form-item>
        <el-form-item label="姓名" prop="display_name">
          <el-input v-model="form.display_name" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="form.email" />
        </el-form-item>
        <el-form-item label="角色" prop="role">
          <el-select v-model="form.role" style="width: 100%">
            <el-option label="只读" value="global_viewer" />
            <el-option label="开发者" value="developer" />
            <el-option label="管理员" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="editingUser" label="状态">
          <el-select v-model="form.status" style="width: 100%">
            <el-option label="正常" value="active" />
            <el-option label="禁用" value="disabled" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitLoading" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>

    <!-- 重置密码对话框 -->
    <el-dialog v-model="resetVisible" title="重置密码" width="400px">
      <el-form ref="resetFormRef" :model="resetForm" :rules="resetRules" label-width="80px">
        <el-form-item label="新密码" prop="new_password">
          <el-input v-model="resetForm.new_password" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitLoading" @click="handleReset">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getUserList, createUser, updateUser, resetPassword, deleteUser } from '../api/user'

const loading = ref(false)
const submitLoading = ref(false)
const users = ref([])
const total = ref(0)
const query = reactive({ page: 1, page_size: 20, keyword: '' })

const dialogVisible = ref(false)
const editingUser = ref(null)
const formRef = ref(null)
const form = reactive({ username: '', password: '', display_name: '', email: '', role: 'global_viewer', status: 'active' })

const resetVisible = ref(false)
const resetFormRef = ref(null)
const resetForm = reactive({ new_password: '' })
const resetUserId = ref(null)

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }, { min: 8, message: '密码至少8位', trigger: 'blur' }],
  role: [{ required: true, message: '请选择角色', trigger: 'change' }],
}
const resetRules = {
  new_password: [{ required: true, message: '请输入新密码', trigger: 'blur' }, { min: 8, message: '密码至少8位', trigger: 'blur' }],
}

onMounted(() => loadData())

async function loadData() {
  loading.value = true
  try {
    const res = await getUserList(query)
    users.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

function showDialog(user) {
  editingUser.value = user || null
  if (user) {
    Object.assign(form, { username: user.username, password: '', display_name: user.display_name, email: user.email, role: user.role, status: user.status })
  } else {
    Object.assign(form, { username: '', password: '', display_name: '', email: '', role: 'global_viewer', status: 'active' })
  }
  dialogVisible.value = true
}

function showResetDialog(user) {
  resetUserId.value = user.id
  resetForm.new_password = ''
  resetVisible.value = true
}

async function handleSubmit() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  submitLoading.value = true
  try {
    if (editingUser.value) {
      await updateUser(editingUser.value.id, { display_name: form.display_name, email: form.email, role: form.role, status: form.status })
    } else {
      await createUser(form)
    }
    ElMessage.success(editingUser.value ? '更新成功' : '创建成功')
    dialogVisible.value = false
    loadData()
  } finally {
    submitLoading.value = false
  }
}

async function handleReset() {
  const valid = await resetFormRef.value.validate().catch(() => false)
  if (!valid) return
  submitLoading.value = true
  try {
    await resetPassword(resetUserId.value, resetForm)
    ElMessage.success('密码重置成功')
    resetVisible.value = false
  } finally {
    submitLoading.value = false
  }
}

async function handleDelete(id) {
  await deleteUser(id)
  ElMessage.success('删除成功')
  loadData()
}
</script>
