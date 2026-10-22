# Vidify 🎧➡️🎬

Vidify 是一个将本地音频转成字幕视频的 Go CLI。它使用本地 whisper.cpp 转录语音，输出以同步文本为主体的 MP4，并同时导出标准 SRT。项目是长期维护的个人工具，灵感来自为 EnglishPod 一类音频资源制作视频版本的需求，也参考了一个 Java 项目的实现思路；用途不限于英语学习。

**状态：开发中。** `process` 已接通音频转换、真实转录、累积字幕分页和 MP4 渲染，也已实现任务阶段状态和 Ctrl+C 取消。完整 MVP 尚未完成，主要缺口包括跨进程输出目录锁、阶段进度、结构化日志、清理警告和最终视觉样式。当前行为与验证范围见 [CLI 契约](docs/cli-contract.md)，已接受的目标与取舍见 [ADR 0001](docs/adr/0001-mvp-product-and-pipeline.md)。

## 运行前准备

从源码构建需要 **Go 1.23.0 或更高版本**。运行转换还需要自行准备以下依赖；Vidify 不下载、安装或打包它们。

| 依赖 | 要求与查找方式 |
| --- | --- |
| FFmpeg / ffprobe | FFmpeg 必须具备 `ass`（libass）滤镜和 `libx264`、`aac` 编码器。通过 PATH 或 `--ffmpeg-bin` 指定 FFmpeg；ffprobe 优先从 FFmpeg 同目录查找，再查 PATH。 |
| whisper.cpp | 使用能输出 JSON 的本地可执行文件。默认从 PATH 依次查找 `whisper-cli`、`whisper`、`main`，也可用 `--whisper-bin` 指定。版本与构建兼容范围尚未确定。 |
| Whisper 模型 | 使用与本地 whisper.cpp 兼容的模型文件，通过 `--model` 显式提供。当前真实样本验证使用 `ggml-base.en.bin` 英语模型。 |

MVP 正式支持范围为 WAV、MP3、M4A（AAC/ALAC），其中每种编码已有一种短样本完成端到端验证。其他 FFmpeg 可解码格式可以尝试，兼容性没有保证。当前验证环境是 Linux；跨平台发布行为仍待验证。

## 构建

```sh
git clone https://github.com/FeiNiaoBF/vidify.git
cd vidify
go build -o vidify .
```

Windows 下可将构建产物命名为 `vidify.exe`，后续命令使用相应名称。

## 处理一段音频

准备好输入文件和模型后运行：

```sh
./vidify process recording.wav --model ./models/ggml-base.en.bin --output-dir ./out
```

若外部工具不在 PATH 中，可以显式指定：

```sh
./vidify process recording.wav \
  --model ./models/ggml-base.en.bin \
  --output-dir ./out \
  --ffmpeg-bin /path/to/ffmpeg \
  --whisper-bin /path/to/whisper-cli
```

相对路径从运行命令时的工作目录计算；带空格的路径需要加引号。输入和模型保持只读。通过预检后，Vidify 会创建不存在的输出目录，并在处理期间使用系统临时目录和输出目录中的暂存文件。

成功时退出码为 0，终端打印两条 `Created` 消息及产物的绝对路径。示例产物为：

```text
out/
├── recording.mp4
└── recording.srt
```

MP4 为 1920×1080、30 fps、H.264 视频，输入音频重新编码为 AAC。当前画面为临时黑底、白色 DejaVu Serif 衬线文本：每个字幕片段出现后留在当前页，放不下下一个完整片段时翻页，最后一页持续到音频结束。SRT 保留片段原始的发声区间，因此其中一条字幕的结束时间可以早于视频中该段文字的消失时间。原始结束时间也可能略晚于音频时长，当前不会裁短。

## 限制与失败处理

- 任一同名 `.mp4` 或 `.srt` 已存在时，命令会拒绝处理。需要重试时可改用新的 `--output-dir`，以保留已有文件。
- 当前发布依赖输出文件系统支持硬链接，尚无跨进程目录锁。使用时应串行处理同一输出目录。
- SRT 先发布，MP4 后发布。失败或取消后可能留下有效 SRT；MP4 发布失败的错误会报告其路径。检查已有产物后，再选择新的输出目录重试。
- Ctrl+C 会取消任务并停止、等待当前外部进程。普通返回路径会尝试清理临时文件，但清理错误目前不会报告；异常退出也可能留下残留文件。
- 语言选择和自动检测尚未接入，识别语言由所用 whisper.cpp 的默认值决定。内置临时字体的多语言覆盖和复杂文字塑形尚未验证，不能据此承诺多语言视频可用。
- MVP 面向单个本地音频，不提供批处理、视频输入或说话人识别。当前预检尚未明确拒绝同时含视频流的容器。

完整的错误定位和实现差距见 [CLI 契约](docs/cli-contract.md#输出和错误约定)。查看当前可用选项无需安装外部工具或模型：

```sh
./vidify --help
./vidify process --help
```

## 开发与深入阅读

在仓库根目录运行现有检查：

```sh
go test ./...
```

这些测试覆盖字幕分页、SRT 发布保护、时长解析和任务状态，不会调用真实 FFmpeg 或 whisper.cpp；它们不能替代真实音频的转录和渲染验证。

- [CLI 契约](docs/cli-contract.md)：命令行为、历史验收范围、未完成项，以及[代码阅读路径](docs/cli-contract.md#代码阅读路径)。
- [术语表](GLOSSARY.md)：Segment、转录页和字幕视频的含义。
- [ADR 0001](docs/adr/0001-mvp-product-and-pipeline.md)：MVP 产品边界、工程决策与未决事项。
- [开发流程图](docs/vidify-development-roadmap.html)：在浏览器中打开，查看处理链路及尚待补齐的可靠性能力；当前行为以 CLI 契约为准。

## 许可

项目使用 [MIT License](LICENSE)。依赖与内置临时字体的许可说明见 [第三方声明](THIRD_PARTY_NOTICES.md)。
