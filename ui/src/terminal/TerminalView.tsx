import React, { useEffect, useRef } from 'react'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

export default function TerminalView() {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const termRef = useRef<Terminal | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    const term = new Terminal({
      convertEol: true,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
      fontSize: 13,
      cursorBlink: true,
      allowTransparency: false,
      theme: {
        background: '#2d2E2c',
        foreground: '#F0F0F0',
        cursor: '#FFFFFF',
        selectionBackground: '#3a3c38',
      },
    })
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.open(containerRef.current!)
    fit.fit()
    term.focus()

    termRef.current = term
    fitRef.current = fit

    // Setup WebSocket bridge
    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const url = `${proto}://${location.host}/api/term/ws`
    const ws = new WebSocket(url)
    wsRef.current = ws

    // Ensure we can read binary if server sends it
    ws.binaryType = 'arraybuffer'

    ws.onopen = () => {
      // Send initial size
      const dims = term._core?._renderService?.dimensions || { cols: term.cols, rows: term.rows }
      const cols = term.cols
      const rows = term.rows
      ws.send(JSON.stringify({ type: 'resize', cols, rows }))
    }
    ws.onmessage = (ev) => {
      if (ev.data instanceof ArrayBuffer) {
        const s = new TextDecoder().decode(new Uint8Array(ev.data))
        term.write(s)
      } else if (typeof ev.data === 'string') {
        term.write(ev.data)
      }
    }
    ws.onclose = () => {
      // Show a small message in terminal
      term.writeln('\r\n[connection closed]')
    }
    ws.onerror = () => {
      term.writeln('\r\n[connection error]')
    }

    const onData = term.onData((d) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'input', data: d }))
      }
    })
    const onResize = term.onResize(({ cols, rows }) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', cols, rows }))
      }
    })

    const onWindowResize = () => {
      fit.fit()
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
      }
    }
    window.addEventListener('resize', onWindowResize)

    const onContainerClick = () => {
      term.focus()
    }

    const el = containerRef.current!
    el?.addEventListener('click', onContainerClick)

    return () => {
      el?.removeEventListener('click', onContainerClick)
      window.removeEventListener('resize', onWindowResize)
      onData.dispose()
      onResize.dispose()
      ws.close()
      term.dispose()
    }
  }, [])

  return (
    <div style={{ flex: 1, display: 'flex', minHeight: 0, background: '#2d2E2c' }}>
      <div ref={containerRef} style={{ flex: 1, minHeight: 0, outline: 'none' }} tabIndex={0} />
    </div>
  )
}
