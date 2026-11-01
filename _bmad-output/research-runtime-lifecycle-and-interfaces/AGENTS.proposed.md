# Согласованный блок корневого AGENTS.md

Намерение bmad-project-context: **setup**. В корне и поддеревьях AGENTS.md/CLAUDE.md не
обнаружены. Предлагается один корневой блок для Go и frontend; отдельных дочерних
инструкций не создаётся. Внешних governance-правил для Xray Mini пользователь не задавал;
правило TopHunter staging не относится к этому проекту. Существующие инструкции не
переносятся и не удаляются, поэтому ledger миграции пуст.

Пользователь согласовал полный блок 2026-11-01. Он установлен без изменения текста
в [корневой AGENTS.md](../../AGENTS.md). Ниже сохранён согласованный текст и его обоснование.

```markdown
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
```

## Почему эти строки нужны

| Строки | Проверенное основание | Почему инструкция полезна |
|---|---|---|
| Требования и исторический аудит | Spec пользователя; audit header 50b8cb3; research 910c353 | Предотвращает перенос старых дефектов и объявление непроверенной приёмки. |
| Ownership/UI | Независимо проверенные refs 4/5/15/16 | Правила пересекают manager, worker, facade и frontend. |
| Runtime trust path | Независимо проверенные refs 8/9 | Частичное изменение одного запуска может обойти pin/containment. |
| Toolchain/bindings | go.mod:3–7; Taskfile.yml:8–14 | Windows-target generation и закреплённые версии важны для правильного входа в build. |
| Build concurrency | Taskfile frontend → bindings/npm ci/build; VERIFICATION предупреждает о параллельных npm | Полный build изменяет ту же рабочую frontend-среду. |
| Package/evidence | refs 11/18–20; VERIFICATION | Граница поставки и область доказательств проходят через несколько компонентов. |

Предложений удалить существующие правила нет. Hook/lint не заменяет источники требований
или честное обозначение среды проверки. Автоматизацию проверки комплектности package и
расширение result metadata можно рассматривать отдельно; здесь конфигурация CI не меняется.
Блок загружается из корневого AGENTS.md; обновлять после значимых
изменений указанных контрактов. Документ не содержит branch/commit policy: такая политика
для Xray Mini пользователем не задавалась.
