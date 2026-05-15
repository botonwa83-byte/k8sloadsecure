import request from '../utils/request'

export function listRoles(params) {
  return request({
    url: '/roles',
    method: 'get',
    params
  })
}

export function getRole(id) {
  return request({
    url: `/roles/${id}`,
    method: 'get'
  })
}

export function createRole(data) {
  return request({
    url: '/roles',
    method: 'post',
    data
  })
}

export function updateRole(id, data) {
  return request({
    url: `/roles/${id}`,
    method: 'put',
    data
  })
}

export function deleteRole(id) {
  return request({
    url: `/roles/${id}`,
    method: 'delete'
  })
}

export function getUserRoles(userId) {
  return request({
    url: `/user-roles/${userId}`,
    method: 'get'
  })
}

export function assignRole(userId, data) {
  return request({
    url: `/user-roles/${userId}`,
    method: 'post',
    data
  })
}

export function removeRole(userId, roleId) {
  return request({
    url: `/user-roles/${userId}/${roleId}`,
    method: 'delete'
  })
}