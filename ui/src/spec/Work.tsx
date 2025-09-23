import React, { useEffect, useMemo, useState } from 'react'
import { Card, Flex, Input, Select, Table, Typography } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { api } from '../lib/api'
import type { TaskItem } from '../types'

export default function WorkView() {
  const [tasks, setTasks] = useState<TaskItem[]>([])
  const [status, setStatus] = useState<string>('All')
  const [owner, setOwner] = useState<string>('All')
  const [priority, setPriority] = useState<string>('All')
  const [q, setQ] = useState<string>('')

  const owners = useMemo(() => {
    const s = new Set<string>()
    for (const t of tasks) if (t.owner) s.add(t.owner)
    return ['All', ...Array.from(s).sort()]
  }, [tasks])

  async function load() {
    const params = new URLSearchParams()
    if (status !== 'All') params.set('status', status)
    if (owner !== 'All') params.set('owner', owner)
    if (priority !== 'All') params.set('priority', priority)
    if (q.trim()) params.set('q', q.trim())
    const url = '/api/tasks/list' + (params.toString() ? `?${params.toString()}` : '')
    const arr = await api<TaskItem[]>(url)
    setTasks(arr)
  }

  useEffect(() => { load() }, [status, owner, priority, q])

  const cols: ColumnsType<TaskItem> = [
    { title: 'Task', dataIndex: 'title', key: 'title', render: (v, r) => (<span>{v || r.path}</span>) },
    { title: 'Status', dataIndex: 'status', key: 'status' },
    { title: 'Owner', dataIndex: 'owner', key: 'owner' },
    { title: 'Pri', dataIndex: 'priority', key: 'priority', width: 80 },
    { title: 'Due', dataIndex: 'due', key: 'due', width: 140 },
  ]

  return (
    <Flex vertical gap={12} className="flex-1 min-h-0 h-full">
      <Card size="small" title="Filters">
        <Flex gap={8} wrap>
          <Select size="small" value={status} onChange={setStatus} options={[{ value: 'All' }, 'backlog','in-progress','blocked','done','draft','accepted'].map(v => ({ value: typeof v === 'string' ? v : v.value }))} />
          <Select size="small" value={owner} onChange={setOwner} options={owners.map(v => ({ value: v }))} />
          <Select size="small" value={priority} onChange={setPriority} options={[{ value: 'All' }, 'P0','P1','P2'].map(v => ({ value: typeof v === 'string' ? v : v.value }))} />
          <Input size="small" placeholder="search" value={q} onChange={(e) => setQ(e.target.value)} className="w-[200px]" />
        </Flex>
      </Card>
      <Card size="small" className="flex-1 min-h-0" bodyStyle={{ height: '100%', overflow: 'auto' }} title={<Typography.Text strong>Tasks</Typography.Text>}>
        <div className="min-h-full">
          <Table rowKey={(r) => r.path} dataSource={tasks} columns={cols} pagination={{ pageSize: 10 }} />
        </div>
      </Card>
    </Flex>
  )
}
