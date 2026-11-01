<!-- bmad:context -->
<!-- Verified 2026-10-31 against 910c353955f02b071fb545b0512b7585cca85189. Managed by bmad-project-context. -->

## Где искать требования

- При изменении подключения, профилей, диагностики или поставки читать FR/ADR в `docs/xray-mini-spec-v0.2.ru.md`; результаты сверки и незавершённая приёмка связаны в `docs/BROWNFIELD.md`.
- Исторический аудит `docs/xray-mini-audit-v0.2.ru.md` относится к 50b8cb3; текущие выводы проверять по коду нужного commit.

## Границы ответственности

- Состояние сессии менять только в `Service.loop`; worker передаёт события с sessionID, длительный I/O выполняет вне loop. Освобождение active связывать со StopCompleted после cleanup.
- UI получает состояние из Go snapshot/events и соблюдает revision ordering; локальный выбор профиля не меняет активную сессию.
- Для запуска Xray сохранять общий путь lock → pinned digest validation → contained creation для preflight и runtime; весь пользовательский трафик обрабатывает отдельный core.

## Сборка и свидетельства

- Использовать toolchain из `go.mod` (Go 1.26.10, Wails 3.0.0). Для bindings выполнять `wails3 task bindings`; источник генерации — `Taskfile.yml`, изменения generated bindings получать генератором.
- Во время `wails3 build` не запускать параллельно npm install/test: build сам вызывает npm ci и генерацию frontend.
- Поставлять весь `bin/` с проверенным runtime и notices, как описано в README; один GUI exe не является полным package.
- Заявляя PASS, указывать app commit, среду и фактический результат. Linux fixture, mock tests и cross-build не закрывают Windows/DPAPI/DACL/Job Object/WebView2 acceptance из `docs/VERIFICATION.md`.

<!-- /bmad:context -->
