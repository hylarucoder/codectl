import React, { useEffect, useMemo, useState } from 'react'
import { api } from '../lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Plus, Save, Trash2 } from 'lucide-react'

type Model = {
  name?: string
  id?: string
  context_window?: number
  default_max_tokens?: number
}
type Provider = {
  name?: string
  base_url?: string
  type?: string
  models?: Model[]
}
type Catalog = Record<string, Provider>

const providerTypeHints: Record<string, { label: string; env?: string; base?: string }> = {
  openai: { label: 'OpenAI (GPT/Coders)', env: 'OPENAI_API_KEY', base: 'https://api.openai.com/v1/' },
  anthropic: { label: 'Anthropic (Claude/Claude Code)', env: 'ANTHROPIC_API_KEY', base: 'https://api.anthropic.com/v1/' },
  google: { label: 'Google (Gemini)', env: 'GOOGLE_API_KEY', base: 'https://generativelanguage.googleapis.com/v1beta/' },
  ollama: { label: 'Ollama (local)', base: 'http://localhost:11434/v1/' },
  glm: { label: 'GLM (Zhipu)', base: '' },
  k2: { label: 'Kimi K2', base: '' },
}

export default function Settings() {
  const [catalog, setCatalog] = useState<Catalog>({})
  const [selected, setSelected] = useState<string>('')
  const [provider, setProvider] = useState<Provider>({})
  const [saved, setSaved] = useState<string>('')

  useEffect(() => {
    ;(async () => {
      const cat = await api<Catalog>('/api/providers')
      setCatalog(cat || {})
      const first = Object.keys(cat || {})[0]
      setSelected(first || '')
      if (first) setProvider(cat[first] || {})
    })()
  }, [])

  useEffect(() => {
    if (!selected) return
    setProvider(catalog[selected] || {})
  }, [selected, catalog])

  const providers = useMemo(() => Object.keys(catalog), [catalog])

  function addProvider() {
    let i = 1
    let key = 'new'
    while (catalog[key]) { key = `new${i++}` }
    const next: Catalog = { ...catalog, [key]: { name: 'New Provider', type: 'openai', base_url: providerTypeHints.openai.base, models: [] } }
    setCatalog(next)
    setSelected(key)
    setProvider(next[key])
  }

  function removeProvider(k: string) {
    const next: Catalog = { ...catalog }
    delete next[k]
    setCatalog(next)
    if (selected === k) {
      const first = Object.keys(next)[0] || ''
      setSelected(first)
      setProvider(first ? next[first] || {} : {})
    }
  }

  function renameProvider(newKey: string) {
    const k = selected
    if (!k || !newKey || newKey === k) return
    if (catalog[newKey]) { alert('Name already exists'); return }
    const next: Catalog = { ...catalog }
    next[newKey] = next[k]
    delete next[k]
    setCatalog(next)
    setSelected(newKey)
  }

  function onChangeType(t?: string) {
    const hint = t && providerTypeHints[t]
    if (hint && hint.base && !provider.base_url) {
      setProvider(prev => ({ ...prev, base_url: hint.base }))
    }
  }

  function addModel() {
    setProvider(prev => ({ ...prev, models: [ ...(prev.models || []), { name: '', id: '' } ] }))
  }

  function removeModel(idx: number) {
    setProvider(prev => ({ ...prev, models: (prev.models || []).filter((_, i) => i !== idx) }))
  }

  async function save() {
    if (!selected) return
    const next: Catalog = { ...catalog, [selected]: provider }
    setCatalog(next)
    await api('/api/providers', { method: 'PUT', body: JSON.stringify(next) })
    setSaved('Saved')
    setTimeout(() => setSaved(''), 1500)
  }

  const hint = provider.type ? providerTypeHints[provider.type] : undefined

  return (
    <div className="flex gap-3 flex-1 min-h-0 h-full">
      {/* Left panel */}
      <div className="w-[300px] flex-none h-full border rounded-xl bg-background">
        <div className="flex items-center justify-between p-3">
          <div className="font-medium">Providers</div>
          <Button size="sm" variant="ghost" onClick={addProvider}>
            <Plus className="h-4 w-4" />
          </Button>
        </div>
        <Separator />
        <ScrollArea className="h-[calc(100%-48px)]">
          <div className="p-2">
            {providers.map((k) => (
              <div key={k} className={`flex items-center rounded-md px-2 py-2 mb-1 cursor-pointer ${selected === k ? 'bg-accent' : 'hover:bg-accent/50'}`}
                   onClick={() => setSelected(k)}>
                <div className="flex-1 min-w-0">
                  <div className="truncate font-medium text-sm">{k}</div>
                  <div className="truncate text-muted-foreground text-xs">{catalog[k]?.name}</div>
                </div>
                <Button size="icon" variant="ghost" onClick={(e) => { e.stopPropagation(); removeProvider(k) }}>
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            ))}
          </div>
        </ScrollArea>
      </div>

      {/* Right panel */}
      <div className="flex-1 min-w-0 h-full border rounded-xl bg-background flex flex-col">
        <div className="flex items-center justify-between p-3">
          <div className="font-medium">{selected ? `Edit: ${selected}` : 'Select a provider'}</div>
          {selected && (
            <div className="flex items-center gap-2">
              <Input placeholder="rename key" className="h-8" onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  renameProvider((e.target as HTMLInputElement).value.trim())
                }
              }} />
              <Button onClick={save}>
                <Save className="h-4 w-4 mr-1" /> Save
              </Button>
            </div>
          )}
        </div>
        <Separator />
        <ScrollArea className="flex-1">
          {selected && (
            <div className="p-4 space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Display Name</label>
                <Input placeholder="e.g., Anthropic US" value={provider.name || ''} onChange={(e) => setProvider(prev => ({ ...prev, name: e.target.value }))} />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Type</label>
                <Select value={provider.type} onValueChange={(v) => { setProvider(prev => ({ ...prev, type: v })); onChangeType(v) }}>
                  <SelectTrigger className="w-[280px]">
                    <SelectValue placeholder="Select provider type" />
                  </SelectTrigger>
                  <SelectContent>
                    {Object.keys(providerTypeHints).map(k => (
                      <SelectItem key={k} value={k}>{providerTypeHints[k].label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Base URL</label>
                <Input placeholder="https://.../v1/" value={provider.base_url || ''} onChange={(e) => setProvider(prev => ({ ...prev, base_url: e.target.value }))} />
              </div>
              {hint && (
                <div className="text-sm text-muted-foreground -mt-2">
                  {hint.env ? (
                    <span>Set env var <Badge variant="outline">{hint.env}</Badge> for auth.</span>
                  ) : 'No API key required by default.'}
                </div>
              )}

              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <label className="text-sm font-medium">Models</label>
                  <Button variant="ghost" size="sm" onClick={addModel}><Plus className="h-4 w-4" /></Button>
                </div>
                <div className="rounded-md border divide-y">
                  <div className="grid grid-cols-[1fr_1fr_100px_100px_40px] gap-2 px-3 py-2 text-xs text-muted-foreground">
                    <div>Name</div>
                    <div>ID</div>
                    <div>Ctx</div>
                    <div>MaxT</div>
                    <div></div>
                  </div>
                  {(provider.models || []).map((m, idx) => (
                    <div key={idx} className="grid grid-cols-[1fr_1fr_100px_100px_40px] gap-2 px-3 py-2 items-center">
                      <Input placeholder="friendly name" value={m.name || ''} onChange={(e) => setProvider(prev => {
                        const list = [...(prev.models || [])]
                        list[idx] = { ...list[idx], name: e.target.value }
                        return { ...prev, models: list }
                      })} />
                      <Input placeholder="model id (e.g., claude-3-5-sonnet-latest)" value={m.id || ''} onChange={(e) => setProvider(prev => {
                        const list = [...(prev.models || [])]
                        list[idx] = { ...list[idx], id: e.target.value }
                        return { ...prev, models: list }
                      })} />
                      <Input type="number" placeholder="e.g. 200000" value={m.context_window ?? ''} onChange={(e) => setProvider(prev => {
                        const list = [...(prev.models || [])]
                        list[idx] = { ...list[idx], context_window: Number(e.target.value || 0) }
                        return { ...prev, models: list }
                      })} />
                      <Input type="number" placeholder="e.g. 8192" value={m.default_max_tokens ?? ''} onChange={(e) => setProvider(prev => {
                        const list = [...(prev.models || [])]
                        list[idx] = { ...list[idx], default_max_tokens: Number(e.target.value || 0) }
                        return { ...prev, models: list }
                      })} />
                      <Button variant="ghost" size="icon" onClick={() => removeModel(idx)}>
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  ))}
                </div>
              </div>

              {saved && <div className="text-xs text-muted-foreground">{saved}</div>}
            </div>
          )}
        </ScrollArea>
      </div>
    </div>
  )
}
