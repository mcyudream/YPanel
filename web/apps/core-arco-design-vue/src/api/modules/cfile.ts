import api from '../index'
import type { FileEntry, FileListResp, FileReadResp } from './file'

// 容器内文件 API（/api/v1/docker/containers/{id}/files/*），与宿主 file.ts 同形状。
// 搜索不做（容器内全盘 grep 开销大）；上传走 multipart，下载 token 直开。

export default {
  list: async (containerId: string, path: string) => {
    const res = await api.get(`api/v1/docker/containers/${encodeURIComponent(containerId)}/files/list?path=${encodeURIComponent(path)}`)
    return res.data as FileListResp
  },
  read: async (containerId: string, path: string, opts?: { raw?: boolean }) => {
    let q = `api/v1/docker/containers/${encodeURIComponent(containerId)}/files/read?path=${encodeURIComponent(path)}`
    if (opts?.raw) {
      q += '&raw=1'
    }
    const res = await api.get(q)
    return res.data as FileReadResp
  },
  write: (containerId: string, path: string, content: string) =>
    api.post(`api/v1/docker/containers/${encodeURIComponent(containerId)}/files/write`, { path, content }),
  mkdir: (containerId: string, path: string) =>
    api.post(`api/v1/docker/containers/${encodeURIComponent(containerId)}/files/mkdir`, { path }),
  rename: (containerId: string, from: string, to: string) =>
    api.post(`api/v1/docker/containers/${encodeURIComponent(containerId)}/files/rename`, { from, to }),
  delete: (containerId: string, paths: string[]) =>
    api.post(`api/v1/docker/containers/${encodeURIComponent(containerId)}/files/delete`, { paths }),
  chmod: (containerId: string, path: string, mode: string, recursive?: boolean) =>
    api.post(`api/v1/docker/containers/${encodeURIComponent(containerId)}/files/chmod`, { path, mode, recursive: !!recursive }),
  chown: (containerId: string, path: string, owner: string, group: string, recursive?: boolean) =>
    api.post(`api/v1/docker/containers/${encodeURIComponent(containerId)}/files/chown`, { path, owner, group, recursive: !!recursive }),
  upload: (containerId: string, dir: string, file: File, onProgress?: (percent: number) => void) => {
    const form = new FormData()
    form.append('file', file)
    return api.post(`api/v1/docker/containers/${encodeURIComponent(containerId)}/files/upload?path=${encodeURIComponent(dir)}`, form, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: (e) => {
        if (onProgress && e.total) {
          onProgress(Math.round((e.loaded / e.total) * 100))
        }
      },
    })
  },
  downloadURL: (containerId: string, path: string, token: string) =>
    `api/v1/docker/containers/${encodeURIComponent(containerId)}/files/download?path=${encodeURIComponent(path)}&token=${encodeURIComponent(token)}`,
}

export type { FileEntry, FileListResp, FileReadResp }
