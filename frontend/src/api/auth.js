import request from '../utils/request'

export function login(data) {
  return request.post('/auth/login', data)
}

export function logout() {
  return request.post('/auth/logout')
}

export function changePassword(data) {
  return request.put('/auth/password', data)
}

export function getMe() {
  return request.get('/auth/me')
}

export function getMyNamespaces() {
  return request.get('/my/namespaces')
}
