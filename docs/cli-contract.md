# MVP 命令行契约

状态：首轮 CLI 实现基线。已实现 `process` 注册、帮助和未实现错误返回；位置参数、选项和媒体处理尚未实现。依据已确认的 [MVP 决策](adr/0001-mvp-product-and-pipeline.md)，本文定义调用方式、帮助和错误行为。媒体处理与后续选项逐步接入；下述完整调用示例仍是目标接口。

## 主要入口

```sh
vidify process recording.m4a --model ./models/ggml-base.en.bin --output-dir ./out
```

- `process` 执行一次音频到字幕视频的完整任务。
- 恰好一个位置参数表示输入音频；首轮不同时支持 `--input`。
- `--model` 接收本地模型文件路径，必须显式提供，值不能为空。
- `--output-dir` 接收输出目录路径，首轮要求显式提供，值不能为空；它不接收 MP4 文件名。
- 首轮只提供长选项，`-h` 为帮助选项。短别名可在有需要时增加。
- 相对路径相对于调用者的当前工作目录解释。带空格路径按所用 Shell 的规则加引号。
- `--` 结束选项解析，可用于传入以 `-` 开头的文件名。

例如：

```sh
vidify process "my recording.m4a" --model ./models/ggml-base.en.bin --output-dir ./out
vidify process --model ./models/ggml-base.en.bin --output-dir ./out -- -recording.wav
```

完整 MVP 成功后，示例输入对应 `out/recording.mp4` 与 `out/recording.srt`。文件发布与拒绝覆盖遵循 ADR；这不是首轮 CLI 练习要实现的功能。

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

Example:
  vidify process recording.m4a --model ./models/ggml-base.en.bin --output-dir ./out
```

根命令无参数时展示帮助；`vidify --help` 能发现 `process`。帮助必须在没有 FFmpeg、whisper.cpp 或模型时正常显示，不启动外部进程，不创建输出目录或临时文件。

## 输出和错误约定

- 显式请求帮助：帮助写入 stdout，退出码为 0。
- 参数错误：诊断写入 stderr，非零退出；提示相关参数或 `vidify process --help`。
- 未知命令、未知选项、缺少或多余输入均报错，不猜测用户意图。
- 错误只呈现一次，避免 Cobra 与入口函数重复打印同一错误。
- 完整流程的进度、警告和错误写入 stderr；成功结果信息写入 stdout。精确结果格式稍后确定。
- 媒体处理尚未接入时，合法参数应进入处理入口并返回明确的“尚未实现”错误，非零退出。不得打印成功或创建占位 MP4/SRT。
- 最终任务成功、取消和清理警告语义遵循 ADR；各类非零退出码的细分暂未确定。

## 首轮验收

首轮只验证命令发现、参数解析、基本参数约束和错误传播，不验证音频内容、路径可访问性或外部依赖。以下示例中的模型路径可尚不存在。

| 调用 | 预期行为 |
| --- | --- |
| `vidify` | 展示根帮助 |
| `vidify --help` | stdout 展示帮助，包含 process，退出 0 |
| `vidify process --help` | stdout 展示用途、参数与示例，退出 0 |
| `vidify process --model model.bin --output-dir out` | 缺少输入，非零退出 |
| `vidify process a.wav b.wav --model model.bin --output-dir out` | 输入过多，非零退出 |
| `vidify process a.wav --output-dir out` | 缺少 model，非零退出 |
| `vidify process a.wav --model model.bin` | 缺少 output-dir，非零退出 |
| 显式传入空字符串作为输入、model 或 output-dir | 指出空值参数，非零退出 |
| 传入未知命令或未知选项 | 非零退出，提供帮助线索 |
| 参数合法，包括引号包裹的空格路径和 `--` 后的文件名 | 到达处理入口，返回“尚未实现”，不生成文件 |

每种错误应单独验证，不依赖多个错误同时出现时的报告顺序。路径中有空格不等于空路径，不能随意修改合法文件名。

## 后续接入

已确认的可执行文件显式路径、语言选择和 `--keep-temp` 会随对应行为一起接入，避免出现被接受却不起作用的选项。`--version` 在版本信息接入时补充；详细日志选项和工作区路径选项仍需确定。MVP 不将内部转换、排版或发布阶段暴露为独立子命令。

协作方式：AI 按小步编写代码，同时解释设计理由、具体逻辑和验证结果；维护者参与理解、设计与审查。当前步骤只交付命令注册与帮助，参数解析和媒体处理尚未实现。

## 当前开发方式

下一步采用轻量纵向切片，优先让一个正常的本地音频经过真实转录、字幕排版和编码，生成有效 SRT 与可播放 MP4。尽量沿用现有结构，将切片的主要实现集中在一个文件中，保留现有 Cobra 入口；不预建空目录、通用抽象或按阶段机械拆包。代码按可持续演进的产品代码编写，注释解释关键取舍与逻辑。

代表性流程跑通并完成针对性验证后，在扩展更多功能前整理已显现的独立职责，让 CLI 保持精简。以 ADR 的已接受要求为边界，暂定的字体、样式、检查细节应明确标注。旧 HTML 路线图仅用于理解业务步骤，不要求逐阶段搭建基础设施后才联调。
