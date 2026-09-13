# Windows 시스템 복구 도우미

비전문가도 버튼 한 번으로 Microsoft 권장 Windows 복구 절차를 실행할 수 있는 데스크톱 앱입니다.

앱은 관리자 권한으로 실행되며 다음 고정 명령을 순서대로 수행합니다.

```text
DISM.exe /Online /Cleanup-Image /RestoreHealth
sfc.exe /scannow
```

사용자가 입력한 명령을 실행하는 기능은 없습니다. 개인 파일과 설치된 앱을 삭제하지 않습니다.

화면의 **빠른 지원 열기** 버튼은 Windows 11의 Microsoft 빠른 지원 앱을 실행합니다. 빠른 지원 앱이 설치되어 있지 않으면 [Microsoft Store](https://apps.microsoft.com/detail/9p7bp5vnwkx5)에서 설치할 수 있습니다.

Windows 업데이트가 실패하거나 멈춘 경우에는 다음 2단계 흐름을 제공합니다.

1. **업데이트 문제 해결**: Windows 11 도움말 보기 앱의 Microsoft 자동 문제 해결사를 실행합니다.
2. **업데이트 캐시 초기화**: 자동 문제 해결로 해결되지 않을 때만 BITS, Windows Update, 암호화 서비스를 잠시 중지하고 `SoftwareDistribution`과 `catroot2` 폴더를 날짜가 붙은 백업 이름으로 변경합니다. 서비스는 작업 성공 여부와 관계없이 원래 상태로 복구합니다.

## 기술 구성

- Go 백엔드
- Wails v2 데스크톱 셸
- Vite + 순수 HTML/CSS/JavaScript UI
- Windows 10/11 및 WebView2 Runtime

Next.js는 서버 렌더링과 라우팅이 필요 없는 단일 화면 앱에는 빌드 크기와 복잡성만 늘리므로 사용하지 않았습니다.

## 개발 환경

Windows에서 아래 항목을 설치합니다.

1. Go 1.25 이상
2. Node.js 20 이상
3. WebView2 Runtime (Windows 10/11에는 일반적으로 설치되어 있음)
4. Wails CLI

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.14.0
wails doctor
```

## 실행 및 빌드

```powershell
wails dev
wails build -clean
```

완성된 실행 파일은 `build\bin\Windows System Repair Helper.exe`에 생성됩니다. 실행 시 Windows 사용자 계정 컨트롤(UAC)이 관리자 권한을 요청합니다.

## 무설치 단일 파일 배포

다음 명령은 별도 설치 없이 바로 실행할 수 있는 Windows 단일 파일을 만듭니다.

```powershell
wails build -clean -webview2 embed
```

`build\bin\Windows System Repair Helper.exe` 파일 하나만 전달하면 됩니다. 설치 과정이나 별도 파일 복사가 필요하지 않습니다. Windows 11에 WebView2가 없거나 너무 오래된 경우에는 앱에 내장된 Microsoft 부트스트랩을 통해 자동 설치를 안내하며, 이때만 인터넷 연결이 필요합니다.

## 앱 업데이트 배포

앱은 실행할 때마다 백그라운드에서 `mireu-san/windows-troubleshoot-tool`의 최신 공개 정식 GitHub Release를 조회합니다. 현재 앱 버전은 제목 옆에 표시합니다. 새 버전이 있으면 파일을 자동으로 다운로드하고 GitHub API의 SHA-256 digest와 비교한 뒤 **새 버전을 설치하시겠습니까?** 안내를 표시합니다.

**설치하기**를 누르면 앱을 종료하고 새 EXE로 교체한 뒤 다시 실행합니다. **나중에**를 누르면 현재 앱을 계속 사용할 수 있으며, 화면의 업데이트 설치 버튼으로 나중에 설치할 수 있습니다. 검사·복구 작업 중에는 설치를 차단합니다. EXE 교체 또는 새 프로세스 시작이 실패하면 이전 파일로 복원합니다. 설치 파일만 첨부된 릴리스는 해당 설치 프로그램을 실행합니다.

다운로드에 실패하거나 검증 정보가 없거나 일치하지 않으면 설치하지 않습니다. 자동 업데이트를 위해 GitHub Release의 실행 파일 첨부가 필요합니다.

현재 기본 앱 버전은 `2026.09.13`입니다. 새 버전 배포 시 Windows에서 버전을 지정하여 빌드합니다.

```powershell
wails build -clean -platform windows/amd64 -webview2 embed -ldflags "-X main.appVersion=2026.09.13"
```

1. 변경 사항을 커밋하고 GitHub에 push합니다.
2. 저장소의 Releases에서 `v2026.09.13` 태그로 정식 릴리스를 생성합니다. 버전은 빌드에 지정한 값과 일치해야 하며 `vYYYY.MM.DD` 또는 `vYYYY.MM.DD.N` 형식을 사용합니다. 같은 날 추가 배포 시 마지막 번호를 증가시킵니다.
3. `build/bin/WindowsSystemRepairHelper.exe`를 릴리스 첨부 파일로 업로드하고 게시합니다. 무설치 EXE가 우선 선택됩니다. 설치 파일만 배포하는 경우 `WindowsSystemRepairHelper-amd64-installer.exe` 이름을 사용합니다. 이 배포 규칙은 Windows x64용입니다.
4. 이전 버전의 앱을 실행하여 자동 업데이트 확인 후 자동 다운로드 후 설치 안내가 표시되는지 확인합니다.

단순한 소스 push나 태그 생성만으로 업데이트가 배포되지는 않습니다. 정식 Release와 첨부 실행 파일이 필요합니다. 비공개 저장소는 인증 없이 조회할 수 없으며, 앱에는 GitHub 토큰을 포함하지 않습니다. 릴리스가 없거나 통신에 실패해도 앱 사용을 방해하지 않습니다. 다음 실행 시 다시 확인합니다. 새 버전의 파일이 준비되지 않은 경우에는 배포 내역 확인을 안내합니다.

API 동작은 [GitHub Releases 공식 문서](https://docs.github.com/en/rest/releases/releases#get-the-latest-release)를 따릅니다.

## 공식 참고 자료

- [Microsoft: 시스템 파일 검사기 도구를 사용하여 누락되거나 손상된 시스템 파일 복구](https://support.microsoft.com/ko-kr/windows/experience/backup-recovery/use-the-system-file-checker-tool-to-repair-missing-or-corrupted-system-files)
- [Wails installation guide](https://wails.io/docs/gettingstarted/installation/)

## Defender 빠른 검사

**Defender 빠른 검사** 버튼은 Microsoft 공식 [Start-MpScan](https://learn.microsoft.com/en-us/powershell/module/defender/start-mpscan) 명령의 `-ScanType QuickScan`을 실행합니다. 검사 중 중복 실행과 앱 내 다른 복구 작업의 동시 실행을 막습니다. Defender 설정이나 제외 항목은 변경하지 않습니다. 위협 탐지 및 조치 결과는 **Windows 보안 · 검사 결과 보기**에서 확인합니다. Defender가 비활성화되어 있거나 검사 명령이 실패하면 오류를 안내합니다.

검사 및 복구 중 **검사 및 복구 취소**를 누르면 확인 후 실행 중인 DISM/SFC 명령을 중단하며 다음 단계로 넘어가지 않습니다. 이미 적용된 변경은 되돌리지 않습니다.

## 라이선스

이 프로젝트의 자체 작성 코드는 [MIT License](LICENSE)로 공개합니다. Copyright (c) 2026 Ayin Kim. 저작권 및 라이선스 고지를 유지하는 조건으로 상업적 이용, 수정, 재배포를 허용하며, 소프트웨어는 보증 없이 제공됩니다.

“비상업적 목적으로 개발”은 개발 취지이며 이용 제한이 아닙니다. 외부 라이브러리와 Microsoft 구성 요소에는 각자의 라이선스가 적용됩니다.

## 복구 후 자동 종료

검사 시작 전에 **DISM과 SFC 모두 성공하면 5분 뒤 컴퓨터 종료**를 선택하고 확인하면, 두 단계 모두 성공한 경우에만 Windows에 종료를 예약합니다. 기본값은 꺼짐이며 실행 중에는 변경할 수 없습니다. 복구 실패 또는 취소 시에는 예약하지 않습니다. 예약 실패는 복구 완료 화면에 별도로 표시합니다.

Windows의 `shutdown.exe /s /t 300`을 사용합니다. [Microsoft 문서](https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/shutdown)에 따라 0보다 큰 대기 시간을 지정하면 앱 강제 종료가 적용되므로, 저장하지 않은 내용이 손실될 수 있습니다. 실행 전 파일을 저장하세요. 앱을 닫아도 예약은 유지됩니다. **컴퓨터 종료 예약 취소** 버튼은 `shutdown.exe /a`를 실행하며, 앱을 다시 열어도 사용할 수 있습니다. 복구 중 누르면 이번 복구 후 자동 종료 설정도 해제합니다. 이 명령은 다른 프로그램이나 사용자가 설정한 Windows 종료 예약도 취소할 수 있습니다. Windows에서 이미 취소한 경우에도 이 버튼으로 앱의 예약 상태를 해제할 수 있습니다.

앱이 종료 예약을 유지하는 동안 새 검사·복구와 앱 업데이트 설치를 차단합니다. 계속 작업하려면 종료 예약을 취소하세요.

### 버전 비교 및 배포

이번 릴리스는 `v2026.09.13`으로 배포합니다. 이전 `2026.09.12` 앱에서도 이 세 자리 버전을 인식할 수 있습니다. 현재 앱은 세 자리와 네 자리 버전을 모두 숫자로 비교하며 생략된 네 번째 자리는 0으로 취급합니다. 버전 비교는 PC 날짜·시간을 사용하지 않습니다.

이번 버전 빌드 명령:

```powershell
wails build -clean -platform windows/amd64 -webview2 embed -ldflags "-X main.appVersion=2026.09.13"
```

정식 Release 태그는 `v2026.09.13`로 만들고 빌드한 `WindowsSystemRepairHelper.exe`를 첨부합니다.

## 배포 파일명

빌드 결과는 자동으로 `build/bin/Windows System Repair Helper.exe`라는 이름으로 생성됩니다. 수동 다운로드용으로 이 파일을 Release에 첨부합니다. 기존 앱의 자동 업데이트 호환성을 위해 같은 파일을 `WindowsSystemRepairHelper.exe`라는 이름으로 복사하여 함께 첨부해야 합니다. 두 파일의 내용은 동일합니다. 자동 업데이트는 기존 실행 파일 경로를 유지하므로 이미 설치된 파일명은 바뀌지 않습니다.

```powershell
Copy-Item 'build/bin/Windows System Repair Helper.exe' 'build/bin/WindowsSystemRepairHelper.exe'
```
