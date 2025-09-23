import React, { useMemo } from 'react'
import { Layout, Menu, theme } from 'antd'
import { HomeOutlined, FolderOpenOutlined, DiffOutlined, ToolOutlined, SettingOutlined, CodeOutlined } from '@ant-design/icons'
import { Outlet, useLocation, useNavigate } from 'react-router'

const { Sider, Content } = Layout

export default function App() {
  const { token } = theme.useToken()
  const location = useLocation()
  const navigate = useNavigate()
  const pathname = (location.pathname.replace(/^\/+/, '') || 'home')
  const menuItems = useMemo(() => ([
    { key: 'home', icon: <HomeOutlined />, label: 'Home' },
    { key: 'explorer', icon: <FolderOpenOutlined />, label: 'Explorer' },
    { key: 'diff', icon: <DiffOutlined />, label: 'Diff' },
    { key: 'work', icon: <ToolOutlined />, label: 'Work' },
    { key: 'terminal', icon: <CodeOutlined />, label: 'Terminal' },
    { key: 'settings', icon: <SettingOutlined />, label: 'Settings' },
  ]), [])
  const onSelect = (next: string) => navigate(next === 'home' ? '/' : `/${next}`)

  return (
    <Layout className="min-h-screen">
      <Sider theme="dark" width={220} style={{ borderRight: `1px solid ${token.colorBorderSecondary}` }}>
        <div
          className="h-12 flex items-center px-4 font-semibold cursor-pointer"
          style={{ color: token.colorTextLightSolid }}
          onClick={() => onSelect('home')}
          title="Go Home"
        >
          codectl
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[pathname]}
          items={menuItems as any}
          onClick={(info) => onSelect(info.key)}
          style={{ borderRight: 0 }}
        />
      </Sider>
      <Layout className="min-h-screen">
        <Content className="p-3 flex flex-col h-full min-h-0">
          <div className="flex-1 min-h-0 flex">
            <Outlet />
          </div>
        </Content>
      </Layout>
    </Layout>
  )
}
