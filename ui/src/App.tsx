import React, { useMemo, useState } from 'react'
import { Layout, Menu, theme } from 'antd'
import { HomeOutlined, FolderOpenOutlined, DiffOutlined, ToolOutlined } from '@ant-design/icons'
import Home from './home/Home'
import Explorer from './spec/Explorer'
import DiffView from './spec/Diff'
import WorkView from './spec/Work'

const { Sider, Content } = Layout

export default function App() {
  const { token } = theme.useToken()
  const [tab, setTab] = useState<string>(() => {
    const h = (typeof location !== 'undefined' ? location.hash : '').replace(/^#/, '')
    return h && ['home','explorer','diff','work'].includes(h) ? h : 'home'
  })
  const menuItems = useMemo(() => ([
    { key: 'home', icon: <HomeOutlined />, label: 'Home' },
    { key: 'explorer', icon: <FolderOpenOutlined />, label: 'Explorer' },
    { key: 'diff', icon: <DiffOutlined />, label: 'Diff' },
    { key: 'work', icon: <ToolOutlined />, label: 'Work' },
  ]), [])
  const body = useMemo(() => {
    switch (tab) {
      case 'explorer': return <Explorer />
      case 'diff': return <DiffView />
      case 'work': return <WorkView />
      case 'home':
      default: return <Home />
    }
  }, [tab])

  // keep hash in sync
  function onSelect(next: string) {
    setTab(next)
    if (typeof history !== 'undefined') {
      const url = next ? `#${next}` : '#home'
      history.replaceState(null, '', url)
    }
  }

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
          onClick={(info) => onSelect(info.key)}
          style={{ borderRight: 0 }}
        />
      </Sider>
      <Layout style={{ minHeight: '100vh' }}>
        <Content style={{ padding: 12, display: 'flex', flexDirection: 'column', height: '100%', minHeight: 0 }}>
          <div style={{ flex: 1, minHeight: 0, display: 'flex' }}>
            {body}
          </div>
        </Content>
      </Layout>
    </Layout>
  )
}
