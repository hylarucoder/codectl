# CODECTL

<p align="center">
    <img src="https://github.com/user-attachments/assets/effc6bc1-ef96-49cc-8751-6f9d1052e248" width="800"/>
<p>

> SDD is all you need

中文文档。English version: README.md

本地 WebUI 的 SDD 工具，最大化 codex 等编码代理的有效利用率。

## Feature

- Spec Driven Development Workflow (Spec -> Task -> Coding)
- 本地 WebUI（推荐）
- Manage CLI Coding Agent (Codex  / Claude Code / Gemini CLI)
- Manage MCP / 3rd Party Model
- TUI + CLI：既可交互式使用，也可脚本化集成。

## 为什么是 Spec‑Driven Development

- 规格: 在 `vibe-docs/spec/` 定义规范
- 任务: 在 `vibe-docs/task/` 定义任务
- 编码: 通过大模型执行编码

## 快速开始

1) 构建并运行 codectl（默认启动本地 WebUI）：

```bash
# 本地开发运行
go run . -o   # 启动服务并打开浏览器

# 或编译二进制
go build -o codectl
./codectl -o  # 启动服务并打开浏览器
```

## 用法

```bash
codectl                         # 启动内嵌 WebUI（默认）
codectl -a 127.0.0.1:8787 -o    # 自定义地址并自动打开浏览器
 codectl webui -o               # 同上（显式子命令）
# TODO: optimize this
# short cut for codex --dangerously-bypass-approvals-and-sandbox -m gpt-5 -c model_reasoning_effort=high
codectl codex                   # codex + gpt 5 high
# TODO: implement this
codectl update                  # 未来将从 GitHub Releases 自更新
codectl version                 # 打印 codectl 版本（仅数字，便于脚本）
# TODO: maybe better tui
codectl config                  # 初始化并打印配置目录（生成 provider/models/mcp 文件）

codectl spec                    # 打开交互式 Spec UI（选择表格 + 左侧 Markdown + 右侧日志 + 底部输入）
codectl spec new "<说明>"       # 调用 codex exec 生成规范草案，保存到 vibe-docs/spec

codectl check                   # 检测 vibe-docs/spec 下的 .spec.mdx 的 frontmatter（至少含 title）
codectl check --json            # 以 JSON 报告形式输出

codectl provider sync           # 手动同步/生成 ~/.codectl/provider.json（可自定义编辑）
codectl provider schema         # 输出 provider.json 的 JSON Schema（用于校验/补全）
```

说明：安装/卸载/升级/状态等操作均在 TUI 内通过斜杠命令完成：/add、/remove、/upgrade、/status；当前不提供独立的 “codectl cli add/remove/...” 子命令。

## Roadmap

- [ ] 1. 原型（Prototype）
- [ ] 2. 更好的 Spec TUI
- [ ] 3. 配置向导（MCP/自定义 Provider）

## 开发与构建

前置要求：Go 1.25+（推荐最新稳定版）

```bash
# 拉取依赖（首次运行会自动拉取）
go mod download

# 本地运行
go run .

# 构建二进制
go build -o codectl
```

欢迎提交 Issue/PR：建议先在 `vibe-docs/spec/` 中补充或调整规范，再提交实现与文档。

## 免责声明

codectl 旨在帮助你更便捷地安装、检测与配置第三方工具，本项目本身不提供模型推理能力。第三方 CLI/MCP
的功能、稳定性与条款以各自官方为准，请按需阅读并遵循其使用政策。

## 许可协议

本项目基于 MIT License 开源，详见 `LICENSE` 文件。
