# Vidify 🎧➡️🎬

Vidify 是一个长期维护的个人 Go CLI 项目，目标是将本地音频转录为以同步文本为主体的 MP4 视频，并同时输出标准 SRT。MVP 的产品边界与工程决策见 [ADR 0001：MVP 产品与工程决策](docs/adr/0001-mvp-product-and-pipeline.md)。仓库目前已提供 `process` 命令入口及帮助；参数解析与媒体处理尚未实现，文中的完整调用示例仍是目标用法。

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

# 3. 计划用法（尚未实现）

```sh
./vidify process recording.m4a --model ./models/ggml-base.en.bin --output-dir ./out
```

命令接口与首轮验收见 [MVP 命令行契约](docs/cli-contract.md)。输入使用一个位置参数，模型与输出目录显式指定；其它选项随对应功能接入。

# License

See [LICENSE](LICENSE) for details.
