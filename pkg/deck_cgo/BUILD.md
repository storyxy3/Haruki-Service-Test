# pkg/deck_cgo — Build Guide

## Overview

```
Go (engine.go + pool.go)
  │ CGo
  ▼
deck_c_api.h / deck_c_api.cpp     ← pure-C shim we wrote
  │ C++ calls
  ▼
sekai_deck_recommend.cpp           ← upstream C++ engine
  (NeuraXmy/sekai-deck-recommend-cpp)
```

No Python runtime required after the library is built.

---

## Step 1 — Clone the upstream C++ source

```powershell
# Inside pkg/deck_cgo/
git clone --recursive https://github.com/NeuraXmy/sekai-deck-recommend-cpp vendor/sekai-deck-recommend-cpp
```

---

## Step 2 — Build the shared library

### Windows (MSVC, recommended)

Requires: Visual Studio 2022 + CMake ≥ 3.15

```powershell
cd pkg\deck_cgo
cmake -B build -G "Visual Studio 17 2022" -A x64
cmake --build build --config Release
cmake --install build --config Release
```

Output: `pkg/deck_cgo/lib/windows_amd64/sekai_deck_recommend_c.dll`

### Windows (MinGW/MSYS2)

```bash
cd pkg/deck_cgo
cmake -B build -G "MinGW Makefiles" -DCMAKE_BUILD_TYPE=Release
cmake --build build
cmake --install build
```

### Linux (GCC ≥ 11)

```bash
cd pkg/deck_cgo
cmake -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build -- -j$(nproc)
cmake --install build
```

Output: `pkg/deck_cgo/lib/linux_amd64/libsekai_deck_recommend_c.so`

### macOS (Apple Clang / Homebrew GCC)

```bash
cd pkg/deck_cgo
cmake -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build -- -j$(sysctl -n hw.logicalcpu)
cmake --install build
```

---

## Step 3 — Copy the runtime library alongside the binary

On **Windows**, copy `sekai_deck_recommend_c.dll` next to `server.exe`.  
On **Linux/macOS**, either:
- Set `LD_LIBRARY_PATH` / `DYLD_LIBRARY_PATH`, or
- Run `ldconfig` after installing to a system lib path, or
- Use `rpath` (add `-Wl,-rpath,\$ORIGIN/lib` to CGo LDFLAGS).

---

## Step 4 — Build Go with CGo enabled

```powershell
# Windows
set CGO_ENABLED=1
go build -o server.exe ./cmd/server/...
```

```bash
# Linux / macOS
CGO_ENABLED=1 go build -o server ./cmd/server/...
```

---

## Cross-compilation note

CGo does **not** support true cross-compilation out of the box.  
Recommended CI strategy:
- **Windows**: build `sekai_deck_recommend_c.dll` on a Windows runner, commit to `lib/windows_amd64/`
- **Linux**: build `.so` on a Linux runner, commit to `lib/linux_amd64/`
- **macOS**: build `.dylib` on a macOS runner, commit to `lib/darwin_amd64/` and `lib/darwin_arm64/`

All pre-built libraries can be committed to the repository (they are small, ~2–5 MB).

---

## Disabling the CGo engine (fallback to HTTP)

If the library is not present, the service falls back to the existing HTTP-based
Python backend automatically. See `internal/service/deck_recommender.go`.

Set in `configs.yaml`:
```yaml
deck_recommend:
  use_local_engine: false  # force HTTP fallback
```
