# codectl

面向开发者的一站式 Code Agent 配置助手。帮助你更快地安装、检测、配置与切换主流代码智能体工具（如 Codex、Claude Code、Gemini），统一 API Key、代理、预设提示词与多配置档，减少在不同生态之间来回折腾的时间。

## 核心价值

- 快速上手：一步步引导安装与配置常见 Code Agent CLI。
- 统一管理：集中管理 `OPENAI_API_KEY`、`ANTHROPIC_API_KEY`、`GOOGLE_API_KEY` 等环境变量与代理设置。
- 配置档切换：为不同项目/场景维护多套配置，快速切换默认提供商与参数。
- 健康检查：检测 CLI 是否已安装、版本是否匹配、密钥与网络是否可用。
- 可扩展：面向更多提供商与编辑器/工具链做集成（Roadmap）。

## 支持的工具（当前聚焦）

- Codex CLI：`@openai/codex`
- Claude Code：`@anthropic-ai/claude-code`
- Gemini CLI：`@google/gemini-cli`

> 以上均为官方/社区提供的 CLI 工具，codectl 旨在帮助你对其进行统一配置与环境检测。

## 快速开始

1) 安装常用 Code Agent CLI（可按需选择）：

```bash
# Codex
npm install -g @openai/codex

# Claude Code
npm install -g @anthropic-ai/claude-code

# Gemini CLI
npm install -g @google/gemini-cli
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

3) 运行 codectl（内置版本检测 TUI）：

```bash
# 本地开发运行
go run .

# 或编译二进制
go build -o codectl
./codectl
```

提示：启动后会展示 Codex / Claude / Gemini 的安装与版本信息：
- r 重新检测，u 升级到最新，q 退出。
后续版本将补充完整的配置向导与更全面的健康检查。

## 典型场景

- 为新机器一键完成 Agent CLI 安装与 Key 配置。
- 切换默认提供商（如从 OpenAI 切到 Anthropic）而不影响项目脚手架。
- 针对不同项目使用不同模型/上下文/提示词模板。
- 统一设置代理，便于 CLI 与编辑器扩展共用网络配置。

## Roadmap

- 0.1 原型：
  - TUI 框架搭建（Bubble Tea），基础交互与退出逻辑。
- 0.2 健康检查：
  - 自动检测 Codex/Claude/Gemini CLI 是否已安装与版本信息。
  - 检查 API Key、代理连通性并提供修复建议。
- 0.3 配置向导：
  - 引导写入/更新环境变量与常见 CLI 的配置文件。
  - 支持创建与切换配置档（profiles）。
- 0.4 生态集成：
  - 输出可被编辑器/终端工具读取的统一配置（如 `~/.config/codectl/config.yaml`）。
  - 常见编辑器/终端插件的配置提示与自动化脚本。
- 0.5 可扩展提供商：
  - 支持更多 LLM/代码智能体与企业代理/私有化部署。

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

项目采用 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 构建 TUI。欢迎提交 Issue/PR 讨论功能设计与实现。

## 常见链接

- OpenAI Key 申请：https://platform.openai.com/ （账号与付费要求以官方为准）
- Anthropic Key 申请：https://console.anthropic.com/
- Google AI Studio（Gemini Key）：https://aistudio.google.com/

## 免责声明

codectl 仅帮助你更便捷地安装与配置第三方工具，本项目本身不提供模型推理能力。第三方 CLI 的功能、稳定性与条款以各自官方为准，请按需阅读并遵循其使用政策。

## 许可协议

本项目基于 MIT License 开源，详见 `LICENSE` 文件。
