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
- Provider 目录 v2（`~/.codectl/provider.json`）与 JSON Schema
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
codectl webui -o                # 同上（显式子命令）

# 规格（Spec）相关
codectl spec                    # 在浏览器中打开 Spec UI
codectl spec new "<说明>"       # 通过 Codex 生成规范草案，保存到 vibe-docs/spec
codectl check [--json]          # 校验 vibe-docs/spec 下 *.spec.mdx 的 frontmatter

# 配置与 Provider
codectl config                  # 初始化 ~/.codectl（provider/models/mcp）并打印路径
codectl config -w               # 运行交互式配置向导
codectl provider sync           # 创建/规范化 ~/.codectl/provider.json（v2）
codectl provider schema         # 输出 provider.json（v2）的 JSON Schema

# 实用工具
codectl codex [args...]         # 快捷启动 Codex（gpt-5，高推理）
codectl version                 # 打印版本（仅数字，便于脚本）
codectl update                  # 自更新（规划中；占位）
```

说明：默认监听 `127.0.0.1:8787`；使用 `-a` 可修改。`codectl cli` 作为兼容入口，亦会打开 WebUI。

## Roadmap

- [ ] 1. 原型（Prototype）
- [ ] 2. 更好的 Spec TUI
- [ ] 3. 配置向导（MCP/自定义 Provider）

## 配置目录

默认位置：`~/.codectl/`

- `provider.json`（v2）：Provider 目录；可编辑新增 Provider/Model。可用 `codectl provider schema` 查看 Schema。
- `models.json`：如不存在，将由 Provider 目录初始化。
- `mcp.json`：如不存在，`codectl config` 会写入示例。

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

常用 Make 任务：

- `make start` – 热重载开发（需先安装 `github.com/air-verse/air`）
- `make format` – `go fmt ./...`
- `make lint` – `golangci-lint run` 或回退到 `go vet ./...`
- `make test` – 生成 `coverage.out`
- `make docker-test` – 在 Docker 中运行测试（示例：`GO_TEST_FLAGS='-v -run TestLoad'`）

欢迎提交 Issue/PR：建议先在 `vibe-docs/spec/` 中补充或调整规范，再提交实现与文档。

## 免责声明

codectl 旨在帮助你更便捷地安装、检测与配置第三方工具，本项目本身不提供模型推理能力。第三方 CLI/MCP
的功能、稳定性与条款以各自官方为准，请按需阅读并遵循其使用政策。

## 安全与环境变量

- 请勿提交任何密钥。使用环境变量：`OPENAI_API_KEY`、`ANTHROPIC_API_KEY`、`GOOGLE_API_KEY`，以及代理相关变量。
- 应用本地运行；请遵循各 Provider 的使用条款。

## 许可协议

本项目基于 MIT License 开源，详见 `LICENSE` 文件。
