import React, { useMemo, useState } from 'react'
import { Layout, Tabs, Typography } from 'antd'
import Explorer from './Explorer'
import DiffView from './Diff'
import WorkView from './Work'

const { Header, Content } = Layout

export default function SpecUI() {
  const [tab, setTab] = useState<string>('explorer')
  const items = useMemo(() => ([
    { key: 'explorer', label: 'Explorer', children: <Explorer /> },
    { key: 'diff', label: 'Diff', children: <DiffView /> },
    { key: 'work', label: 'Work', children: <WorkView /> },
  ]), [])

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ padding: '12px 16px' }}>
        <Typography.Title level={4} style={{ margin: 0, color: '#fff' }}>Spec UI</Typography.Title>
      </Header>
      <Content style={{ padding: 12 }}>
        <Tabs activeKey={tab} onChange={setTab as any} items={items as any} />
      </Content>
      {/* no footer/status bar */}
    </Layout>
  )
}
