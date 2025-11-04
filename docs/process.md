# Vidify 处理流程

## 总体数据流
- 输入：音频文件（wav/mp3 等）
- 语音识别：调用 Whisper.cpp，将音频转成 SRT 字幕
- 视频合成：调用 FFmpeg，在纯色或自定义背景上烧录 SRT，并叠加原始音频
- 输出：带字幕的视频文件（mp4）

## CLI 模式
1. 参数解析：`vidify convert -i input.mp3 -o output.mp4` 由 Cobra 解析，合并 CLI 参数、环境变量与配置文件。
2. 识别阶段：`internal/tools/transcriber` 调用 `whisper-cli`，必要时将音频转为 16 kHz 单声道 WAV，并生成 `.srt`。
3. 合成阶段：`internal/video` 和 `internal/ffmpeg` 根据配置拼装命令，渲染带字幕的视频；字幕样式、分辨率等来自配置或 CLI 覆盖。
4. 收尾：`internal/storage` 将产物写入目标目录，清理临时文件，CLI 打印结果路径。

## Web API 模式（规划中）
- `cmd/server` 暴露 `POST /api/v1/convert`，请求体通过 multipart 上传音频与可选的 SRT、模型参数。
- Server 将音频暂存于临时目录，复用 `pipeline` 完成转录和渲染。
- 处理完成后返回 JSON：`{"output":"outputs/<task>.mp4"}`；未来扩展任务查询与取消接口。

## 配置与资源
- FFmpeg 与 Whisper 可执行文件需在 `PATH` 或通过配置提供绝对路径。
- `configs/` 中的模板定义默认输出目录、模型路径、字幕样式；CLI 参数可覆盖关键选项。
- Whisper 模型建议放在 `models/` 或 `third_party/whisper/` 下，避免重复下载。

## 常见故障排查
- FFmpeg 未安装或路径不正确：确认 `ffmpeg -version` 可用，或在配置中显式指定。
- Whisper CLI / 模型缺失：检查 `whisper-cli` 路径与模型文件名是否与配置一致。
- Windows 路径转义导致字幕滤镜失败：使用双反斜杠或在参数中包裹引号。
- 输出目录权限不足：确保 `output/` 和临时目录具有写入权限。下面是 vidify 的最小“项目流程”概览，分为 CLI 和 Web API 两种使用模式，简要说明数据如何在模块之间流转。

总体数据流
- 输入：音频文件（wav/mp3 等）
- 语音识别：调用 Whisper.cpp，用指定模型把音频转成 SRT 字幕
- 视频合成：调用 FFmpeg，在纯色或自定义背景上烧录 SRT，同步合成音频
- 输出：带字幕的视频文件（mp4）

CLI 模式流程
1) 参数解析
   - 必填：-i 输入音频、-o 输出视频
   - 若无 -srt，则需提供 -model 与 -whisper-bin
2) 识别阶段（可跳过）
   - 调用 whisper.cpp（-m 模型 -f 音频 -osrt -of 输出前缀）
   - 生成临时 SRT 文件
3) 合成阶段
   - 调用 ffmpeg：生成背景画面 + 烧录 SRT + 叠加音频
   - 输出 mp4 到指定路径
4) 结果
   - 控制台打印输出视频路径

Web API 模式流程
1) 接口
   - POST /api/convert（multipart）
   - 表单字段：audio（文件）、model、lang、whisper_bin
2) 暂存与参数准备
   - 将上传音频写入 tmp/uploads
   - 确定模型、语言、whisper 可执行路径
3) 识别与合成
   - 若无 SRT，先 whisper 生成 SRT
   - ffmpeg 烧录字幕并合成视频
4) 返回结果
   - JSON：{"output": "outputs/xxx.mp4"}

核心模块职责
- transcribe：封装对 whisper.cpp 的调用，输出 SRT
- video：封装对 ffmpeg 的调用，生成视频
- pipeline：串联“识别 -> 合成”的主流程，处理临时目录与结果路径
- server：提供 /healthz 与 /api/convert
- config：集中配置与参数传递

目录与产物
- 输入：用户指定音频
- 临时：tmp/ 下的中间文件（如 SRT）
- 输出：outputs/ 下的视频文件
- 模型：models/ 下的 whisper 模型
- 第三方：third_party/whisper.cpp 可执行文件

分支分流规则
- 若提供现成 SRT：跳过识别，直接进入视频合成
- 未提供 SRT：必须提供模型与 whisper 可执行文件，先识别再合成

失败常见点（排查顺序）
- ffmpeg 未安装或不在 PATH
- whisper 可执行文件路径错误或未构建
- 模型文件不存在或路径错误
- Windows 路径转义导致字幕滤镜失败（需正确转义路径）

这就是最小闭环流程；在此基础上可扩展异步任务、样式模板、波形/背景图等能力。需要的话，我可以把这套流程画成一张时序图或生成一张 README 中的“工作流示意图”。
