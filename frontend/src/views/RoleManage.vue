<template>
  <div class="role-manage">
    <div class="header">
      <h2>角色管理</h2>
      <button class="btn btn-primary" @click="openCreateModal">创建角色</button>
    </div>

    <div class="search-bar">
      <input 
        type="text" 
        v-model="searchKeyword" 
        placeholder="搜索角色名称"
        @input="handleSearch"
      />
    </div>

    <table class="table">
      <thead>
        <tr>
          <th>角色名称</th>
          <th>描述</th>
          <th>类型</th>
          <th>父角色</th>
          <th>权限数</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="role in roleList" :key="role.id">
          <td>{{ role.name }}</td>
          <td>{{ role.description }}</td>
          <td>
            <span :class="['tag', role.type === 'system' ? 'tag-danger' : 'tag-success']">
              {{ role.type === 'system' ? '系统角色' : '自定义角色' }}
            </span>
          </td>
          <td>{{ getParentName(role.parent_id) }}</td>
          <td>{{ role.permissions ? role.permissions.length : 0 }}</td>
          <td>
            <button class="btn btn-sm btn-info" @click="viewRole(role)">查看</button>
            <button 
              class="btn btn-sm btn-warning" 
              @click="openEditModal(role)"
              :disabled="role.type === 'system'"
            >编辑</button>
            <button 
              class="btn btn-sm btn-danger" 
              @click="handleDelete(role)"
              :disabled="role.type === 'system'"
            >删除</button>
          </td>
        </tr>
      </tbody>
    </table>

    <div class="pagination" v-if="total > pageSize">
      <button 
        class="btn btn-sm" 
        :disabled="currentPage === 1"
        @click="currentPage--"
      >上一页</button>
      <span>{{ currentPage }} / {{ totalPages }}</span>
      <button 
        class="btn btn-sm" 
        :disabled="currentPage === totalPages"
        @click="currentPage++"
      >下一页</button>
    </div>

    <div class="modal" v-if="showModal" @click.self="closeModal">
      <div class="modal-content">
        <div class="modal-header">
          <h3>{{ isEdit ? '编辑角色' : '创建角色' }}</h3>
          <button class="close-btn" @click="closeModal">×</button>
        </div>
        <div class="modal-body">
          <form>
            <div class="form-group">
              <label>角色名称</label>
              <input 
                type="text" 
                v-model="formData.name" 
                placeholder="请输入角色名称"
                :disabled="isEdit && currentRole?.type === 'system'"
              />
            </div>
            <div class="form-group">
              <label>描述</label>
              <input 
                type="text" 
                v-model="formData.description" 
                placeholder="请输入角色描述"
              />
            </div>
            <div class="form-group">
              <label>父角色</label>
              <select v-model="formData.parent_id">
                <option :value="0">无</option>
                <option v-for="r in roleList" :key="r.id" :value="r.id">
                  {{ r.name }}
                </option>
              </select>
            </div>
            <div class="form-group">
              <label>权限配置</label>
              <div class="permission-grid">
                <div v-for="resource in resources" :key="resource.name" class="permission-item">
                  <label class="resource-label">{{ resource.label }}</label>
                  <div class="actions">
                    <label v-for="action in actions" :key="action">
                      <input 
                        type="checkbox" 
                        :checked="hasPermission(resource.name, action)"
                        @change="togglePermission(resource.name, action)"
                      />
                      {{ getActionLabel(action) }}
                    </label>
                  </div>
                </div>
              </div>
            </div>
          </form>
        </div>
        <div class="modal-footer">
          <button class="btn btn-secondary" @click="closeModal">取消</button>
          <button class="btn btn-primary" @click="handleSubmit">确认</button>
        </div>
      </div>
    </div>

    <div class="modal" v-if="showViewModal" @click.self="closeViewModal">
      <div class="modal-content">
        <div class="modal-header">
          <h3>角色详情 - {{ viewRoleData.name }}</h3>
          <button class="close-btn" @click="closeViewModal">×</button>
        </div>
        <div class="modal-body">
          <div class="detail-item">
            <span class="label">角色名称：</span>
            <span>{{ viewRoleData.name }}</span>
          </div>
          <div class="detail-item">
            <span class="label">描述：</span>
            <span>{{ viewRoleData.description }}</span>
          </div>
          <div class="detail-item">
            <span class="label">类型：</span>
            <span>{{ viewRoleData.type === 'system' ? '系统角色' : '自定义角色' }}</span>
          </div>
          <div class="detail-item">
            <span class="label">父角色：</span>
            <span>{{ getParentName(viewRoleData.parent_id) }}</span>
          </div>
          <div class="detail-item">
            <span class="label">权限列表：</span>
            <div class="permission-list">
              <div v-for="perm in viewRoleData.permissions" :key="perm.id" class="perm-item">
                <strong>{{ getResourceLabel(perm.resource) }}:</strong>
                <span>{{ perm.actions }}</span>
              </div>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn btn-primary" @click="closeViewModal">关闭</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { listRoles, createRole, updateRole, deleteRole, getRole } from '../api/role'

