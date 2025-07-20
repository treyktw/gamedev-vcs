# Game Development VCS

A modern version control system built specifically for game development, with first-class support for binary assets, real-time collaboration, and UE5 integration.

## 🎯 Why This Exists

Git wasn't designed for game development. This VCS solves the core problems that make Git unsuitable for game teams:

| **Challenge** | **Git Problems** | **Our Solution** |
|---------------|------------------|-------------------|
| **Binary Assets** | Git LFS is clunky, expensive, slow | First-class binary support with chunking & smart diffs |
| **File Locking** | No built-in locking = merge conflicts on binaries | Real-time file locking with UE5 editor integration |
| **Asset Dependencies** | No understanding of .uasset references | Dependency graph tracking prevents broken references |
| **Team Awareness** | Can't see who's working on what | Live presence: see who's editing which assets in real-time |
| **Push Granularity** | Must push entire repo state | Push single files or asset trees independently |
| **Storage Bloat** | Binary files balloon repo size | Content-addressable storage + delta compression |
| **UE5 Integration** | Zero native support | Deep editor integration with asset browser overlays |

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                               DEVELOPER WORKSTATIONS                              │
├─────────────────┬─────────────────┬─────────────────┬─────────────────────────────┤
│   UE5 Editor    │    CLI Tool     │   Desktop GUI   │     Web Browser            │
│   + Plugin      │   (vcs.exe)     │   (Optional)    │   (Dashboard)              │
│   (C++)         │   (Go)          │   (Tauri/Go)    │                            │
└─────────────────┴─────────────────┴─────────────────┴─────────────────────────────┘
                                │
                                │ HTTPS/WSS over VPN
                                ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                            YOUR SERVER (Kubernetes)                             │
├─────────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐                 │
│  │   API Gateway   │  │  Web Dashboard  │  │   File Proxy    │                 │
│  │   (Traefik)     │  │ (Elixir/Phoenix)│  │   (Go/Nginx)    │                 │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘                 │
│                                │                                                │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐                 │
│  │   Core VCS API  │  │   Real-time     │  │   Auth Service  │                 │
│  │   (Go)          │  │   State (Redis) │  │   (Go/JWT)      │                 │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘                 │
│                                │                                                │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐                 │
│  │   ClickHouse    │  │  Content Store  │  │   Background    │                 │
│  │  (Analytics)    │  │  (File System)  │  │   Workers       │                 │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Make (optional, for easy commands)

### 1. Clone & Build

```bash
git clone https://github.com/yourstudio/gamedev-vcs
cd gamedev-vcs

# Install dependencies and build
make build

# Or manually:
go mod download
go build -o build/vcs ./cmd/vcs
go build -o build/vcs-server ./cmd/vcs-server
```

### 2. Start the Server

```bash
# Run with make
make run-server

# Or directly
./build/vcs-server

# Server starts on http://localhost:8080
# Health check: http://localhost:8080/health
```

### 3. Use the CLI

```bash
# Basic commands (currently mock implementations)
./build/vcs init my-game
./build/vcs add Assets/Characters/Hero.uasset
./build/vcs commit -m "Added hero character"
./build/vcs push
./build/vcs status --team
./build/vcs lock Assets/Levels/MainLevel.umap
```

## 📦 Current Status

This is the **foundation phase**. We have:

### ✅ Working
- **CLI Framework**: Full command structure with Git-like interface
- **Server Framework**: RESTful API with all endpoints defined
- **Build System**: Cross-platform builds, Docker support
- **Health Monitoring**: Basic health checks and logging

### 🚧 In Development (Next 2 Weeks)
- **Content-Addressable Storage**: Hash-based file storage with deduplication
- **Redis Integration**: Real-time file locking and team presence
- **ClickHouse Analytics**: Commit history and team productivity metrics
- **Binary Asset Handling**: Chunked uploads for large UE5 files
- **Dependency Tracking**: UE5 asset reference analysis

### 🔮 Future Features
- **UE5 Plugin**: Deep editor integration
- **Web Dashboard**: Elixir/Phoenix real-time collaboration UI
- **Conflict Resolution**: Visual merge tools for binary assets
- **Mobile App**: Team monitoring on the go

## 🛠️ Development

### Available Make Commands

```bash
make help              # Show all available commands
make build            # Build both CLI and server
make build-cli        # Build just the CLI tool
make build-server     # Build just the server
make run-server       # Build and run server
make run-cli          # Build and run CLI
make dev-server       # Run server in development mode
make test             # Run tests
make clean            # Clean build artifacts
make docker-build     # Build Docker images
make quick-test       # Run smoke tests
```

### Development Workflow

```bash
# Start development server with hot reload
make dev-server

# In another terminal, test CLI commands
make dev-cli

# Run tests
make test

# Build for production
make build-all-platforms
```

