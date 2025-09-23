import React, { useEffect, useMemo, useRef, useState } from 'react'
import { Button, Card, Flex, Input, Typography, theme } from 'antd'
import { api } from '../lib/api'

interface Session {
  id: string
  title: string
  created: string
}

interface SessionMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  ts: string
}

export default function Home() {
  const { token } = theme.useToken()
  const [session, setSession] = useState<Session | null>(null)
  const [messages, setMessages] = useState<SessionMessage[]>([])
  const [input, setInput] = useState('')
  const [busy, setBusy] = useState(false)
  const listRef = useRef<HTMLDivElement>(null)

  // Auto scroll when messages change
  useEffect(() => {
    const el = listRef.current
    if (el) { el.scrollTop = el.scrollHeight }
  }, [messages])

  // Init session and SSE
  useEffect(() => {
    let es: EventSource | null = null
    let cancelled = false
    ;(async () => {
      try {
        const s = await api<Session>('/api/sessions', { method: 'POST', body: JSON.stringify({ title: 'Home Chat' }) })
        if (cancelled) return
        setSession(s)
        if (typeof window !== 'undefined' && 'EventSource' in window) {
          es = new EventSource(`/api/sessions/${s.id}/stream`)
          es.addEventListener('status', (ev: MessageEvent) => {
            try {
              const data = JSON.parse(ev.data) as { state: string, command?: string }
              setMessages((arr) => arr.concat([{ id: `${Date.now()}-status`, role: 'system', content: `(${data.state}${data.command ? `: ${data.command}` : ''})`, ts: new Date().toISOString() }]))
            } catch {}
          })
          es.addEventListener('message', (ev: MessageEvent) => {
            try {
              const msg = JSON.parse(ev.data) as SessionMessage
              setMessages((arr) => arr.concat([msg]))
            } catch {}
          })
        }
      } catch (e) {
        // ignore init errors in MVP
        // optionally, we could surface a toast here
      }
    })()
    return () => { cancelled = true; if (es) es.close() }
  }, [])

  async function send(kind: 'polish' | 'ask' | 'code') {
    if (!session) return
    const text = input.trim()
    if (!text) return
    setBusy(true)
    try {
      // Post user message
      const msg = await api<SessionMessage>(`/api/sessions/${session.id}/messages`, {
        method: 'POST',
        body: JSON.stringify({ role: 'user', content: text })
      })
      setMessages((arr) => arr.concat([msg]))
      setInput('')
      // Trigger a command to show activity via SSE
      await api(`/api/sessions/${session.id}/commands`, { method: 'POST', body: JSON.stringify({ name: kind }) })
    } finally {
      setBusy(false)
    }
  }

  function MsgBubble({ m }: { m: SessionMessage }) {
    const align: React.CSSProperties = m.role === 'user' ? { alignSelf: 'flex-end', background: token.colorPrimary, color: token.colorTextLightSolid } : m.role === 'system' ? { alignSelf: 'center', background: token.colorFillTertiary, color: token.colorTextSecondary } : { alignSelf: 'flex-start', background: token.colorFillSecondary }
    const pad = m.role === 'system' ? '2px 8px' : '8px 12px'
    const radius = 12
    return (
      <div style={{ maxWidth: '80%', margin: '4px 0', padding: pad, borderRadius: radius, ...align }}>{m.content}</div>
    )
  }

  return (
    <Flex vertical gap={12} style={{ flex: 1, minHeight: 0, height: '100%' }}>
      <Card size="small" title={<Typography.Text strong>Home</Typography.Text>}>
        <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
          Chat-like dialog. Use buttons to submit: Polish / Ask / Code.
        </Typography.Paragraph>
      </Card>
      <Card size="small" style={{ flex: 1, minHeight: 0 }} bodyStyle={{ height: '100%', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        <div ref={listRef} style={{ flex: 1, overflow: 'auto', display: 'flex', flexDirection: 'column' }}>
          {messages.length === 0 && (
            <Typography.Text type="secondary">Start by typing below and choose an action.</Typography.Text>
          )}
          {messages.map((m) => <MsgBubble key={m.id} m={m} />)}
        </div>
      </Card>
      <Card size="small">
        <Flex gap={8} align="start">
          <Input.TextArea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Type your message..."
            autoSize={{ minRows: 2, maxRows: 4 }}
            style={{ flex: 1 }}
            disabled={!session || busy}
          />
          <Flex vertical gap={8}>
            <Button type="primary" onClick={() => send('polish')} disabled={!session || busy || !input.trim()}>polish</Button>
            <Button onClick={() => send('ask')} disabled={!session || busy || !input.trim()}>ask</Button>
            <Button onClick={() => send('code')} disabled={!session || busy || !input.trim()}>code</Button>
          </Flex>
        </Flex>
      </Card>
    </Flex>
  )
}
