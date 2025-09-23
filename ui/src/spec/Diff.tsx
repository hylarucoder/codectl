import React, { useEffect, useMemo, useState } from 'react'
import { Card, Flex, List, Radio, Typography, theme } from 'antd'
import { api } from '../lib/api'
import type { DiffChangeItem, DiffFileResponse, SplitRow, SplitSide } from '../types'

type Mode = 'all' | 'staged' | 'worktree'

function SplitDiff({ rows }: { rows: SplitRow[] }) {
  const { token } = theme.useToken()
  const cellStyle = (t: SplitSide): React.CSSProperties => {
    switch (t) {
      case 'del': return { background: token.colorErrorBg }
      case 'add': return { background: token.colorSuccessBg }
      case 'meta': return { color: token.colorTextSecondary }
      default: return {}
    }
  }
  const wrap: React.CSSProperties = {
    display: 'grid',
    gridTemplateColumns: '6ch 1fr 6ch 1fr',
    columnGap: 8,
    rowGap: 2,
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace'
  }
  const cellBase: React.CSSProperties = { whiteSpace: 'pre', padding: '0 8px' }
  const lnStyle: React.CSSProperties = {
    textAlign: 'right',
    color: token.colorTextTertiary,
    userSelect: 'none',
    padding: '0 4px'
  }
  return (
    <div style={{ overflow: 'auto' }}>
      <div style={wrap}>
        {rows.map((r, i) => {
          const ln = r.ln ?? ''
          const rn = r.rn ?? ''
          return (
            <React.Fragment key={i}>
              <div style={lnStyle}>{ln}</div>
              <div style={{ ...cellBase, ...cellStyle(r.lt) }}>{r.left}</div>
              <div style={lnStyle}>{rn}</div>
              <div style={{ ...cellBase, ...cellStyle(r.rt) }}>{r.right}</div>
            </React.Fragment>
          )
        })}
      </div>
    </div>
  )
}

function DiffBody({ diff }: { diff: string }) {
  const { token } = theme.useToken()
  const lines = (diff || '').split(/\r?\n/)
  return (
    <pre style={{ whiteSpace: 'pre-wrap', margin: 0, fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace' }}>
      {lines.map((ln, i) => {
        let color = token.colorText
        if (ln.startsWith('+') && !ln.startsWith('+++')) color = token.colorSuccess
        else if (ln.startsWith('-') && !ln.startsWith('---')) color = token.colorError
        else if (ln.startsWith('@@') || ln.startsWith('diff ')) color = token.colorInfo
        return <div key={i} style={{ color }}>{ln}</div>
      })}
    </pre>
  )
}

export default function DiffView() {
  const { token } = theme.useToken()
  const [mode, setMode] = useState<Mode>('all')
  const [items, setItems] = useState<DiffChangeItem[]>([])
  const [selected, setSelected] = useState<DiffChangeItem | null>(null)
  const [diff, setDiff] = useState<string>('')
  const [specOnly, setSpecOnly] = useState<boolean>(false)
  const [view, setView] = useState<'unified' | 'split'>('split')
  const [splitRows, setSplitRows] = useState<SplitRow[]>([])

  async function loadChanges(m: Mode, only: boolean) {
    const q = `/api/diff/changes?mode=${m}&specOnly=${only ? '1' : '0'}`
    const arr = await api<DiffChangeItem[]>(q)
    setItems(arr)
    if (arr.length) {
      openFile(arr[0])
    } else {
      setSelected(null); setDiff('')
    }
  }

  async function openFile(it: DiffChangeItem) {
    setSelected(it)
    const q = `/api/diff/file?path=${encodeURIComponent(it.path)}&mode=${mode}${view==='split' ? '&format=split' : ''}`
    const d = await api<DiffFileResponse>(q)
    setDiff(d.diff || '')
    setSplitRows(d.split || [])
  }

  useEffect(() => { loadChanges(mode, specOnly) }, [mode, specOnly])
  useEffect(() => {
    if (selected && view === 'split') {
      openFile(selected)
    }
  }, [view])

  const grouped = useMemo(() => {
    const m: Record<string, DiffChangeItem[]> = {}
    for (const it of items) {
      const g = it.group || 'Other'
      if (!m[g]) m[g] = []
      m[g].push(it)
    }
    return m
  }, [items])

  return (
    <Flex gap={12} style={{ minHeight: 520 }}>
      <Card size="small" title={
        <Flex align="center" gap={8}>
          <Typography.Text strong>Changes</Typography.Text>
          <Radio.Group size="small" value={mode} onChange={e => setMode(e.target.value)}>
            <Radio.Button value="all">All</Radio.Button>
            <Radio.Button value="staged">Staged</Radio.Button>
            <Radio.Button value="worktree">Worktree</Radio.Button>
          </Radio.Group>
          <Radio checked={specOnly} onChange={e => setSpecOnly(e.target.checked)}>Spec only</Radio>
          <div style={{ flex: 1 }} />
          <Radio.Group size="small" value={view} onChange={e => setView(e.target.value)}>
            <Radio.Button value="split">Side by side</Radio.Button>
            <Radio.Button value="unified">Unified</Radio.Button>
          </Radio.Group>
        </Flex>
      } style={{ width: 420, flex: '0 0 auto' }} bodyStyle={{ maxHeight: 640, overflow: 'auto' }}>
        {Object.keys(grouped).map(group => (
          <div key={group} style={{ marginBottom: 8 }}>
            <Typography.Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>{group}</Typography.Text>
            <List
              size="small"
              dataSource={grouped[group]}
              renderItem={(it) => (
                <List.Item style={{ cursor: 'pointer', background: selected?.path === it.path ? token.colorFillSecondary : undefined }} onClick={() => openFile(it)}>
                  <Typography.Text>{it.path}</Typography.Text>
                  <Typography.Text type="secondary" style={{ marginLeft: 8 }}>{it.status}</Typography.Text>
                </List.Item>
              )}
            />
          </div>
        ))}
      </Card>
      <Card size="small" style={{ flex: 1 }} title={selected ? selected.path : 'Diff'} bodyStyle={{ maxHeight: 640, overflow: 'auto' }}>
        {view === 'split'
          ? (splitRows && splitRows.length ? <SplitDiff rows={splitRows} /> : <Typography.Text type="secondary">No diff</Typography.Text>)
          : (diff ? <DiffBody diff={diff} /> : <Typography.Text type="secondary">No diff</Typography.Text>)}
      </Card>
    </Flex>
  )
}