## 🔧 Configuration

### Server Environment Variables

```bash
PORT=8080                                    # Server port
STORAGE_PATH=./storage                       # Local file storage path
REDIS_URL=redis://localhost:6379           # Redis connection
CLICKHOUSE_URL=http://localhost:8123       # ClickHouse connection
JWT_SECRET=your-secret-key                  # JWT signing key
ENVIRONMENT=development                      # Environment mode
```

### CLI Configuration

The CLI automatically connects to `https://vcs.yourstudio.com` by default, but you can override:

```bash
vcs --server http://localhost:8080 status
```

## 📊 API Endpoints

### Authentication
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/refresh` - Token refresh

### Projects
- `GET /api/v1/projects` - List projects
- `POST /api/v1/projects` - Create project
- `GET /api/v1/projects/:id` - Get project details
- `DELETE /api/v1/projects/:id` - Delete project

### Files
- `POST /api/v1/files/upload` - Upload file
- `GET /api/v1/files/:hash` - Download file
- `POST /api/v1/files/upload-chunk` - Chunked upload
- `POST /api/v1/files/finalize-upload` - Finalize upload

### Locks
- `POST /api/v1/locks/:project/:file` - Lock file
- `DELETE /api/v1/locks/:project/:file` - Unlock file
- `GET /api/v1/locks/:project` - List locks

### Real-time
- `GET /api/v1/ws` - WebSocket connection

### Analytics
- `GET /api/v1/analytics/productivity/:project` - Team productivity
- `GET /api/v1/analytics/activity/:project` - Activity feed
- `GET /api/v1/analytics/dependencies/:project` - Dependency graph

## 🎮 Workflow Examples

### Daily Game Development

```bash
# Check what teammates are working on
vcs status --team
# Output:
# 📝 alice: editing Characters/Hero.uasset (locked)
# 🎨 bob: working on Materials/Shader.umat
# 🔧 charlie: free

# Lock a level file before editing
vcs lock Levels/MainLevel.umap
# ✅ Locked Levels/MainLevel.umap

# Work in UE5 (plugin shows live lock status)
# - Red padlock = locked by others
# - Green checkmark = locked by you
# - Blue dot = someone viewing but not editing

# Commit and push specific assets
vcs add Levels/MainLevel.umap Characters/NewNPC.uasset
vcs commit -m "Added NPC spawn points to main level"
vcs push Characters/NewNPC.uasset  # Push just this asset
```

### Asset Management

```bash
# Add large binary assets (auto-chunked)
vcs add Assets/Meshes/HighPolyCharacter.fbx
# ⬆️ Uploading in chunks: [████████████████████████████████] 100%

# Check dependencies before deleting
vcs deps Assets/Textures/HeroSkin.uasset
# ⚠️  Warning: 15 assets depend on this texture

# View team productivity
vcs analytics --team --days 7
# 📊 Last 7 days:
#   alice: 23 commits, 45 files
#   bob: 18 commits, 32 files
```

## 🐳 Docker

```bash
# Build images
make docker-build

# Run server
make docker-run

# Or use Docker Compose (coming soon)
docker-compose up -d
```

## 📁 Project Structure

```
gamedev-vcs/
├── cmd/
│   ├── vcs/           # CLI tool main
│   └── vcs-server/    # Server main
├── internal/          # Internal packages (upcoming)
│   ├── storage/       # Content-addressable storage
│   ├── api/           # API handlers
│   ├── auth/          # Authentication
│   └── analytics/     # ClickHouse integration
├── docker/            # Docker configurations
├── k8s/               # Kubernetes manifests
├── web/               # Web dashboard (Elixir)
├── docs/              # Documentation
├── build/             # Build artifacts
├── Makefile           # Build automation
└── go.mod             # Go dependencies
```

## 🤝 Contributing

This is currently a private project for our game studio. Once we reach MVP, we'll consider open-sourcing components.

## 📋 Roadmap

### Week 1: Storage Foundation
- [ ] Content-addressable storage system
- [ ] Redis real-time state management
- [ ] Basic file operations (upload/download)
- [ ] File locking with Redis

### Week 2: Analytics & Features
- [ ] ClickHouse integration
- [ ] Asset dependency tracking
- [ ] Chunked binary uploads
- [ ] WebSocket real-time updates

### Week 3: UE5 Integration
- [ ] Basic UE5 plugin
- [ ] Asset browser integration
- [ ] Auto-lock on edit
- [ ] CLI communication

### Week 4: Polish & Deploy
- [ ] Web dashboard foundation
- [ ] Kubernetes deployment
- [ ] Production hardening
- [ ] Team testing

## 📝 License

Proprietary - YourStudio 2025


http://localhost:8080/api/v1/auth/google
http://localhost:8080/api/v1/auth/google/callback