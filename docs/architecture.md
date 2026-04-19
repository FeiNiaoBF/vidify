# Vidify 系统架构

## 架构概述

Vidify 采用 **Clean Architecture（六边形架构）** + **Modular Monolith** 设计，核心目标是：

- 高可维护性与可测试性（单人开发友好）
- 清晰的依赖方向（外层依赖内层）
- 便于从 CLI 平滑演进到 GUI

所有外部依赖（Whisper.cpp、FFmpeg、Viper 等）都被隔离在 Infrastructure 层，Domain 层完全不依赖任何外部库。

---

## 分层架构图

```mermaid
graph TD
    subgraph Presentation ["Presentation Layer\n(用户交互层)"]
        CLI["CLI\n(vidify 命令行)"]
        GUI["GUI\n(Wails + Svelte)"]
    end

    subgraph Application ["Application Layer\n(应用层)"]
        Service["App Service\n(协调器)"]
        Transcribe["UseCase: Transcribe\n(语音转文字)"]
        Render["UseCase: Render\n(视频合成)"]
    end

    subgraph Domain ["Domain Layer\n(领域核心)"]
        Models["业务模型\n(Audio, Subtitle, Video, Config)"]
        Ports["端口接口\n(Driving + Driven Ports)"]
    end

    subgraph Infrastructure ["Infrastructure Layer\n(基础设施层)"]
        Whisper["Whisper Adapter\n(os/exec)"]
        FFmpeg["FFmpeg Adapter\n(os/exec)"]
        Config["Config Adapter\n(Viper Loader)"]
        Logger["Logger"]
        Progress["Progress Reporter"]
    end

    %% 依赖关系
    CLI & GUI --> Service
    Service --> Transcribe & Render
    Transcribe & Render --> Models & Ports
    Whisper & FFmpeg & Config & Progress --> Ports

    %% 外部依赖
    Whisper -.-> ExternalW["Whisper.cpp\n(二进制 + 模型)"]
    FFmpeg -.-> ExternalF["FFmpeg\n(二进制)"]

    classDef pres fill:#E3F2FD,stroke:#1976D2,stroke-width:2px
    classDef app fill:#E8F5E9,stroke:#388E3C,stroke-width:2px
    classDef domain fill:#FFF3E0,stroke:#F57C00,stroke-width:2px
    classDef infra fill:#F3E5F5,stroke:#7B1FA2,stroke-width:2px

    class Presentation pres
    class Application app
    class Domain domain
    class Infrastructure infra
```

---

## 核心设计原则

依赖倒置（Dependency Inversion）：高层模块不依赖低层模块，两者都依赖抽象（端口）
领域层独立：domain 层零外部依赖，是整个项目的核心
端口与适配器：Driving Ports（主端口）由 CLI/GUI 调用，Driven Ports（被驱动端口）由 Infrastructure 实现
Use Case 显式化：每个主要业务流程都有独立的 Use Case（Transcribe / Render）

---

## 项目目录结构简要

```
internal/
├── domain/
│   ├── port/               # driving.go (Processor, Result) + driven.go (Transcriber, VideoMerger, ProgressReporter, Logger)
│   ├── audio.go
│   ├── config.go
│   ├── subtitle.go
│   ├── video.go
│   └── errors.go
├── application/
│   ├── usecase/
│   │   ├── transcribe.go
│   │   └── render.go
│   └── service.go
├── infrastructure/
│   ├── whisper/            # port.Transcriber implementation
│   ├── ffmpeg/             # port.VideoMerger implementation
│   ├── config/             # Viper config loader
│   ├── logger/             # port.Logger implementation (slog-backed)
│   └── progress/           # port.ProgressReporter implementation (mpb)
├── adapter/
│   └── cli/                # Cobra commands; calls di.Build(), never imports infra directly
├── di/
│   └── wire.go             # Single composition root: wires all infra → use cases → service
└── common/
    └── errors/             # Shared error utilities (future use)
```
