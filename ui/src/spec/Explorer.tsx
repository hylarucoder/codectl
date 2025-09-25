import React, { useEffect, useMemo, useState } from 'react'
import { Card, Flex, Tree, Typography, Spin } from 'antd'
import { FolderOutlined, FileMarkdownOutlined, FileOutlined, FileImageOutlined, CodeOutlined, FileTextOutlined } from '@ant-design/icons'
import ReactMarkdown from 'react-markdown'
import type { DataNode } from 'antd/es/tree'
import { api } from '../lib/api'
import type { FsTreeNode, SpecDocMeta } from '../types'

function fileIcon(path: string, isDir: boolean): React.ReactNode {
  if (isDir) return <FolderOutlined />
  const p = path.toLowerCase()
  if (p.endsWith('.spec.mdx')) return <FileMarkdownOutlined />
  if (p.endsWith('.task.mdx')) return <FileTextOutlined />
  if (p.endsWith('.md') || p.endsWith('.mdx')) return <FileMarkdownOutlined />
  if (p.endsWith('.json') || p.endsWith('.yaml') || p.endsWith('.yml') || p.endsWith('.toml')) return <FileTextOutlined />
  if (p.endsWith('.png') || p.endsWith('.jpg') || p.endsWith('.jpeg') || p.endsWith('.gif') || p.endsWith('.svg') || p.endsWith('.webp')) return <FileImageOutlined />
  if (p.endsWith('.ts') || p.endsWith('.tsx') || p.endsWith('.js') || p.endsWith('.jsx') || p.endsWith('.go') || p.endsWith('.py') || p.endsWith('.rs') || p.endsWith('.java') || p.endsWith('.rb') || p.endsWith('.sh')) return <CodeOutlined />
  return <FileOutlined />
}

function withIconTitle(name: string, icon: React.ReactNode): React.ReactNode {
  return (
    <span>
      <span className="mr-1.5 inline-flex items-center">{icon}</span>
      <span>{name}</span>
    </span>
  )
}

function toTreeData(n: FsTreeNode): DataNode {
  const key = n.path || n.name
  const titleText = n.name || n.path || ''
  const icon = fileIcon(n.path || '', n.dir)
  return {
    key,
    title: withIconTitle(titleText, icon),
    selectable: !n.dir,
    // Keep relative path for selection
    path: n.path || '',
    children: (n.children || []).map(c => toTreeData(c)) as any,
  } as any
}

export default function Explorer() {
  const [root, setRoot] = useState<FsTreeNode | null>(null)
  const [loading, setLoading] = useState(false)
  const [selectedPath, setSelectedPath] = useState<string>('')
  const [doc, setDoc] = useState<SpecDocMeta | null>(null)

  async function loadTree() {
    setLoading(true)
    try {
      const node = await api<FsTreeNode>('/api/fs/tree?base=vibe-spec&depth=4')
      setRoot(node)
    } finally {
      setLoading(false)
    }
  }

  async function openPath(p: string) {
    setSelectedPath(p)
    try {
      const q = '/api/spec/doc?base=vibe-spec&path=' + encodeURIComponent(p)
      const d = await api<SpecDocMeta>(q)
      setDoc(d)
    } catch (e) {
      setDoc({ path: p, content: '(failed to load file)' })
    }
  }

  useEffect(() => { loadTree() }, [])

  const treeData = useMemo(() => root ? [toTreeData({ ...root, name: 'spec' })] : [], [root])

  function stripFrontmatter(s?: string): string {
    if (!s) return ''
    const lines = s.split(/\r?\n/)
    if (lines.length > 0 && lines[0].trim() === '---') {
      for (let i = 1; i < lines.length; i++) {
        if (lines[i].trim() === '---') {
          return lines.slice(i + 1).join('\n')
        }
      }
    }
    return s
  }

  return (
    <Flex gap={12} className="flex-1 min-h-0 h-full">
      <Card size="small" title="Files" className="w-[360px] flex-none h-full" bodyStyle={{ height: '100%', overflow: 'auto' }}>
        {loading && <Spin />}
        {!loading && (
          <Tree
            showLine
            treeData={treeData}
            onSelect={(keys, info) => {
              const node: any = info.node
              const rel = (node && node.path) || ''
              if (rel) openPath(rel)
            }}
          />
        )}
      </Card>
      <Card size="small" className="flex-1 min-w-0 h-full" title={
        <Flex gap={8}>
          <Typography.Text strong>Preview</Typography.Text>
          {doc?.path && <Typography.Text type="secondary">{doc.path}</Typography.Text>}
        </Flex>
      } bodyStyle={{ height: '100%', overflow: 'auto' }}>
        {!doc && <Typography.Text type="secondary">Select a file (.spec.mdx) on the left to preview.</Typography.Text>}
        {doc && (
          <div className="pr-2">
            <ReactMarkdown>{stripFrontmatter(doc.content)}</ReactMarkdown>
          </div>
        )}
      </Card>
    </Flex>
  )
}
