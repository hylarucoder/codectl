export type Role = 'user' | 'admin' | 'superuser'

export interface User {
  id: number
  email: string
  name?: string
  role: Role
  createdAt?: string
  updatedAt?: string
}

export interface ApiError {
  message: string
  traceId?: string
  status?: number
}

export interface HealthResponse { status: 'ok' | string }

export interface UploadResponse {
  id: number
  path: string
  name: string
  size: number
  mime: string
  url: string
}

// WebUI — Spec UI related types
export interface FsTreeNode {
  path: string
  name: string
  dir: boolean
  children?: FsTreeNode[]
}

export interface SpecDocMeta {
  path: string
  title?: string
  status?: string
  fields?: Record<string, string>
  errors?: string[]
  warnings?: string[]
  content?: string
}

export interface DiffChangeItem {
  path: string
  status: string // e.g., ' M', 'A ', '??'
  group: 'Staged' | 'Unstaged' | 'Untracked' | string
}

export interface DiffFileResponse {
  path: string
  mode: 'all' | 'staged' | 'worktree' | string
  diff: string
  split?: SplitRow[]
}

export interface TaskItem {
  path: string
  title?: string
  status?: string
  owner?: string
  priority?: string
  due?: string
  fields?: Record<string, string>
}

export type SplitSide = 'ctx' | 'del' | 'add' | 'meta' | 'empty'
export interface SplitRow { left: string; right: string; lt: SplitSide; rt: SplitSide; ln?: number; rn?: number }
