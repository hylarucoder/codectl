# codectl

> Maybe Spec Driven Development is all your need

聚焦于 Spec‑Driven Development（规范驱动开发，SDD）。

在此基础上，额外提供对 Coding Agent、LLM 供应商与 MCP的安装、检测、配置与切换能力，尽量把"配置/对齐/验证"这类重复性工作沉到工具里。

Note：Coding Agent 迭代非常迅速，codectl 可能会被更强大的模型能力所取代。

## 为什么是 Spec‑Driven Development

- 规范优先：功能从 `vibe-docs/spec/` 中的规范开始定义，再落地到 CLI/TUI 与实现代码，保证讨论与实现对齐。
- 可验证：通过 `codectl spec` 打开交互式界面，浏览规范与记录对话日志。
- 易协作：配合 `vibe-docs/AGENTS.md`，为人类与 AI Coding Agent 提供一致的协作约束与上下文。

## 功能总览

- CLI 管理：安装、卸载、检测与升级
- 统一环境与供应商管理
- MCP：规划管理 MCP 客户端/服务端的基本配置，便于在工具生态间复用上下文与凭据。
- TUI + CLI：既可交互式使用，也可脚本化集成。

## 支持的工具（当前聚焦）

- Codex CLI：`@openai/codex`
- Claude Code：`@anthropic-ai/claude-code`
- Gemini CLI：`@google/gemini-cli`

> 以上均为官方/社区提供的 CLI。codectl 统一做安装/配置/检测，不替代其原生能力。

## 快速开始

1) 构建并运行 codectl：

```bash
# 本地开发运行
go run .

# 或编译二进制
go build -o codectl
./codectl
```

2) 配置 API Key（示例为 macOS/Linux）：

```bash
export OPENAI_API_KEY=your_openai_key
export ANTHROPIC_API_KEY=your_anthropic_key
export GOOGLE_API_KEY=your_google_key

# 如需代理（可选）
export HTTP_PROXY=http://127.0.0.1:7890
export HTTPS_PROXY=http://127.0.0.1:7890
```

## 用法

```bash
# CLI 工具管理（TUI）
codectl cli                     # 打开 CLI 管理 TUI（支持 /add、/remove、/upgrade 等）

# 自更新（占位，暂未实现自动下载）
codectl update                  # 未来将从 GitHub Releases 自更新
 
# 查看版本与配置位置
codectl version                 # 打印 codectl 版本（仅数字，便于脚本）
codectl config                  # 初始化并打印配置目录（生成 provider/models/mcp 文件）

# 规范（Spec）
codectl spec                    # 打开交互式 Spec UI（选择表格 + 左侧 Markdown + 右侧日志 + 底部输入）
codectl spec new "<说明>"       # 调用 codex exec 生成规范草案，保存到 vibe-docs/spec

# 校验文档 frontmatter（MDX）
codectl check                   # 检测 vibe-docs/spec 下的 .spec.mdx 的 frontmatter（至少含 title）
codectl check --json            # 以 JSON 报告形式输出

# 模型管理（新增）
codectl model ls                # 列出本地模型清单
codectl model ls-remote         # 列出远端可用模型清单（占位）
codectl model add kimi-k2-0905-preview kimi-k2-0711-preview
codectl model remove kimi-k2-0905-preview

# 工具与 MCP 清单
# 已集成到 TUI 的状态面板中
codectl mcp ls                  # 列出本地 MCP 服务端
codectl mcp ls-remote           # 列出远端可用 MCP 服务端（占位）
// 远端最新版本展示亦可通过 TUI 升级检查查看

# 远端清单来源（provider.json, v2）
# ls-remote 会优先从 ~/.codectl/provider.json 读取“providers 映射”并扁平化 models：
#
# {
#   "ollama": {
#     "name": "Ollama",
#     "base_url": "http://localhost:11434/v1/",
#     "type": "openai",
#     "models": [{"name": "Qwen 3 30B", "id": "qwen3:30b"}]
#   }
# }
#
# 如该文件不存在，将使用内置默认骨架作为回退。
codectl provider sync           # 手动同步/生成 ~/.codectl/provider.json（可自定义编辑）
codectl provider schema        # 输出 provider.json 的 JSON Schema（用于校验/补全）

# MCP 配置（mcp.json）
# MCP 独立保存在 ~/.codectl/mcp.json；仅支持“名称 → 配置”映射结构（不再兼容旧的数组格式）：
# {
#   "Framelink Figma MCP": {
#     "command": "npx",
#     "args": ["-y", "figma-developer-mcp", "--figma-api-key=YOUR-KEY", "--stdio"]
#   }
# }

# Demo（Bubble Tea 示例）
codectl demo autocomplete       # 运行自动补全示例（按 Tab 补全，Esc/Enter 退出）
codectl demo chat               # 运行聊天消息示例（Viewport + Textarea）
codectl demo markdown [file]    # 使用 Glamour 渲染 Markdown（可选文件路径）

# 开发快捷方式（仅 main.go）
# 通过环境变量在 `go run main.go` 下直接启动 Chat Demo：
#
#   CODECTL_DEMO=chat go run main.go
#
# 不设置时 `go run main.go` 将按默认行为启动 CLI。
```

