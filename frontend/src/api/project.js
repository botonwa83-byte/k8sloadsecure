import request from '../utils/request'

export function getProjectList(params) {
  return request.get('/projects', { params })
}

export function createProject(data) {
  return request.post('/projects', data)
}

export function getProject(id) {
  return request.get(`/projects/${id}`)
}

export function updateProject(id, data) {
  return request.put(`/projects/${id}`, data)
}

export function deleteProject(id) {
  return request.delete(`/projects/${id}`)
}

export function assignUser(projectId, data) {
  return request.post(`/projects/${projectId}/users`, data)
}

export function removeUser(projectId, userId) {
  return request.delete(`/projects/${projectId}/users/${userId}`)
}

export function getNamespaces() {
  return request.get('/namespaces')
}
