# Vidify 架构概览

## 1. 项目定位

Vidify 是一个基于 Go 1.23+ 的音频转视频工具，结合 Whisper.cpp 的本地语音识别能力与 FFmpeg 的视频编解码能力，将音频文件快速转换为带同步字幕的视频文件。系统同时提供命令行体验与可扩展的服务端入口，强调稳定、易部署与高可维护性。

## 2. 核心特性

- 🎯 高精度语音识别：通过 Whisper.cpp 支持多语言离线识别，保证字幕准确度
- 🎬 智能视频生成：基于 FFmpeg 自动生成带字幕的视频，支持自定义样式
- 🚀 高并发处理：流水线设计允许批量任务并发执行
- 🔧 双模式支持：CLI 与 Web API 入口复用同一处理核心
- ⚙️ 灵活配置：通过配置文件与运行时参数调整字幕样式、线程数等
- 📦 模块化设计：内部包划分清晰，便于扩展与维护

## 3. 技术栈与外部依赖

| 组件 | 选型 | 说明 |
|--------------|----------------|------|
| 编程语言 | Go 1.23+ | 主体服务与 CLI 逻辑 |
| 语音识别 | Whisper.cpp | 离线、多语言语音识别 |
| 视频处理 | FFmpeg 8.0+ | 字幕烧录与音视频合成 |
| CLI 框架 | Cobra | 子命令与参数解析 |
| 配置管理 | Viper | 统一加载配置与环境变量 |
| 日志框架 | Zap | 结构化日志输出 |

## 4. 模块结构

```
vidify/
├── cmd/
│   ├── cli/                # Cobra CLI 子命令（convert、batch、version）
│   ├── server/             # Web 处理器骨架
│   └── vidify/main.go      # CLI 二进制入口
├── internal/
│   ├── audio/              # 音频格式检测与预处理
│   ├── config/             # 配置模型与加载逻辑
│   ├── ffmpeg/             # FFmpeg 命令封装
│   ├── logging/            # 应用级日志初始化
│   ├── pipeline/           # 转录与渲染编排
│   ├── storage/            # 输出与临时文件管理
│   ├── subtitle/           # 字幕格式与样式
│   ├── tools/transcriber/  # Whisper CLI 调用封装
│   ├── video/              # 视频渲染抽象
│   └── whisper/            # Whisper 模型与任务封装
├── pkg/
│   ├── errors/             # 错误包装与分类
│   └── utils/              # 通用工具函数
├── configs/                # 默认配置模板（可自定义）
├── docs/                   # 架构、流程、任务文档
├── test/                   # 测试骨架与预留用例
└── third_party/whisper/    # 预置 whisper-cli 及模型占位
```

### 4.1 模块职责

- **cmd**：提供用户入口。`cmd/cli` 负责 Cobra 子命令，`cmd/vidify/main.go` 将内部流水线与 CLI 参数对接，`cmd/server` 为未来的 Web API 扩展预留结构。
- **internal/audio & whisper**：准备音频输入并调用 Whisper CLI 生成 `.srt` 字幕；支持线程与模型配置。
- **internal/pipeline**：串联“音频准备 → 语音识别 → 视频渲染”，并协调临时目录与任务状态。
- **internal/subtitle & video & ffmpeg**：描述字幕样式、构造 FFmpeg 命令并封装渲染输出。
- **internal/config & logging & storage**：集中管理配置、日志以及输出目录，保证行为可配置且可观测。
- **pkg/**：对外暴露的轻量辅助库（错误、工具），避免重复实现。

## 5. 处理流程

### 5.1 CLI 模式

1. 用户通过 `vidify convert -i input.mp3 -o output.mp4` 触发任务，Cobra 解析参数与配置覆盖。
2. `internal/pipeline` 创建临时目录，调用 `internal/tools/transcriber` 执行 Whisper CLI，生成同步字幕。
3. `internal/video` 组合音频、字幕与背景参数，借助 `internal/ffmpeg` 拼装命令渲染 MP4。
4. 结果写入输出目录，同时输出日志与 CLI 提示；中间文件按策略清理。

### 5.2 Web API 模式（规划中）

- `cmd/server` 将提供 Gin 入口，复用同一 `pipeline`。
- 上传的音频暂存到 `storage` 模块管理的路径，返回任务 ID。
- 转码完成后返回视频下载地址或触发后续通知。

## 6. 配置与运行

- 系统要求：Go 1.23+、FFmpeg 8.0+（需在 `PATH` 中）、Whisper.cpp 可执行文件及模型。
- 开发者可通过 `go mod tidy` 同步依赖，`make build` 或 `go run cmd/vidify/main.go` 快速体验。
- `configs/` 提供默认模板，运行时可用环境变量或 CLI 参数覆盖，如字幕样式、线程数、输出目录等。
- 日志默认输出到标准输出，兼容 Zap 的 `development` 模式，便于调试。

## 7. 演进方向

- 完成基于 FFmpeg 的渲染服务，实现更多样式与背景配置。
- 补齐端到端集成测试，覆盖样例音频与模型下载。
- 暴露 Whisper 线程与性能调优选项，匹配 README 中的配置能力。
- 启用 Web API 与任务查询接口，实现 CLI/HTTP 统一流水线。
- 扩展字幕格式支持（SRT/VTT）与云存储输出，支撑 README 中的路线图。

---

该架构文档以项目 README 为基础，聚焦核心能力与模块职责，同时保持 Vidify“音频转视频自动化”的主旨不变。
