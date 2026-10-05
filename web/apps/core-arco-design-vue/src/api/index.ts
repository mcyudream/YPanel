import axios from 'axios'

// YPanel 统一 API 封装。
// 后端契约：HTTP 200 + { code: 0 成功 | 业务错误码, message, data }；鉴权 Authorization: Bearer <jwt>。
// 会话失效（2001/2002）全局登出；业务错误全局 toast，组件可 catch 自行处理。
// 下载/上传等需要原始响应的场景使用 api 的原始实例方法并自行处理。

declare module 'axios' {
  export interface AxiosRequestConfig {
    /** 业务错误时不弹全局 toast，由调用方自行处理 */
    silent?: boolean
  }
}

const api = axios.create({
  baseURL: (import.meta.env.DEV && import.meta.env.VITE_ENABLE_PROXY) ? '/proxy/' : import.meta.env.VITE_APP_API_BASEURL,
  timeout: 1000 * 60,
  responseType: 'json',
})

// 原始实例：不解析业务信封（文件下载等）
export const rawApi = axios.create({
  baseURL: api.defaults.baseURL,
  timeout: 1000 * 60 * 10,
})

rawApi.interceptors.request.use((request) => {
  const appAccountStore = useAppAccountStore()
  if (request.headers && appAccountStore.isLogin) {
    request.headers.Authorization = `Bearer ${appAccountStore.token}`
  }
  return request
})

api.interceptors.request.use(
  (request) => {
    const appAccountStore = useAppAccountStore()
    if (request.headers) {
      request.headers['Accept-Language'] = 'zh-CN'
      if (appAccountStore.isLogin && request.headers.Authorization === undefined) {
        request.headers.Authorization = `Bearer ${appAccountStore.token}`
      }
    }
    return request
  },
)

// 业务错误 → 统一提示（config.silent 可跳过全局提示，由调用方自行处理）；2001/2002 会话失效全局登出
function handleBizError(code: number, message: string, silent?: boolean) {
  if (code === 2001 || code === 2002) {
    useAppAccountStore().requestLogout()
    useFaToast().error('登录失效', { description: message })
  }
  else if (!silent) {
    useFaToast().error('操作失败', { description: message })
  }
  return Promise.reject({ code, message })
}

api.interceptors.response.use(
  (response) => {
    if (typeof response.data === 'object' && response.data !== null && 'code' in response.data) {
      if (response.data.code !== 0) {
        return handleBizError(response.data.code, response.data.message || '未知错误', response.config.silent)
      }
      // 约定：返回完整信封 { code, message, data }，模块层取 .data
      return Promise.resolve(response.data)
    }
    return Promise.reject(response)
  },
  error => Promise.reject(error),
)

export default api