const roleList = ref([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(10)
const searchKeyword = ref('')
const showModal = ref(false)
const showViewModal = ref(false)
const isEdit = ref(false)
const currentRole = ref(null)
const viewRoleData = ref({})

const formData = ref({
  name: '',
  description: '',
  parent_id: 0,
  permissions: []
})

const resources = [
  { name: 'user', label: '用户管理' },
  { name: 'role', label: '角色管理' },
  { name: 'project', label: '项目管理' },
  { name: 'namespace', label: '命名空间' },
  { name: 'pod', label: 'Pod' },
  { name: 'deployment', label: 'Deployment' },
  { name: 'service', label: 'Service' },
  { name: 'configmap', label: 'ConfigMap' },
  { name: 'secret', label: 'Secret' },
  { name: 'audit', label: '审计日志' },
  { name: 'approval', label: '审批管理' }
]

const actions = ['view', 'create', 'update', 'delete', 'approve', 'export']

const totalPages = computed(() => Math.ceil(total.value / pageSize.value))

function getParentName(parentId) {
  if (!parentId) return '无'
  const parent = roleList.value.find(r => r.id === parentId)
  return parent ? parent.name : '未知'
}

function getActionLabel(action) {
  const labels = {
    view: '查看',
    create: '创建',
    update: '修改',
    delete: '删除',
    approve: '审批',
    export: '导出'
  }
  return labels[action] || action
}

function getResourceLabel(resource) {
  const res = resources.find(r => r.name === resource)
  return res ? res.label : resource
}

function hasPermission(resource, action) {
  const existing = formData.value.permissions.find(p => p.resource === resource)
  if (!existing) return false
  return existing.actions.includes(action)
}

function togglePermission(resource, action) {
  const index = formData.value.permissions.findIndex(p => p.resource === resource)
  if (index === -1) {
    formData.value.permissions.push({ resource, actions: [action] })
  } else {
    const actionIndex = formData.value.permissions[index].actions.indexOf(action)
    if (actionIndex === -1) {
      formData.value.permissions[index].actions.push(action)
    } else {
      formData.value.permissions[index].actions.splice(actionIndex, 1)
    }
  }
}

function loadRoles() {
  listRoles({ page: currentPage.value, page_size: pageSize.value }).then(res => {
    roleList.value = res.data.list
    total.value = res.data.total
  })
}

function handleSearch() {
  currentPage.value = 1
  loadRoles()
}

function openCreateModal() {
  isEdit.value = false
  currentRole.value = null
  formData.value = {
    name: '',
    description: '',
    parent_id: 0,
    permissions: []
  }
  showModal.value = true
}

function openEditModal(role) {
  isEdit.value = true
  currentRole.value = role
  formData.value = {
    name: role.name,
    description: role.description,
    parent_id: role.parent_id || 0,
    permissions: role.permissions ? role.permissions.map(p => ({
      resource: p.resource,
      actions: JSON.parse(p.actions)
    })) : []
  }
  showModal.value = true
}

function closeModal() {
  showModal.value = false
}

function handleSubmit() {
  const data = {
    name: formData.value.name,
    description: formData.value.description,
    parent_id: formData.value.parent_id,
    permissions: formData.value.permissions.map(p => `${p.resource}:${p.actions.join(',')}`)
  }

  if (isEdit.value) {
    updateRole(currentRole.value.id, data).then(() => {
      alert('更新成功')
      closeModal()
      loadRoles()
    }).catch(err => {
      alert(err.response?.data?.message || '更新失败')
    })
  } else {
    createRole(data).then(() => {
      alert('创建成功')
      closeModal()
      loadRoles()
    }).catch(err => {
      alert(err.response?.data?.message || '创建失败')
    })
  }
}

function viewRole(role) {
  getRole(role.id).then(res => {
    viewRoleData.value = res.data
    showViewModal.value = true
  })
}

function closeViewModal() {
  showViewModal.value = false
}

function handleDelete(role) {
  if (!confirm(`确定要删除角色 "${role.name}" 吗？`)) return
  deleteRole(role.id).then(() => {
    alert('删除成功')
    loadRoles()
  }).catch(err => {
    alert(err.response?.data?.message || '删除失败')
  })
}

onMounted(() => {
  loadRoles()
})
</script>

<style scoped>
.role-manage {
  padding: 20px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.header h2 {
  margin: 0;
}

.search-bar {
  margin-bottom: 20px;
}

.search-bar input {
  width: 300px;
  padding: 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
}

.table {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 20px;
}

.table th, .table td {
  padding: 12px;
  text-align: left;
  border-bottom: 1px solid #ddd;
}

.table th {
  background-color: #f5f5f5;
}

.tag {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.tag-danger {
  background-color: #f8d7da;
  color: #721c24;
}

.tag-success {
  background-color: #d4edda;
  color: #155724;
}

.btn {
  padding: 6px 12px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
}

.btn-primary {
  background-color: #007bff;
  color: white;
}

.btn-secondary {
  background-color: #6c757d;
  color: white;
}

.btn-info {
  background-color: #17a2b8;
  color: white;
}

.btn-warning {
  background-color: #ffc107;
  color: #212529;
}

.btn-danger {
  background-color: #dc3545;
  color: white;
}

.btn-sm {
  padding: 4px 8px;
  font-size: 12px;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 10px;
}

.modal {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
}

.modal-content {
  background-color: white;
  border-radius: 8px;
  width: 600px;
  max-height: 80vh;
  overflow-y: auto;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 15px;
  border-bottom: 1px solid #ddd;
}

.modal-header h3 {
  margin: 0;
}

.close-btn {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
}

.modal-body {
  padding: 15px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 15px;
  border-top: 1px solid #ddd;
}

.form-group {
  margin-bottom: 15px;
}

.form-group label {
  display: block;
  margin-bottom: 5px;
}

.form-group input, .form-group select {
  width: 100%;
  padding: 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
  box-sizing: border-box;
}

.permission-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
}

.permission-item {
  background-color: #f5f5f5;
  padding: 10px;
  border-radius: 4px;
}

.resource-label {
  font-weight: bold;
  margin-bottom: 5px;
  display: block;
}

.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.actions label {
  display: flex;
  align-items: center;
  gap: 5px;
  cursor: pointer;
}

.detail-item {
  margin-bottom: 10px;
}

.detail-item .label {
  font-weight: bold;
}

.permission-list {
  margin-top: 10px;
}

.perm-item {
  padding: 5px;
  background-color: #f5f5f5;
  margin-bottom: 5px;
  border-radius: 4px;
}
</style>