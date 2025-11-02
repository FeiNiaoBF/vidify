# Vidify 🎧➡️🎬

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go" alt="Go Version" />
  <img src="https://img.shields.io/github/license/FeiNiaoBF/vidify" alt="License" />
  <img src="https://img.shields.io/github/v/release/FeiNiaoBF/vidify" alt="Release" />
</p>

一个基于 Go 语言的音频转视频工具，通过集成 Whisper.cpp 进行语音识别和 FFmpeg 进行视频处理，将音频文件自动转换为带同步字幕的视频文件。

## 特性

- 🎯 **高精度语音识别**：集成 Whisper.cpp，支持多语言音频识别
- 🎬 **智能视频生成**：基于 FFmpeg 自动生成带同步字幕的视频
- 🚀 **高并发处理**：支持批量任务并发执行
- 🔧 **双模式支持**：提供 CLI 命令行和 Web API 两种使用方式
- ⚙️ **灵活配置**：支持自定义字幕样式、视频参数等
- 📦 **模块化设计**：清晰的代码结构，易于扩展和维护

## 系统要求

- Go 1.23 或更高版本
- FFmpeg 4.0+ (需安装到系统 PATH)
- Whisper.cpp (自动集成或手动配置)
- 操作系统：Linux / macOS / Windows

## 快速开始

### 安装

```bash
# 克隆项目
git clone https://github.com/FeiNiaoBF/vidify.git
cd vidify

# 安装依赖
go mod download

# 编译项目
make build

# 或直接运行
go run main.go
```

### CLI 模式使用

```bash
# 基本用法
./vidify convert -i input.mp3 -o output.mp4

# 指定字幕样式
./vidify convert -i input.mp3 -o output.mp4 \
  --subtitle-style "FontSize=24,PrimaryColour=&HFFFFFF"

# 批量处理
./vidify batch -d ./audio_files -o ./output_videos

# 查看帮助
./vidify --help
```

### Web API 模式

```bash
# 启动 API 服务
./vidify serve --port 8080

# 调用 API
curl -X POST http://localhost:8080/api/v1/convert \
  -F "file=@input.mp3" \
  -F "config={\"subtitle_style\":\"FontSize=24\"}"
```

## 项目结构

```
vidify/
├── cmd/                    # 命令行入口
│   ├── cli/               # CLI 命令实现
│   └── server/            # Web 服务器
├── internal/              # 私有应用代码
│   ├── audio/            # 音频处理模块
│   ├── video/            # 视频生成模块
│   ├── subtitle/         # 字幕处理模块
│   ├── whisper/          # Whisper 集成
│   ├── ffmpeg/           # FFmpeg 封装
│   ├── processor/        # 任务处理器
│   └── config/           # 配置管理
├── pkg/                   # 可复用的公共库
│   ├── logger/           # 日志工具
│   ├── validator/        # 参数验证
│   └── utils/            # 通用工具
├── api/                   # API 定义
│   ├── v1/               # API v1 版本
│   └── middleware/       # 中间件
├── configs/               # 配置文件
│   ├── config.yaml       # 默认配置
│   └── config.example.yaml
├── test/                  # 测试文件
│   ├── integration/      # 集成测试
│   └── testdata/         # 测试数据
├── scripts/               # 构建和部署脚本
├── docs/                  # 文档
│   ├── architecture.md   # 架构设计
│   └── api.md            # API 文档
├── deployments/           # 部署配置
│   ├── docker/           # Docker 配置
│   └── k8s/              # Kubernetes 配置
├── main.go               # 程序入口
├── Makefile              # 构建脚本
├── go.mod                # Go 模块文件
└── README.md             # 项目说明
```

## 配置说明

在 `configs/config.yaml` 中配置：

```yaml
app:
  name: "Vidify"
  version: "1.0.0"
  env: "production"

server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 60
  write_timeout: 60

whisper:
  model_path: "./models/ggml-base.bin"
  language: "auto"
  threads: 4

ffmpeg:
  binary_path: "ffmpeg"
  default_format: "mp4"
  video_codec: "libx264"
  audio_codec: "aac"

subtitle:
  font_name: "Arial"
  font_size: 24
  primary_colour: "&HFFFFFF"
  outline_colour: "&H000000"

processor:
  max_workers: 4
  queue_size: 100
  temp_dir: "./temp"
  output_dir: "./output"
```

## 运行测试

```bash
# 运行所有测试
make test

# 运行单元测试
go test ./internal/...

# 运行集成测试
go test ./test/integration/...

# 生成测试覆盖率报告
make coverage
```

<!-- ## 📖 API 文档

详细的 API 文档请参考 [API Documentation](docs/api.md) -->

### 主要端点

- `POST /api/v1/convert` - 音频转视频
- `GET /api/v1/status/:task_id` - 查询任务状态
- `GET /api/v1/download/:task_id` - 下载生成的视频
- `DELETE /api/v1/task/:task_id` - 取消任务

<!-- ## Docker 部署

```bash
# 构建镜像
docker build -t vidify:latest .

# 运行容器
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/configs:/app/configs \
  -v $(pwd)/output:/app/output \
  vidify:latest
``` -->

## 贡献指南

欢迎贡献代码！请遵循以下步骤：

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

### 代码规范

- 遵循 [Effective Go](https://golang.org/doc/effective_go.html) 编码规范
- 使用 `gofmt` 格式化代码
- 编写单元测试，保持测试覆盖率 > 80%
- 提交前运行 `make lint` 检查代码

## 开发路线图

- [ ] 基础音频转视频功能
- [ ] Whisper.cpp 集成
- [ ] CLI 命令行工具
- [ ] Web API 服务
- [ ] 支持更多字幕格式 (SRT, VTT)
- [ ] 视频预览功能
- [ ] 云存储集成 (S3, OSS)
- [ ] WebSocket 实时进度推送
- [ ] 前端管理界面

## 许可证

本项目采用 [MIT License](LICENSE) 开源协议。

## 致谢

- [Whisper.cpp](https://github.com/ggerganov/whisper.cpp) - 高性能语音识别
- [FFmpeg](https://ffmpeg.org/) - 强大的多媒体处理工具

## 联系方式

如有问题或建议，请通过以下方式联系：

- 提交 [Issue](https://github.com/FeiNiaoBF/vidify/issues)
- 邮件：暂无

---

⭐ 如果这个项目对您有帮助，请给一个 Star 支持！
```
