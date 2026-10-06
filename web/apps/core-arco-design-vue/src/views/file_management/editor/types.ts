import type { FileEntry } from '@/api/modules/file'

export interface FileTreeNode {
  entry: FileEntry
  children: FileTreeNode[] | null
  expanded: boolean
  loading: boolean
}

export interface FileTreeMenuItem {
  label: string
  icon?: string
  variant?: 'default' | 'destructive'
  disabled?: boolean
  handle?: () => void
}

export interface FileTreeApi {
  toggle: (node: FileTreeNode) => void
  openFile: (node: FileTreeNode) => void
  menuItems: (node: FileTreeNode) => FileTreeMenuItem[][]
}
