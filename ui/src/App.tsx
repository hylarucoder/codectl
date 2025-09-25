import React, { useMemo } from 'react'
import { Outlet } from 'react-router'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
import { AppSidebar, type NavItem } from '@/components/app-sidebar'
import { FolderOpen, GitCompare, Home, Settings2, SquareTerminal, Wrench } from 'lucide-react'

export default function App() {

  const navItems: NavItem[] = useMemo(() => (
    [
      { title: 'Home', url: '/', icon: Home },
      { title: 'Explorer', url: '/explorer', icon: FolderOpen },
      { title: 'Diff', url: '/diff', icon: GitCompare },
      { title: 'Work', url: '/work', icon: Wrench },
      { title: 'Terminal', url: '/terminal', icon: SquareTerminal },
      { title: 'Settings', url: '/settings', icon: Settings2 },
    ]
  ), [])

  return (
    <SidebarProvider>
      <AppSidebar navMain={navItems} />
      <SidebarInset>
        <div className="p-3 flex flex-col h-full min-h-0">
          <div className="flex-1 min-h-0 flex">
            <Outlet />
          </div>
        </div>
      </SidebarInset>
    </SidebarProvider>
  )
}
