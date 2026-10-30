# Проверка прототипа · 30 октября 2026

## Выполнено

- Go 1.26.10, Linux amd64; Wails Go module / CLI / npm runtime **v3.0.0**.
- **14 backend-тестов**, включая config preservation для REALITY/XHTTP/RAW/TCP, запрет лишних top-level features, traversal, хранение, отмену подготовки, повторные Connect/Disconnect, Wait ровно один раз, очистку временных конфигов, занятые порты, скрытие секретов в ошибках, монотонные revision, периодические проверки без параллельных проб.
- `go test -race -count=2 ./internal/...` — успешно после миграции.
- Реальный дочерний **тестовый** процесс запускался и отменялся через execRunner. Это проверка процесса, а не Xray.
- `go vet ./internal/...` — успешно.
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet -tags production ./...` — успешно.
- Windows amd64 cross-compilation backend-тестов — успешно; запуск этих тестов на Windows не выполнялся.
- **14 frontend-тестов** (Vitest + jsdom + Testing Library) — успешно: честный browser-only режим, разделённые статусы, импорт/отмена/очистка JSON, подтверждение удаления, защита от повторного Connect, ошибки bridge, отмена подготовки, обязательный путь, устаревшие ответы и cleanup polling; 3 новых теста проверяют v3 bindings/ID/аргументы, nil-срезы, ошибки и отсутствие backend в браузере.
- `npm run typecheck`, `npm test`, `npm run build` — успешно: TypeScript + Vitest + Vite production.
- `wails3 build` — Windows GUI **cross-build** всего приложения с `-tags production`, `-H windowsgui`, `CGO_ENABLED=0` — успешно. Команда регенерирует bindings, выполняет `npm ci` и frontend build, генерирует Windows ресурсы (asInvoker, DPI, icon) и собирает exe. Это не запуск GUI.
- `go version -m bin/XrayMini.exe` подтверждает Go 1.26.10, Wails v3.0.0, production, windows/amd64; `file` подтверждает PE32+ GUI x86-64.
- Go module/imports и bindings соответствуют репозиторию `github.com/slonikmak/XRay-mini`; после переименования повторены все перечисленные проверки.
- Генератор v3 обнаружил **1 service, 7 methods, 3 models, 1 event**; lifecycle-методы не экспортируются во frontend.

Воспроизводимая cross-сборка (после установки CLI `v3.0.0` и Go 1.26.10):

```sh
wails3 build
```

Проектный Taskfile нацелен на Windows amd64 даже на Linux-host. Он использует production-флаги официального v3.0.0 Windows template. Без `production` Wails v3 собирает dev-вариант — не используйте его как релизный exe. Утверждение предыдущей v2-версии о «служебной заглушке» к v3 не относится.

В этой Linux-среде CLI был собран без GTK через `CGO_ENABLED=0 go install -tags server github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0`. Это только **CLI**, не приложение: у поставляемого Windows exe нет тега `server`. Обычная установка CLI на Windows приведена в README.

SHA-256 `bin/XrayMini.exe`:

```text
007872b25429e64a8f8f2b7cf9df1931b125ea478cd18f11077441ec4cbe68f5
```

## Не проверено

- Запуск UI/WebView2 на Windows, DPAPI/DACL/Job Object во время исполнения.
- Реальный Xray Core, настоящий сервер, REALITY handshake и XHTTP end-to-end.
- Настоящий внешний health-check: в тестах сеть подменена безопасной тестовой реализацией.
- Визуальный браузерный прогон: Chromium не запустился из-за ограничений среды; нет утверждения о screenshot/visual QA.
- Установщик, подпись exe, обновления, антивирусная совместимость, DPI и доступность UI на реальном устройстве.

## Что проверить на своей Windows-машине

1. Запустить приложение без прав администратора; убедиться, что список профилей доступен.
2. Импортировать собственный доверенный JSON, закрыть и открыть приложение: профиль должен сохраниться.
3. Проверить отдельно RAW+REALITY и XHTTP+REALITY с вашим Xray и сервером.
4. Нажать Connect, быстро Cancel startup, затем Connect повторно.
5. Проверить отсутствие xray.exe после Disconnect и закрытия окна.
6. Проверить конфликт портов, ошибочный конфиг, недоступный сервер, потерю сети.
7. Настроить только выбранное приложение на локальный прокси и проверить его трафик.
8. Убедиться в ACL каталога и удалить остаточные `runtime-*` после аварийного завершения, если они есть.

Только в скачиваемом архиве есть неподписанный `bin/XrayMini.exe`, собранный из этих исходников. В Git-репозитории exe не хранится. Xray Core не включён. Если Windows предупреждает о неизвестном издателе, не отключайте системную защиту ради прототипа: можно собрать исходники самостоятельно в доверенной среде.

## Основание миграции

- [Wails v3.0.0 release, 26 октября 2026](https://wails.io/blog/wails-v3-released/).
- [Официальный changelog](https://v3.wails.io/changelog/) и исходники опубликованного Go-модуля `github.com/wailsapp/wails/v3@v3.0.0`: service lifecycle, runtime, React template, Windows build tasks и manifest.
- `internal/core` не изменялся при миграции. Настоящий Xray, внешнее подключение и изменение системной сети не выполнялись.
