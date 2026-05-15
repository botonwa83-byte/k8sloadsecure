import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '../router'

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
  withCredentials: true,
})

request.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.code !== 0) {
      ElMessage.error(res.message || '请求失败')
      if (res.code === 40102) {
        router.push('/login')
      }
      return Promise.reject(new Error(res.message))
    }
    return res
  },
  (error) => {
    if (error.response) {
      const { status, data } = error.response
      if (status === 401) {
        router.push('/login')
      }
      ElMessage.error(data?.message || '请求失败')
    } else {
      ElMessage.error('网络异常')
    }
    return Promise.reject(error)
  }
)

export default request
