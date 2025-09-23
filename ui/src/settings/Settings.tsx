import React, { useEffect, useMemo, useState } from 'react'
import { Button, Card, Flex, Form, Input, InputNumber, List, Select, Space, Table, Tag, Typography, message } from 'antd'
import { PlusOutlined, DeleteOutlined, SaveOutlined } from '@ant-design/icons'
import { api } from '../lib/api'

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
  const [form] = Form.useForm<Provider>()
  const [msgApi, ctx] = message.useMessage()

  useEffect(() => {
    ;(async () => {
      const cat = await api<Catalog>('/api/providers')
      setCatalog(cat || {})
      const first = Object.keys(cat || {})[0]
      setSelected(first || '')
    })()
  }, [])

  useEffect(() => {
    if (!selected) return
    form.setFieldsValue(catalog[selected] || {})
  }, [selected, catalog])

  const providers = useMemo(() => Object.keys(catalog), [catalog])

  function addProvider() {
    let i = 1
    let key = 'new'
    while (catalog[key]) { key = `new${i++}` }
    const next: Catalog = { ...catalog, [key]: { name: 'New Provider', type: 'openai', base_url: providerTypeHints.openai.base, models: [] } }
    setCatalog(next)
    setSelected(key)
  }

  function removeProvider(k: string) {
    const next: Catalog = { ...catalog }
    delete next[k]
    setCatalog(next)
    if (selected === k) setSelected(Object.keys(next)[0] || '')
  }

  function renameProvider(newKey: string) {
    const k = selected
    if (!k || !newKey || newKey === k) return
    if (catalog[newKey]) { msgApi.error('Name already exists'); return }
    const next: Catalog = { ...catalog }
    next[newKey] = next[k]
    delete next[k]
    setCatalog(next)
    setSelected(newKey)
  }

  function onChangeType(t?: string) {
    const hint = t && providerTypeHints[t]
    if (hint && hint.base && !form.getFieldValue('base_url')) {
      form.setFieldValue('base_url', hint.base)
    }
  }

  function addModel() {
    const p = form.getFieldsValue()
    const m: Model = { name: '', id: '' }
    form.setFieldValue('models', [ ...(p.models || []), m ])
  }

  function removeModel(idx: number) {
    const p = form.getFieldsValue()
    const arr = [ ...(p.models || []) ]
    arr.splice(idx, 1)
    form.setFieldValue('models', arr)
  }

  async function save() {
    if (!selected) return
    const p = await form.validateFields()
    const next: Catalog = { ...catalog, [selected]: p }
    setCatalog(next)
    await api('/api/providers', { method: 'PUT', body: JSON.stringify(next) })
    msgApi.success('Saved')
  }

  const selectedType = Form.useWatch('type', form)
  const hint = selectedType ? providerTypeHints[selectedType] : undefined

  return (
    <Flex gap={12} className="flex-1 min-h-0 h-full">
      {ctx}
      <Card size="small" title={
        <Flex align="center" justify="space-between">
          <Typography.Text strong>Providers</Typography.Text>
          <Button size="small" icon={<PlusOutlined />} onClick={addProvider}>Add</Button>
        </Flex>
      } className="w-[320px] flex-none h-full" bodyStyle={{ height: '100%', overflow: 'auto' }}>
        <List
          size="small"
          dataSource={providers}
          renderItem={(k) => (
            <List.Item
              className="cursor-pointer"
              style={{ background: selected === k ? 'rgba(0,0,0,0.06)' : undefined }}
              onClick={() => setSelected(k)}
              actions={[<Button size="small" key="del" danger type="text" icon={<DeleteOutlined />} onClick={(e) => { e.stopPropagation(); removeProvider(k) }} />]}
            >
              <List.Item.Meta title={<span>{k}</span>} description={catalog[k]?.name} />
            </List.Item>
          )}
        />
      </Card>
      <Card size="small" className="flex-1 min-w-0 h-full" bodyStyle={{ height: '100%', overflow: 'auto' }} title={selected ? `Edit: ${selected}` : 'Select a provider'}
        extra={selected && <Space>
          <Input size="small" placeholder="rename key" onPressEnter={(e) => renameProvider((e.target as HTMLInputElement).value.trim())} />
          <Button type="primary" icon={<SaveOutlined />} onClick={save}>Save</Button>
        </Space>}
      >
        {selected && (
          <Form form={form} layout="vertical" initialValues={catalog[selected] || {}}>
            <Form.Item name="name" label="Display Name">
              <Input placeholder="e.g., Anthropic US" />
            </Form.Item>
            <Form.Item name="type" label="Type">
              <Select
                placeholder="Select provider type"
                options={Object.keys(providerTypeHints).map(k => ({ value: k, label: providerTypeHints[k].label }))}
                onChange={onChangeType}
              />
            </Form.Item>
            <Form.Item name="base_url" label="Base URL">
              <Input placeholder="https://.../v1/" />
            </Form.Item>
            {hint && (
              <Typography.Paragraph type="secondary" className="mt-[-8px]">
                {hint.env ? (<span>Set env var <Tag>{hint.env}</Tag> for auth.</span>) : 'No API key required by default.'}
              </Typography.Paragraph>
            )}
            <Form.Item label="Models" shouldUpdate>
              <Space direction="vertical" className="w-full">
                <Button size="small" icon={<PlusOutlined />} onClick={addModel}>Add Model</Button>
                <Form.List name="models">
                  {(fields) => (
                    <Table
                      size="small"
                      pagination={false}
                      dataSource={fields}
                      rowKey={(f) => String(f.key)}
                      columns={[
                        { title: 'Name', render: (_, f) => (<Form.Item name={[f.name, 'name']} className="m-0"><Input placeholder="friendly name" /></Form.Item>) },
                        { title: 'ID', render: (_, f) => (<Form.Item name={[f.name, 'id']} className="m-0"><Input placeholder="model id (e.g., claude-3-5-sonnet-latest)" /></Form.Item>) },
                        { title: 'Ctx', width: 100, render: (_, f) => (<Form.Item name={[f.name, 'context_window']} className="m-0"><InputNumber min={0} placeholder="e.g. 200k" className="w-full" /></Form.Item>) },
                        { title: 'MaxT', width: 100, render: (_, f) => (<Form.Item name={[f.name, 'default_max_tokens']} className="m-0"><InputNumber min={0} placeholder="e.g. 8192" className="w-full" /></Form.Item>) },
                        { title: '', width: 48, render: (_, f, idx) => (<Button danger type="text" icon={<DeleteOutlined />} onClick={() => removeModel(idx)} />) },
                      ]}
                    />
                  )}
                </Form.List>
              </Space>
            </Form.Item>
          </Form>
        )}
      </Card>
    </Flex>
  )
}
