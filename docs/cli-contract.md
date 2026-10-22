# MVP 命令行契约

状态：端到端切片已实现。`process` 接受一个音频位置参数、`--model`、`--output-dir`，可选 `--ffmpeg-bin` 与 `--whisper-bin`；当前实现调用 FFmpeg、ffprobe 和 whisper.cpp，生成 SRT 与按累积文本分页的 ASS 字幕 MP4。真实英语样本已手动验证；完整 MVP 尚未完成。

## 主要入口

```sh
vidify process recording.m4a --model ./models/ggml-base.en.bin --output-dir ./out
```

- `process` 执行一次音频到字幕视频的完整任务。
- 恰好一个位置参数表示输入音频；首轮不同时支持 `--input`。
- `--model` 接收本地模型文件路径，必须显式提供，值不能为空。
- `--output-dir` 接收输出目录路径，首轮要求显式提供，值不能为空；它不接收 MP4 文件名。有效处理时若目录不存在，Vidify 会创建它。
- `--ffmpeg-bin`、`--whisper-bin` 可选，用于指定对应可执行文件路径；未指定时从 PATH 查找。ffprobe 从 FFmpeg 同目录或 PATH 自动查找；Whisper 默认依次查找 `whisper-cli`、`whisper`、`main`。
- 首轮只提供长选项，`-h` 为帮助选项。短别名可在有需要时增加。
- 相对路径相对于调用者的当前工作目录解释。带空格路径按所用 Shell 的规则加引号。
- `--` 结束选项解析，可用于传入以 `-` 开头的文件名。

例如：

```sh
vidify process "my recording.m4a" --model ./models/ggml-base.en.bin --output-dir ./out
vidify process --model ./models/ggml-base.en.bin --output-dir ./out -- -recording.wav
```

完整 MVP 成功后，示例输入对应 `out/recording.mp4` 与 `out/recording.srt`。当前切片也使用此命名，并拒绝覆盖已有产物；任务锁、跨平台发布语义和全部失败路径仍以 ADR 为最终要求。

## 帮助草案

下面规定信息内容，不要求 Cobra 输出的空格、排序和排版逐字一致。

```text
Generate a subtitle video and an SRT file from one audio file.

Usage:
  vidify process <audio-file> --model <path> --output-dir <directory>

Flags:
  -h, --help                   Show help for process
      --model string           Local whisper.cpp model file (required)
      --output-dir string      Directory for MP4 and SRT outputs (required)
      --ffmpeg-bin string      Path to the FFmpeg executable
      --whisper-bin string     Path to the whisper.cpp executable

Example:
  vidify process recording.m4a --model ./models/ggml-base.en.bin --output-dir ./out
```

根命令无参数时展示帮助；`vidify --help` 能发现 `process`。帮助必须在没有 FFmpeg、whisper.cpp 或模型时正常显示，不启动外部进程，不创建输出目录或临时文件。

## 输出和错误约定

- 显式请求帮助：帮助写入 stdout，退出码为 0；不探测外部依赖或创建输出目录。
- 参数、依赖或处理错误：诊断写入 stderr，退出非零；错误只呈现一次。
- 成功时将 MP4 与 SRT 路径写入 stdout。当前不显示阶段进度，也没有结构化日志。
- 参数合法时，先检查输入、模型、可执行文件和 FFmpeg 编码能力，再转换音频、转录、生成字幕并渲染视频。
- 默认拒绝覆盖已有输出。SRT 先在输出目录暂存并写完，再以不覆盖方式一次发布，最终路径不会在内容写完前出现；之后发布 MP4。MP4 发布失败时保留已发布的 SRT 并报告其路径。
- 任务在进程内按线性阶段推进，并以 succeeded、failed、canceled 结束；错误包含发生阶段。Ctrl+C 会取消当前子进程、等待其退出，并以非零状态结束。阶段进度、结构化日志、清理失败警告和精确退出码语义仍待实现。

## 当前切片已实现并验证

