# Vidify 🎧➡️🎬

Vidify 是一个长期维护的个人 Go CLI 项目，目标是将本地音频转录为以同步文本为主体的 MP4 视频，并同时输出标准 SRT。MVP 的产品边界与工程决策见 [ADR 0001：MVP 产品与工程决策](docs/adr/0001-mvp-product-and-pipeline.md)。`process` 已具备真实转录、SRT、ASS 累积分页和 MP4 渲染的端到端切片；任务状态和 Ctrl+C 取消已接入，阶段进度、日志、并发发布锁和最终视觉样式等 MVP 要求仍未完成。

**灵感来源:** 为类似 EnglishPod 的音频资源自动创建视频版本。（主要从一个Java项目中得到的灵感，想着可以自己做一个类似的功能）

**(🚧 状态：开发中)**

## 依赖(暂定)

* **Go:** 1.23 或更高版本
* **FFmpeg / ffprobe:** 命令行工具，通常随同一发行包提供；FFmpeg 需在 PATH 中或通过 `--ffmpeg-bin` 指定，ffprobe 会从 FFmpeg 同目录或 PATH 查找。
* **Whisper.cpp:** 编译好的可执行文件（通常叫 `whisper-cli`、`whisper` 或 `main`），需要能在 PATH 中被找到或通过 `--whisper-bin` 指定。
* **Whisper 模型:** 本地兼容的 `.bin` 模型文件，通过 `--model` 显式指定。

## 安装

```bash
# 1. 克隆仓库 (替换为你自己的仓库地址)
# git clone https://github.com/FeiNiaoBF/vidify.git
cd vidify
```

# 2. **构建**
```bash
go build -o vidify .
```

# 3. 使用

```sh
./vidify process recording.wav --model ./models/ggml-base.en.bin --output-dir ./out
```

若 Whisper 不在 PATH 中，可加 `--whisper-bin /path/to/whisper-cli`；FFmpeg 同理使用 `--ffmpeg-bin`。MP4 当前使用临时黑底样式和内置 DejaVu Serif 衬线字体，字幕按实际字形度量累积分页；字体、配色和页面参数仍属临时值。命令接口、当前行为和未完成项见 [MVP 命令行契约](docs/cli-contract.md)。

# License

See [LICENSE](LICENSE) for Vidify's license and [third-party notices](THIRD_PARTY_NOTICES.md) for dependency and embedded-font licenses.
