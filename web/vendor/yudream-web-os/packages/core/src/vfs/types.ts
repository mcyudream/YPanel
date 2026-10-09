/** VFS 节点 */
export interface VNode {
  /** '/Documents/readme.md' */
  path: string
  name: string
  kind: 'file' | 'directory' | 'symlink'
  size: number
  mime?: string
  createdAt: number
  modifiedAt: number
  /** 扩展元数据（标签、星级） */
  meta?: Record<string, unknown>
}

export type VFSEventType = 'create' | 'write' | 'remove' | 'move'

export interface VFSEvent {
  path: string
  type: VFSEventType
}

/**
 * VFS 后端适配器：宿主可实现自己的后端（HTTP/远程 API）挂载进系统。
 */
export interface VFSAdapter {
  stat: (path: string) => Promise<VNode>
  readdir: (path: string) => Promise<VNode[]>
  read: (path: string) => Promise<string | ArrayBuffer>
  write: (path: string, data: string | ArrayBuffer) => Promise<void>
  mkdir: (path: string) => Promise<void>
  move: (src: string, dst: string) => Promise<void>
  copy: (src: string, dst: string) => Promise<void>
  remove: (path: string) => Promise<void>
  watch?: (path: string, cb: (event: VFSEvent) => void) => () => void
}
