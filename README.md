# Vidify 🎧➡️🎬

一个简单的个人 Go 项目，用于将音频文件（如 MP3, WAV）通过 Whisper.cpp 生成字幕，并使用 FFmpeg 将音频和字幕合并成一个带字幕的视频文件（MP4）。

**灵感来源:** 为类似 EnglishPod 的音频资源自动创建视频版本。（主要从一个Java项目中得到的灵感，想着可以自己做一个类似的功能）

**(🚧 状态：开发中)**

## 依赖(暂定)

* **Go:** 1.23 或更高版本
* **FFmpeg:** 命令行工具，需要能在 PATH 中被找到或通过参数指定。
* **Whisper.cpp:** 编译好的可执行文件 (通常叫 `main` 或 `whisper`)，需要能在 PATH 中被找到或通过参数指定。
* **Whisper 模型:** 下载好的 `.bin` 模型文件 (例如 `ggml-base.en.bin`)。

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
```bash
./vidify process -i <foo.mp3> -o <foo.mp4> -m <ggml-base.en.bin> [其他选项...]
```

(TODO)主要选项:
- `-i, --input`: 必需, 输入音频文件。
- `-o, --output`: 必需, 输出视频文件。
- `-m, --model`: 必需, Whisper 模型文件路径。
- `--whisper-bin`: (可选) Whisper.cpp 可执行文件路径 (如果不在 PATH 中)。
- `--ffmpeg-bin`: (可选) FFmpeg 可执行文件路径 (如果不在 PATH 中)。
- `-s, --srt`: (可选) 指定 SRT 字幕输出路径。

# License

See [LICENSE](LICENSE) for details.