- 检查输入和模型文件，使用 FFmpeg 确认音频流可解码，并转成 16 kHz、单声道、16-bit PCM WAV。
- 调用 whisper.cpp JSON 输出，用毫秒偏移创建 `Segment` 并生成标准 SRT。
- 当前 MP4 使用临时黑底和内置 DejaVu Serif 字体。Go 使用同一字体文件的实际字形边界与行高换行并计算页面容量；新页会在需要时缩小字号，字号下限为临时值 32 px。ASS 为每个 Segment 生成一个从原始开始时间延续到本页结束的事件；视频使用 1920×1080、30 fps、H.264，音频使用 AAC。
- 新页从触发分页的 Segment 开始；前页在同一时间结束；末页延伸到 ffprobe 读取的音频时长。ASS 边界统一取最近厘秒，SRT 保留 Whisper 原始毫秒区间。
- 输出名取输入文件基础名。已有 MP4 或 SRT 时拒绝覆盖。
- `/tmp/vidify-prereqs/samples/representative.wav` 的手动端到端验证得到 52 个时间戳片段；MP4 可解码，抽帧检查确认页面累积、切换和字体显示正常，视频时长与音频一致。产物位于 `/tmp/vidify-prereqs/pagination-output-final/`。

## 输入格式验收

当前提交在 Linux、FFmpeg/ffprobe 6.1.1、whisper.cpp 本地构建与 `ggml-base.en.bin` 模型下，使用同一段 11 秒英语音频转换出的以下样本完成端到端验证。这覆盖各格式的一种编码配置，不代表所有采样率、声道、容器变体或损坏方式均已验证。

| 输入格式 | 输入容器时长 | 输出 MP4 时长 | 结果 |
| --- | --- | --- | --- |
| WAV（PCM s16le） | 11.000 秒 | 11.000 秒 | 通过 |
| MP3 | 11.088 秒 | 11.000 秒 | 通过 |
| M4A（AAC） | 11.000 秒 | 11.008 秒 | 通过 |
| M4A（ALAC） | 11.000 秒 | 11.000 秒 | 通过 |

四组输出均包含 H.264 视频（1920×1080、30 fps）与 AAC 音频，FFmpeg 完整解码通过；SRT 非空、序号连续、时间戳递增、区间为正且未超出输入时长。输入文件内容未改变，输出目录中没有暂存文件残留。时长对比使用输入容器时长，不能据此证明字幕逐段同步精度。

异常输入验证覆盖损坏 WAV、只保留前 64 字节的截断 MP3、无音频流的 MP4，以及有音频流但流和容器均未提供时长的零采样 WAV。四种输入均在预检阶段以退出码 1 失败，stdout 为空，未创建输出目录，也未遗留任务临时文件。

已有输出保护覆盖四组成功产物的重复运行，以及输出目录中仅有同名 SRT 或仅有同名 MP4 的情况。均拒绝处理，已有文件内容不变，没有创建其他产物或遗留暂存文件；此验证不覆盖跨进程竞争。

## 尚未完成的 MVP 行为

- 正式输入格式的其他编码配置、更多真实样本和损坏方式，以及长音频仍待验证。
- 阶段进度、结构化日志、任务 ID、UUID 临时目录、`--keep-temp`、跨进程输出目录锁和完整清理警告语义仍待实现。
- 内置 DejaVu Serif 是带许可证的临时字体，但覆盖范围有限；多语言字形覆盖和复杂文字塑形尚未验证，字体替换策略仍待决定。字体文件与许可见 `cmd/cli/assets/`。最终视觉样式、跨平台发布与文件锁、FFmpeg 长音频性能尚未验证。

## 后续接入

已接入 `--ffmpeg-bin` 和 `--whisper-bin`。语言选择、`--keep-temp`、`--version`、详细日志选项和工作区路径选项仍待接入；每个选项应与其实际行为一起加入。MVP 不将内部转换、排版或发布阶段暴露为独立子命令。

协作方式：AI 按小步编写代码，同时解释设计理由、具体逻辑和验证结果；维护者参与理解、设计与审查。累积分页面和首轮职责重构已完成；后续按 MVP 可靠性要求继续小步实现。

## 当前开发方式

累积字幕分页已接入并通过真实样本验证。实现已按职责拆分在 `cmd/cli` 包内：CLI 与任务编排、外部工具、转录与 SRT、字幕布局与 ASS、产物验证与发布；外部 CLI 行为保持不变。进程内阶段状态和 Ctrl+C 取消已实现，四种正式输入格式各有短样本通过验收；下一步先实现输出目录跨进程锁，再补齐临时工作区、清理警告、阶段进度和日志。不要按阶段机械地预建包。字体、配色和页面参数仍是临时值，最终样式继续遵循 ADR 的待决定事项。
