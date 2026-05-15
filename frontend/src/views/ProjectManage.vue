<template>
  <div>
    <div style="display: flex; justify-content: space-between; margin-bottom: 16px">
      <el-input v-model="query.keyword" placeholder="搜索项目名" style="width: 300px" clearable @clear="loadData" @keyup.enter="loadData">
        <template #append>
          <el-button icon="Search" @click="loadData" />
        </template>
      </el-input>
      <el-button type="primary" icon="Plus" @click="showDialog()">新建项目</el-button>
    </div>

    <el-table :data="projects" stripe v-loading="loading">
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="项目名" width="180" />
      <el-table-column prop="description" label="描述" />
      <el-table-column prop="namespaces" label="命名空间">
        <template #default="{ row }">
          <el-tag v-for="ns in row.namespaces" :key="ns" size="small" style="margin: 2px">{{ ns }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="user_count" label="用户数" width="80" />
      <el-table-column label="操作" width="250" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="showDialog(row)">编辑</el-button>
          <el-button size="small" type="success" @click="showUserDialog(row)">分配用户</el-button>
          <el-popconfirm title="确定删除该项目？" @confirm="handleDelete(row.id)">
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
      layout="total, sizes, prev, pager, next"
      @change="loadData"
    />

    <!-- 新建/编辑项目 -->
    <el-dialog v-model="dialogVisible" :title="editingProject ? '编辑项目' : '新建项目'" width="600px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="项目名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="命名空间" prop="namespaces">
          <el-select v-model="form.namespaces" multiple filterable style="width: 100%" placeholder="选择命名空间">
            <el-option v-for="ns in allNamespaces" :key="ns" :label="ns" :value="ns" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitLoading" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>

    <!-- 分配用户 -->
    <el-dialog v-model="userDialogVisible" title="分配用户" width="600px">
      <div style="margin-bottom: 16px; display: flex; gap: 8px">
        <el-select v-model="assignForm.user_id" filterable placeholder="选择用户" style="flex: 1">
          <el-option v-for="u in allUsers" :key="u.id" :label="`${u.username} (${u.display_name})`" :value="u.id" />
        </el-select>
        <el-select v-model="assignForm.permission" style="width: 120px">
          <el-option label="只读" value="read" />
          <el-option label="读写" value="readwrite" />
        </el-select>
        <el-button type="primary" @click="handleAssign">添加</el-button>
      </div>

      <el-table :data="projectUsers" stripe>
        <el-table-column prop="user.username" label="用户名" />
        <el-table-column prop="user.display_name" label="姓名" />
        <el-table-column prop="permission" label="权限" width="100">
          <template #default="{ row }">
            <el-tag :type="row.permission === 'readwrite' ? '' : 'info'" size="small">
              {{ row.permission === 'readwrite' ? '读写' : '只读' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="80">
          <template #default="{ row }">
            <el-popconfirm title="确定移除？" @confirm="handleRemoveUser(row.user_id)">
              <template #reference>
                <el-button size="small" type="danger">移除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getProjectList, createProject, updateProject, deleteProject, getProject, assignUser, removeUser, getNamespaces } from '../api/project'
import { getUserList } from '../api/user'

const loading = ref(false)
const submitLoading = ref(false)
const projects = ref([])
const total = ref(0)
const query = reactive({ page: 1, page_size: 20, keyword: '' })
const allNamespaces = ref([])
const allUsers = ref([])

const dialogVisible = ref(false)
const editingProject = ref(null)
const formRef = ref(null)
const form = reactive({ name: '', description: '', namespaces: [] })
const rules = {
  name: [{ required: true, message: '请输入项目名', trigger: 'blur' }],
  namespaces: [{ required: true, type: 'array', min: 1, message: '请选择命名空间', trigger: 'change' }],
}

const userDialogVisible = ref(false)
const currentProjectId = ref(null)
const projectUsers = ref([])
const assignForm = reactive({ user_id: null, permission: 'readwrite' })

onMounted(async () => {
  loadData()
  try {
    const nsRes = await getNamespaces()
    allNamespaces.value = nsRes.data || []
  } catch (e) { /* handled */ }
  try {
    const userRes = await getUserList({ page: 1, page_size: 1000 })
    allUsers.value = userRes.data.list || []
  } catch (e) { /* handled */ }
})

async function loadData() {
  loading.value = true
  try {
    const res = await getProjectList(query)
    projects.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

function showDialog(project) {
  editingProject.value = project || null
  if (project) {
    Object.assign(form, { name: project.name, description: project.description, namespaces: project.namespaces || [] })
  } else {
    Object.assign(form, { name: '', description: '', namespaces: [] })
  }
  dialogVisible.value = true
}

async function handleSubmit() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  submitLoading.value = true
  try {
    if (editingProject.value) {
      await updateProject(editingProject.value.id, form)
    } else {
      await createProject(form)
    }
    ElMessage.success(editingProject.value ? '更新成功' : '创建成功')
    dialogVisible.value = false
    loadData()
  } finally {
    submitLoading.value = false
  }
}

async function handleDelete(id) {
  await deleteProject(id)
  ElMessage.success('删除成功')
  loadData()
}

async function showUserDialog(project) {
  currentProjectId.value = project.id
  userDialogVisible.value = true
  const res = await getProject(project.id)
  projectUsers.value = res.data.users || []
}

async function handleAssign() {
  if (!assignForm.user_id) {
    ElMessage.warning('请选择用户')
    return
  }
  await assignUser(currentProjectId.value, assignForm)
  ElMessage.success('分配成功')
  const res = await getProject(currentProjectId.value)
  projectUsers.value = res.data.users || []
  loadData()
}

async function handleRemoveUser(userId) {
  await removeUser(currentProjectId.value, userId)
  ElMessage.success('移除成功')
  const res = await getProject(currentProjectId.value)
  projectUsers.value = res.data.users || []
  loadData()
}
</script>
