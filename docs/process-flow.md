# Vidify 核心处理流程图

## 单文件处理流程（时序图）

展示一次完整的音频转视频处理过程中各组件的交互。

```mermaid
sequenceDiagram
    participant User
    participant UI as CLI/GUI
    participant App as AppService
    participant UC as UseCase
    participant Whisper
    participant FFmpeg
    participant FS as 文件系统

    User ->> UI: 提供音频路径 + 配置
    UI ->> App: ProcessAudioToVideo(input, config)
    App ->> App: 验证输入参数
    App ->> FS: 检查文件是否存在
    FS -->> App: 文件存在

    %% Whisper 阶段
    App ->> UC: Transcribe(audio, model, language)
    UC ->> Whisper: RunWhisper(...)
    Whisper ->> FS: 读取音频
    FS -->> Whisper: 音频数据
    Note right of Whisper: os/exec 调用 whisper.cpp<br/>--model --language --output-srt
    Whisper ->> Whisper: 执行语音转写
    Whisper -->> UC: 返回 SRT 字幕
    UC -->> App: SubtitleResult

    App -->> UI: 进度: 字幕生成完成 (50%)

    %% FFmpeg 阶段
    App ->> UC: MergeVideo(audio, subtitle, bgImage)
    UC ->> FFmpeg: RunFFmpeg(mergeOptions)
    Note right of FFmpeg: os/exec 调用 ffmpeg<br/>-loop 1 -i bg.jpg -vf subtitles=...
    FFmpeg ->> FS: 读取背景图 + 音频 + 字幕
    FFmpeg ->> FS: 写入 MP4 文件
    FFmpeg -->> UC: MP4 路径
    UC -->> App: VideoResult

    App -->> UI: 处理完成 + 输出路径
    UI -->> User: 显示结果 / 视频预览
```

---

## 批量处理流程（活动图）

```mermaid
flowchart TD
    Start([开始])
    --> Input[接收输入路径]

    Input --> IsFolder{输入是文件夹?}

    IsFolder -->|是| Scan[扫描文件夹并过滤支持格式<br/>MP3 / WAV / M4A / OGG / FLAC]
    IsFolder -->|否| Single[将单文件加入处理队列]

    Scan --> HasFiles{找到音频文件?}
    HasFiles -->|否| Error1[错误: 未找到支持的音频文件]
    HasFiles -->|是| Queue[构建处理队列]

    Single --> Queue

    Queue --> Init[初始化进度统计<br/>总数 / 成功 / 失败]

    Init --> Loop{队列中还有文件?}

    Loop -->|是| Process[取出下一个文件<br/>更新进度显示]

    Process --> SingleProcess["单文件处理"]

    SingleProcess --> SuccessCheck{本次处理成功?}

    SuccessCheck -->|是| Success[成功计数 +1]
    SuccessCheck -->|否| Fail[失败计数 +1<br/>写入错误日志]

    Success & Fail --> Loop

    Loop -->|否| Summary[输出最终汇总<br/>总数 / 成功 / 失败 / 总耗时]
    Summary --> End([结束])

    Error1 --> End
```

---

## 错误处理策略

| 阶段         | 可能错误                | 处理方式                      |
| ------------ | ----------------------- | ----------------------------- |
| 输入验证     | 文件不存在、格式不支持  | 立即返回错误提示              |
| Whisper 转写 | 模型文件缺失、内存不足  | 提示下载模型 / 建议切换小模型 |
| FFmpeg 合成  | FFmpeg 未安装、编码失败 | 检测 PATH、提示安装指引       |
| 文件写入     | 磁盘空间不足、权限不足  | 提示清理空间 / 检查目录权限   |
