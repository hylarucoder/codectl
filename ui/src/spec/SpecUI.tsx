import React, { useMemo, useState } from 'react'
import { Layout, Menu, theme } from 'antd'
import { FolderOpenOutlined, DiffOutlined, ToolOutlined } from '@ant-design/icons'
import Explorer from './Explorer'
import DiffView from './Diff'
import WorkView from './Work'

const { Sider, Content } = Layout

export default function SpecUI() {
  const { token } = theme.useToken()
  const [tab, setTab] = useState<string>('explorer')

  const menuItems = useMemo(() => ([
    { key: 'explorer', icon: <FolderOpenOutlined />, label: 'Explorer' },
    { key: 'diff', icon: <DiffOutlined />, label: 'Diff' },
    { key: 'work', icon: <ToolOutlined />, label: 'Work' },
  ]), [])

  const body = useMemo(() => {
    switch (tab) {
      case 'diff': return <DiffView />
      case 'work': return <WorkView />
      case 'explorer':
      default: return <Explorer />
    }
  }, [tab])

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider theme="dark" width={220} style={{ borderRight: `1px solid ${token.colorBorderSecondary}` }}>
        <div style={{ height: 48, display: 'flex', alignItems: 'center', padding: '0 16px', color: token.colorTextLightSolid, fontWeight: 600 }}>
          codectl
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[tab]}
          items={menuItems as any}
          onClick={(info) => setTab(info.key)}
          style={{ borderRight: 0 }}
        />
      </Sider>
      <Layout>
        <Content style={{ padding: 12 }}>
          {body}
        </Content>
      </Layout>
    </Layout>
  )
}
