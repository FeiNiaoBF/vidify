下面是 vidify 的最小“项目流程”概览，分为 CLI 和 Web API 两种使用模式，简要说明数据如何在模块之间流转。

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
