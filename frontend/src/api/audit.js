import request from '../utils/request'

export function getAuditLogs(params) {
  return request.get('/audit/logs', { params })
}

export function getAuditReport(params) {
  return request.get('/audit/report', { params })
}

export function getGlobalStats(params) {
  return request.get('/audit/stats', { params })
}

export function getLoginLogs(params) {
  return request.get('/login-logs', { params })
}

export function exportAuditCSV(params) {
  const query = new URLSearchParams(params).toString()
  window.open(`/api/v1/audit/export?${query}`, '_blank')
}
