# Docksmith

A simplified Docker-like container system implemented in Go.

## Overview

Docksmith is a minimal container runtime and build system that demonstrates core containerization concepts:

- Image building from Docksmithfiles
- Layer-based filesystem with caching
- Process isolation using Linux primitives
- Image management

## Architecture

### Components

1. **CLI Layer** (`cmd/`) - Command-line interface using Cobra
2. **Build Engine** (`build/`) - Parses Docksmithfiles and creates layered images
3. **Runtime** (`runtime/`) - Executes containers with isolation
4. **Layers** (`layers/`) - Manages filesystem layers as tar archives
5. **Images** (`images/`) - Handles image manifests and metadata
6. **Cache** (`cache/`) - Implements build cache for faster rebuilds
7. **Util** (`util/`) - Shared utilities for hashing, tar operations, etc.

### Directory Structure

```
~/.docksmith/
├── images/      # Image manifests (JSON files)
├── layers/      # Layer tar archives (sha256_<hash>.tar)
└── cache/       # Build cache index
```

## Docksmithfile Syntax

Supported instructions:

```dockerfile
FROM <image[:tag]>          # Load base image layers
COPY <src> <dest>           # Copy files from build context (creates layer)
RUN <command>               # Execute command (creates layer)
WORKDIR <path>              # Set working directory (config only)
ENV <key=value>             # Set environment variable (config only)
CMD ["exec", "arg", ...]    # Set default command (config only)
```

### Example Docksmithfile

```dockerfile
FROM python:3.11-slim
WORKDIR /app
COPY . /app
ENV MSG=Hello
RUN python setup.py install
CMD ["python", "main.py"]
```

## Commands

### Build an Image

```bash
docksmith build -t myapp:latest ./my-app
```

Builds an image from the Docksmithfile in the `./my-app` directory.

Build features:
- Layer caching based on instruction content and dependencies
- Deterministic layer creation with SHA256 digests
- Isolated RUN execution

### Run a Container

```bash
# Use default CMD from image
docksmith run myapp:latest

# Override CMD
docksmith run myapp:latest /bin/sh
```

Runtime features:
- Linux namespace isolation (CLONE_NEWNS, CLONE_NEWPID, CLONE_NEWUTS)
- chroot filesystem isolation
- Environment variable injection
- Working directory configuration

### List Images

```bash
docksmith images
```

Shows all built images with their tags, digests, and layer counts.

### Remove an Image

```bash
docksmith rmi myapp:latest
```

Deletes an image manifest (layers are retained).

## Building Docksmith

### Prerequisites

- Go 1.21 or later
- Linux (for full isolation features)

### Build

```bash
# Clone or navigate to the project directory
cd CC_Project

# Download dependencies
go mod download

# Build the binary
go build -o docksmith .

# Install to PATH (optional)
go install
```

## Example Applications

### Simple Shell App

Located in `examples/simple-app/`:

```bash
cd examples/simple-app
docksmith build -t simple:latest .
docksmith run simple:latest
```

### Python App

Located in `examples/python-app/`:

```bash
cd examples/python-app
docksmith build -t pyapp:latest .
docksmith run pyapp:latest
```

## Technical Details

### Layer Storage

Layers are stored as tar archives with:
- Deterministic ordering (sorted file paths)
- Zero timestamps for reproducibility
- SHA256 digest as filename: `sha256_<hash>.tar`

### Build Cache

Cache keys are computed from:
- Previous layer digest
- Instruction text
- Current working directory
- Environment variables (sorted)
- Source file hashes (for COPY)

Cache hits avoid re-executing expensive operations.

### Image Manifest

Stored as JSON in `~/.docksmith/images/<name>_<tag>.json`:

```json
{
  "name": "myapp",
  "tag": "latest",
  "digest": "sha256:abc123...",
  "created": "2026-03-12T10:30:00Z",
  "config": {
    "Env": ["MSG=Hello"],
    "Cmd": ["python", "main.py"],
    "WorkingDir": "/app"
  },
  "layers": [
    {
      "digest": "sha256:def456...",
      "size": 12345,
      "createdBy": "COPY . /app"
    }
  ]
}
```

### Isolation

On Linux, containers are isolated using:
- **Namespace isolation**: `unshare --fork --pid --mount --uts`
- **Filesystem isolation**: `chroot` to container rootfs
- **Process isolation**: Separate PID namespace

Fallback: On non-Linux systems, basic process execution without full isolation.

## Limitations

This is a simplified educational implementation. Notable limitations:

- No networking support
- No volume mounting
- No registry pull/push
- Layer deduplication is minimal
- No user namespace isolation
- No resource limits (cgroups)
