<div align="center">

# 🎬 Vidify

**Audio to Video with Subtitles**

将音频一键转化为带字幕的 MP4 视频 — 全部本地运行，保护隐私

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20|%20macOS%20|%20Linux-blue)]()

</div>

---

## 简介

**Vidify** 是一个基于 **Whisper.cpp + FFmpeg** 的本地音频转视频工具。它能自动为音频生成精准的字幕，并合成为可直接发布到 YouTube、Bilibili、抖音等平台的 MP4 视频。

- **CLI 一行命令**：`vidify -i podcast.mp3 -m medium`
- **完全本地**：不上传任何数据，隐私安全
- **高质量字幕**：基于 Whisper.cpp，支持多语言、精准时间戳
- **跨平台**：Windows / macOS / Linux

## 功能特性

### Phase 1: CLI 工具（优先实现）

- 支持音频格式：MP3、WAV、M4A、OGG、FLAC
- 单文件与文件夹批量处理
- Whisper.cpp 语音识别生成 SRT 字幕
  - 模型选择：tiny / base / small / medium / large
  - 语言：自动检测或手动指定
  - 可选翻译模式
- FFmpeg 合成视频（背景图 + 音频 + 硬编码字幕）
- 自定义输出路径和文件名

### Phase 2: 桌面 GUI（Wails + Svelte）

- 拖拽文件、实时进度、日志面板
- 字幕样式自定义（颜色、字号、位置）
- 视频预览、配置保存与加载

## 快速开始

### 前置依赖

**发布包（推荐）：**
- `whisper-cli` 和 `ffmpeg` **已内置**于发布包 `runtime-tools/` 目录，无需额外安装
- Whisper 模型文件**不随发布包分发**（最小几十 MB，最大数 GB；且含多语言变体），需单独下载：

  ```bash
  # 手动下载（替换 medium 为所需大小）
  # 保存至 ~/.vidify/models/ggml-medium.bin
  curl -L https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium.bin \
       -o ~/.vidify/models/ggml-medium.bin
  ```

  | 模型 | 大小 | 适用场景 |
  |------|------|----------|
  | tiny | ~75 MB | 快速测试 |
  | base | ~142 MB | 日常使用 |
  | small | ~466 MB | 均衡 |
  | medium | ~1.5 GB | 推荐 |
  | large | ~3 GB | 最高精度，需 8 GB+ RAM |

**从源码构建（开发环境）：**
- Go 1.26+
- 系统已安装 `ffmpeg`（或通过配置 `ffmpeg-bin` 指定路径）
- 模型文件同上

### 安装

```bash
# 从源码构建
git clone https://github.com/FeiNiaoBF/vidify.git
cd vidify
make build
```

### 使用

```bash
# 最简用法
vidify -i input.mp3

# 指定模型和输出目录
vidify -i input.mp3 -m medium -o ./output

# 指定语言 + 翻译为中文
vidify -i input.mp3 -l en --translate
```

### CLI 参数

| 参数 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| `--input` | `-i` | 输入音频文件路径 | （必填） |
| `--model` | `-m` | Whisper 模型 | `medium` |
| `--language` | `-l` | 语言 | `auto` |
| `--translate` | | 翻译为中文 | `false` |
| `--bg-image` | | 背景图片路径 | 内置默认 |
| `--output` | `-o` | 输出路径 | 当前目录 |

### 二进制解析顺序

`vidify` 按以下优先级定位 `whisper-cli` 和 `ffmpeg`：

1. 配置文件 / 环境变量 / CLI 标志显式指定（最高优先级）
   `whisper-bin: /path/to/whisper-cli`
2. 发布包内置：可执行文件同级目录下 `runtime-tools/<tool>/<os>-<arch>/`
3. 系统 `PATH`（回退）

### 任务缓存

每次处理任务都会在 `~/.vidify/cache/tasks/<task-id>/` 下创建隔离工作目录，用于存放中间产物（SRT 草稿、FFmpeg 滤镜脚本等）。

- **任务成功**：缓存目录自动删除
- **任务失败**：缓存目录保留，日志中可查看路径，便于排查问题

## 项目结构

```
vidify/
├── cmd/                    # 入口（cli / gui）
├── internal/               # 内部业务逻辑
│   ├── domain/             # 领域模型 (Audio / Subtitle / Video / Config)
│   ├── usecase/            # 用例层 (Transcribe & Merge)
│   └── infrastructure/     # 基础设施层 (Whisper / FFmpeg 调用)
├── api/v1/                 # API 定义
├── configs/                # 配置文件
├── docs/                   # 文档
│   ├── PRD.md              # 产品需求文档
│   ├── architecture.md     # 系统架构图
│   └── process-flow.md     # 核心处理流程图
├── test/                   # 测试
├── runtime-tools/          # 随发布包内置的第三方二进制工具
│   ├── whisper/            # Whisper.cpp 预编译二进制（按平台子目录）
│   └── ffmpeg/             # FFmpeg 预编译二进制（按平台子目录）
├── Makefile
├── go.mod
└── README.md
```

## 技术栈

| 组件 | 技术 |
|------|------|
| 语言 | Go 1.26 |
| 语音识别 | Whisper.cpp（os/exec 调用预编译二进制） |
| 视频合成 | FFmpeg（os/exec 调用） |
| GUI 框架 | Wails v2 + Svelte |
| 架构 | 分层架构（Domain / UseCase / Infrastructure） |

## 文档

- [产品需求文档 (PRD)](docs/PRD.md)
- [系统架构图](docs/architecture.md)
- [核心处理流程图](docs/process-flow.md)

## 发布计划

| 阶段 | 时间 | 内容 |
|------|------|------|
| Week 1-2 | CLI 核心 | 单文件处理 + 核心指令 |
| Week 3-4 | CLI 扩展 | 批量处理 + 配置管理 |
| Week 5-8 | GUI | Wails GUI 集成 |
| Week 9 | 测试 | 跨平台测试 + 打包 |
| Week 10 | 发布 | v1.0 正式发布 |

## License

[MIT](LICENSE)
