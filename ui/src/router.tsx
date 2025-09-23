import React, { Suspense, lazy } from 'react'
import { createBrowserRouter } from 'react-router'
import App from './App'

const Home = lazy(() => import('./home/Home'))
const Explorer = lazy(() => import('./spec/Explorer'))
const DiffView = lazy(() => import('./spec/Diff'))
const WorkView = lazy(() => import('./spec/Work'))
const Settings = lazy(() => import('./settings/Settings'))
const TerminalView = lazy(() => import('./terminal/TerminalView'))

export const router = createBrowserRouter([
  {
    path: '/',
    element: <App />,
    children: [
      { index: true, element: <Suspense fallback={<div style={{ padding: 12 }}>Loading…</div>}><Home /></Suspense> },
      { path: 'explorer', element: <Suspense fallback={<div style={{ padding: 12 }}>Loading…</div>}><Explorer /></Suspense> },
      { path: 'diff', element: <Suspense fallback={<div style={{ padding: 12 }}>Loading…</div>}><DiffView /></Suspense> },
      { path: 'work', element: <Suspense fallback={<div style={{ padding: 12 }}>Loading…</div>}><WorkView /></Suspense> },
      { path: 'terminal', element: <Suspense fallback={<div style={{ padding: 12 }}>Loading…</div>}><TerminalView /></Suspense> },
      { path: 'settings', element: <Suspense fallback={<div style={{ padding: 12 }}>Loading…</div>}><Settings /></Suspense> },
    ],
  },
])

export default router