支持的工具参数（可多选）：`all`、`codex`、`claude`、`gemini`。
也支持常见别名：

- Codex: `codex`、`openai`、`openai-codex`
- Claude: `claude`、`claude-code`、`anthropic`
- Gemini: `gemini`、`google`

注意：安装/升级依赖系统已安装 `node`/`npm` 并可访问 npm registry。`codectl config` 仅初始化配置文件；实际 CLI 安装/升级请在 TUI 中执行或使用斜杠命令（如 `/add all`、`/upgrade`）。

## TUI 使用说明

- 启动后顶部展示当前目录与状态面板，列出每个工具：是否已安装、当前版本、npm 最新版本，以及检测来源。
- 当存在可更新版本时，最新版本将高亮提示；使用 `/upgrade` 可批量升级并自动回查结果。
- 输入框支持斜杠命令：输入 `/` 进入命令模式，使用 ↑/↓ 选择、`Tab` 补全、`Enter` 执行、`Esc` 退出。
- 已实现命令：
    - `/doctor`：重新检测并给出状态提示
    - `/status`：在界面底部汇总一行当前状态
    - `/add`：安装受支持的 CLI（如 `/add all`、`/add claude`）
    - `/remove`：卸载受支持的 CLI（如 `/remove gemini`）
    - `/upgrade`（`/update`）：批量升级受支持的 CLI
    - `/task`：生成 `vibe-docs/task/YYMMDD-HHMMSS-<slug>.task.mdx`（可用 `/task <标题>` 指定标题，自动生成 slug 与时间戳）
    - `/spec`：调用 `codex exec <说明>` 生成规范草案，保存到 `vibe-docs/spec/draft-YYMMDD-HHMMSS-<slug>.spec.mdx`
    - `/codex`：在 TUI 中直接运行 `codex`（可附加参数，如 `/codex exec "生成一个README模板"`）
    - `/exit`（`/quit`）：退出界面
    - `/init`：在当前 Git 仓库根目录创建 `vibe-docs/AGENTS.md` 模板文件

提示：界面底部状态栏会显示当前时间、`codectl` 版本，以及在 Git 仓库中的分支/提交信息。

## 典型场景

- 为新机器一键完成 Agent CLI 安装与 Key 配置。
- 切换默认提供商（如从 OpenAI 切到 Anthropic）而不影响项目脚手架。
- 针对不同项目使用不同模型/上下文/提示词模板。
- 统一设置代理，便于 CLI 与编辑器扩展共用网络配置。
- 在团队中以规范驱动对齐预期，再落地实现与测试。

## SDD 工作流与文档约定

- 规范位置：`vibe-docs/spec/`（例如：`vibe-docs/spec/overall.md`）。建议先提 PR 更新规范，再按规范开发。
- Agent 协作：`vibe-docs/AGENTS.md` 用于指导 AI Coding Agent 在仓库中的约束、风格与运行方式。
- 规范浏览 UI：`codectl spec` 打开交互式界面浏览/记录规范上下文。
- 贡献建议：任何新功能或破坏性变更，先在规范中给出目标、边界、接口与验证方式。

## Roadmap

- 0.1 原型：
    - TUI 框架搭建（Bubble Tea），基础交互与退出逻辑。
- 0.2 健康检查：
    - 自动检测 Codex/Claude/Gemini CLI 是否已安装与版本信息。
    - 检查 API Key、代理连通性并提供修复建议。
- 0.3 配置向导：
    - 引导写入/更新环境变量与常见 CLI 的配置文件。
    - 支持创建与切换配置档（profiles）。
    - 0.4 生态与 MCP：
    - 输出统一配置（如 `~/.codectl/config.json`），并规划 MCP 客户端/服务端的基础管理能力（配置与健康检查）。
    - 常见编辑器/终端插件的配置提示与自动化脚本。
- 0.5 可扩展提供商：
    - 支持更多 LLM/代码智能体与企业代理/私有化部署。
    - Provider/MCP 插件机制（提案与 PoC）。

## 开发与构建

前置要求：Go 1.20+（推荐最新稳定版）

```bash
# 拉取依赖（首次运行会自动拉取）
go mod download

# 本地运行
go run .

# 构建二进制
go build -o codectl
```

项目采用 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 构建 TUI。欢迎提交 Issue/PR：建议先在 `vibe-docs/spec/`
中补充或调整规范，再提交实现与文档。

## 兼容性与取舍

- Agent 发展很快：当更强的模型能力能直接完成这类“安装/检测/配置”任务时，codectl 可能被替代或收缩为更薄的一层脚本/规范集。
- 设计目标是“易替换”：确保规范清晰、功能内聚，便于裁撤/迁移而不影响使用者的项目。

## 常见链接

- OpenAI Key 申请：https://platform.openai.com/ （账号与付费要求以官方为准）
- Anthropic Key 申请：https://console.anthropic.com/
- Google AI Studio（Gemini Key）：https://aistudio.google.com/

## 免责声明

codectl 旨在帮助你更便捷地安装、检测与配置第三方工具，本项目本身不提供模型推理能力。第三方 CLI/MCP
的功能、稳定性与条款以各自官方为准，请按需阅读并遵循其使用政策。

## 许可协议

本项目基于 MIT License 开源，详见 `LICENSE` 文件。
