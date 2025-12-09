# Universal ASDF Plugin 🚀

> ⚠️ **주의:** 이 번역은 기계 번역으로 생성되었습니다. 부정확한 부분을 발견하시면 PR로 수정 제안을 보내 주세요.

[![Test](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml/badge.svg)](https://github.com/sumicare/universal-asdf-plugin/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/sumicare/universal-asdf-plugin/graph/badge.svg)](https://codecov.io/gh/sumicare/universal-asdf-plugin)
[![Go Report Card](https://goreportcard.com/badge/github.com/sumicare/universal-asdf-plugin)](https://goreportcard.com/report/github.com/sumicare/universal-asdf-plugin)
[![License](https://img.shields.io/github/license/sumicare/universal-asdf-plugin)](../LICENSE)

**번역본 🌐:** [English](../README.md) • [Українська](./README.UA.md) • [Français](./README.FR.md) • [Deutsch](./README.DE.md) • [Polski](./README.PL.md) • [Română](./README.RO.md) • [Čeština](./README.CS.md) • [Norsk](./README.NO.md) • [日本語](./README.JA.md)

[asdf](https://asdf-vm.com) 플러그인을 Go로 구현한 통합 컬렉션입니다. 여러 저장소에 흩어져 있는 전통적인 Bash 기반 플러그인 대신, 하나의 테스트된 유지 보수 가능한 단일 바이너리로 대체합니다.

## 왜 필요할까요 ❓

- 🔐 **보안** — 여러 리포지토리에 흩어진 Bash 플러그인은 유효한 공격 표면이 됩니다.
- ✅ **신뢰성** — Go는 테스트 코드 작성과 재현 가능한 빌드를 쉽게 만들어 줍니다.
- 🧰 **유지 보수성** — 60개가 넘는 도구를 위한 단일 코드베이스이므로, 각기 다른 규칙을 가진 개별 플러그인을 관리할 필요가 없습니다.

## 빠른 시작 🚀

```bash
# 1. 최신 릴리스 다운로드
curl -LO https://github.com/sumicare/universal-asdf-plugin/releases/latest/download/universal-asdf-plugin-linux-amd64.tar.gz
tar -xzf universal-asdf-plugin-linux-amd64.tar.gz
chmod +x universal-asdf-plugin

# 또는 Go로 설치 (Go 1.24+ 필요)
go install github.com/sumicare/universal-asdf-plugin@latest

# 2. asdf 부트스트랩 (버전 관리자인 asdf 자체 설치)
universal-asdf-plugin install-plugin asdf
universal-asdf-plugin install asdf latest

# 3. 셸 설정 (~/.bashrc, ~/.zshrc 등에 추가)
export PATH="${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH"

# 4. 셸을 다시 시작한 후, 필요한 플러그인 설치
universal-asdf-plugin install-plugin
```

설정이 끝나면 asdf를 통해 도구를 관리할 수 있습니다:

```bash
asdf install go latest
asdf install nodejs latest
asdf global go latest
```

## 사용법 🧪

```bash
# 사용 가능한 버전 목록 보기
universal-asdf-plugin list-all <tool>

# 특정 버전 설치
universal-asdf-plugin install <tool> <version>

# 최신 안정(stable) 버전 조회
universal-asdf-plugin latest-stable <tool>

# 도구별 도움말 보기
universal-asdf-plugin help <tool>

# .tool-versions 파일을 최신 버전으로 갱신
universal-asdf-plugin update-tool-versions
```

## 개발 🛠️

### 사전 요구 사항

- Go 1.24+
- Docker (Dev Container용)

### 시작하기

```bash
# 리포지토리 클론
git clone https://github.com/sumicare/universal-asdf-plugin.git
cd universal-asdf-plugin

# Dev Container가 설정된 VS Code 워크스페이스 열기
code universal-asdf-plugin.code-workspace

# 로컬 빌드
./scripts/build.sh
```

### 테스트 실행

```bash
# golden 파일 업데이트
./scripts/test.sh --update

# 실제 패키지를 다운로드하여 전체 테스트 실행
./scripts/test.sh --online

# mock 서버를 사용한 모든 스모크 테스트 실행
./scripts/test.sh

# mutation 테스트 실행
./scripts/mutation-test.sh

# 린트
./scripts/lint.sh

# 맞춤법 검사
npm install -g cspell
./scripts/spellcheck.sh
./scripts/spellcheck_add.sh
# 그런 다음 .code-workspace의 사용자 사전을 확인하세요.
```

## 라이선스 📄

Copyright 2025 Sumicare

이 프로젝트를 사용함으로써 Sumicare OSS의 [이용 약관](./OSS_TERMS.KO.md)에 동의하는 것으로 간주됩니다.

이 프로젝트는 [Apache License, Version 2.0](../LICENSE)의 적용을 받습니다.
