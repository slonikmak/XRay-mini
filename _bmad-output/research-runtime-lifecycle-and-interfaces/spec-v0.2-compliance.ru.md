# Xray Mini — сверка со спецификацией v0.2

**Дата:** 2026-10-31. **Код:** `910c353955f02b071fb545b0512b7585cca85189`, `main`.
**Вывод:** основные механизмы MVP присутствуют в коде; завершение приёмки не подтверждено.
Главный пробел — фактическая проверка Windows 10/11 и поставляемого приложения.
Есть также условный дефект восстановления после ошибки cleanup, ограничения shutdown
и диагностики, перечисленные ниже.

## Основание и границы

Требования взяты из переданного пользователем корневого
`xray-mini-spec-v0.2.ru.md`, SHA-256
`261d26ee0a1b57139a6ae1d02711338bb9feb55abee69287ac0436ab8e07df5e`.
Версия в `docs/` совпадает после нормализации CRLF/LF; исходные байты и файлы сохранены.

Источник фактов о реализации — [исследование кода](research-runtime-lifecycle-and-interfaces.md)
с шестью вопросами, исходными ссылками и независимой проверкой ключевых выводов.
В таблицах Q1–Q6 означают соответствующие разделы исследования; точные строки можно
проверить там. Состояние незакоммиченных generated bindings исключено из оценки.
Код, документация и сохранённые результаты прочитаны; новые тесты и приложение не запускались.

Старый [аудит](../../docs/xray-mini-audit-v0.2.ru.md) проверял `50b8cb3`.
Его замечания не являются свидетельством текущего дефекта. Текущий код содержит
существенные исправления импорта, маршрутизации, manager loop, доверия бинарнику,
Job Object, health и UI/API; таблица ниже проверяет их по текущим исходникам.

**Обозначения:** «по коду» — механизм найден, runtime-проход не заявлен;
«частично» — конкретная непокрытая часть; «не подтверждено» — нет нужного результата
исполнения. Это не оценка процента готовности.

## Остаточные случаи и решения для следующей работы

| ID | Установленный по коду случай | Следствие и действие |
|---|---|---|
| F1 cleanup error | `RemoveAll` error → StopCompleted → active=nil → Error, следующий Connect разрешён (Q2, refs6). | Частично расходится с требованием завершённого cleanup перед повторным Connect. В отдельном исправлении определить recovery/retry для не удалённого runtime; проверить отказ удаления и сохранение ошибки. В этом исследовании отказ не воспроизводился. |
| F2 shutdown boundaries | Синхронный callback publisher без deadline; Close ждёт publisher. Windows Wait бесконечен, Kill errors игнорируются (Q2/Q3, refs7/10). | Зависший callback может удержать Close; успешное прекращение процесса при отказе Kill не гарантируется этой логикой. Отдельно определить поведение/диагностику shutdown при этих отказах и проверить bounded completion. Частоту проблемы статический анализ не устанавливает. |
| F3 startup diagnostics | Если Wails оставляет bridge callable после ServiceStartup error, failed ListProfiles отменяет применение fallback status из Promise.all (Q5, ref17). | UI показывает generic sync warning вместо причины. Уточнить фактический native lifecycle, затем при необходимости показать безопасную причину. Открытие окна после startup error этим кодом не доказано. |
| F4 log refresh | Diagnostic-only event меняет logSnapshot без status notification (Q5). | Новые категории видны после следующего status/focus/action refresh, а не сразу; это ограничение диагностической панели, точный latency spec не задаёт. |
| F5 package failure | Проверенный exe заменяется раньше копирования license/notice (Q3, ref11). | Поздняя ошибка может оставить неполную папку поставки. При release packaging валидировать весь комплект или сделать staging/atomic promotion; недостающая подпись/installer не объявляются MVP-дефектами. |
| E1 evidence | Saved Linux JSON содержит engine/time/pins, но не app source commit (Q6, ref20). | Результат нельзя объявить повторным прогоном текущего app snapshot. Следующий реальный прогон должен записать app revision, среду, hashes, команду, outcomes и cleanup. |

## Функциональные требования

| Требование | Результат | Свидетельства и граница |
|---|---|---|
| FR-01 профили | По коду | Q1/Q2/Q5: импорт, metadata list, выбор, rename/delete; active-session guard. Импорт — вставка JSON; чтение исходного файла приложением не требуется текущим FR. |
| FR-02 импорт | По коду | Q1: ровно один VLESS со streamSettings, preview и подтверждение отброшенных секций; неоднозначность/зависимости/дубликаты отклоняются. Настройки обязательных транспортов сохраняются. Подтверждение — UI-контракт; backend можно вызвать отдельно без acknowledgment. |
| FR-03 runtime | По коду | Q1/Q3: отдельный JSON, единственный outbound, явные routes, preflight `run -test -config`. В коде используется форма `-config`; сетевой успех устанавливается отдельной проверкой, не preflight. |
| FR-04 proxy | По коду; Windows не подтверждён | Q1/Q4: 127.0.0.1:10808/10809, проверка обоих портов, без direct outbound. Сохранённые Linux проверки используют ephemeral ports и не доказывают Windows fixed-port runtime. |
| FR-05 сессии | По коду; исключение cleanup | Q2: duplicate guard, context cancel, уникальный sessionID, StopCompleted barrier, stale-event rejection. Ошибка удаления каталога всё же освобождает active — F1. |
| FR-06 health | По коду | Q4: реальный HTTPS через HTTP proxy, 5с/30с, serial checks, 2 failures → Degraded, success → Connected; без restart. Постоянного SOCKS health-probe нет; оба proxy проверены сохранённой Linux fixture. |
| FR-07 диагностика | По коду с ограничением | Q4/Q5: безопасные сообщения, pinned version, время проверки, 200 backend/100 UI log entries. Xray-текст отбрасывается; видны только категории наличия stdout/stderr. Условие startup error — F3, обновление журнала — F4. |
| FR-08 UI | По коду; native UX не подтверждён | Q5/Q6: одно окно, profiles/actions/status/addresses/diagnostics; подтверждение приёма команды без ожидания I/O, event/snapshot revision guards; 20 mock/jsdom test definitions. Нет результата WebView2/окна на обеих Windows. |
| FR-09 завершение | Частично | Q2/Q3: kill/wait/join, runtime cleanup, startup recovery под exclusive lock, atomic Job Object. F1/F2 ограничивают cleanup/shutdown; аварийные сценарии Windows не подтверждены. |

## Architecture Decision Rules

| Правило | Результат | Свидетельства и граница |
|---|---|---|
| ADR-001 отдельный core | По коду | Q3: отдельный pinned child, preflight, executable digest; движок в Go не встроен. |
| ADR-002 один backend | По коду | Q3/Q5: Wails service/API/events в Go-приложении; пользовательский трафик обрабатывает Xray. |
| ADR-003 single writer | По коду | Q2: channel/select loop, atomic snapshots, I/O вне owner loop. Это уже не mutex-архитектура старого аудита. |
| ADR-004 session resources | По коду; F1 | Q2: sessionID/context, tagged events, late-start cleanup, waiter до ProcessStarted; removal error остаётся исключением. |
| ADR-005 supervisor | По коду | Q2/Q3: один waiter, idempotent Windows Wait, drains joined. Ошибки Kill не эскалируются, post-kill wait без deadline — граница F2. |
| ADR-006 Job Object | По коду; native не подтверждён | Q3: JOB_LIST при CreateProcessW, kill-on-close, restricted handles для обоих запусков. Helper crash test существует, фактический Windows результат не найден. |
| ADR-007 process/health | По коду | Q4: отдельно processState/networkHealth, 2 failures, recover/reset. После первого отказа state может быть Connected при unhealthy; UI это показывает. |
| ADR-008 backend truth | По коду; native reopen не подтверждён | Q5: subscribe-before-snapshot, revision ordering, focus/visibility refresh. |
| ADR-009 secrets | По коду; native не подтверждён; F1 | Q1/Q2: user DPAPI, ACL текущего пользователя и SYSTEM, приватный runtime, crash residue cleanup после lock. Удаление не является secure erase. |
| ADR-010 bounded tasks | По коду; F2 | Q2/Q4: fixed workers, bounded/coalescing publisher, serial health, bounded logs. Возврат callback необходим для завершения publisher. |
| ADR-011 local proxy | По коду | Q1/Q3: loopback proxy; TUN/системные proxy-настройки не реализуются. Новые маршруты/kill switch не требуются. |
| ADR-012 compatibility | Частично по приёмке | Q3/Q6: pin и digest реализованы, сохранены 12 Linux fixture PASS; Windows transport/packaged-runtime результаты отсутствуют, app commit в старом JSON не записан. |

## Контракт §5–6 и нефункциональные требования §7

| Область | Результат |
|---|---|
| API | Q5: list/import/rename/delete/connect/disconnect/status/recent logs присутствуют; дополнительно PreviewImport. Import принимает имя и JSON, Connect — profileID. Метаданные и status не возвращают конфигурацию. |
| События | Q2/Q4/Q5: ConfigValidated/ProcessStarted/ProcessExited/ProxyReady/HealthFailed/StopCompleted плюс Diagnostic; sessionID, status revision; connection:state-changed. |
| Состояния и сценарии | Q2/Q4: требуемые отображаемые states, подтверждение приёма команды, serial preparation/health, stop barrier. При ошибке удаления файл может остаться после Error — F1. |
| Windows, normal user, WebView2 | Q3: Windows amd64 GUI, manifest asInvoker; WebView2 — prerequisite в README. Поведение без WebView2 и работа без elevation на Win10/11 не подтверждены. Лицензия/notice поставляются, возможен частичный package при поздней ошибке — F5. |
| Отзывчивость | Q2/Q5: I/O вне loop и подтверждение приёма команды без ожидания подготовки. Native responsiveness, focus/reopen остаются приёмочными сценариями. |
| Надёжность/защита | Duplicate/session/hash/ACL/DPAPI механизмы есть. F1/F2 и Windows acceptance остаются открытыми. Полная очистка во всех отказах не доказана. |
| Диагностика/health | Q4/Q5: фиксированные сообщения не обвиняют автоматически сервер; один контрольный URL, bounded logs, serial timeout. Raw Xray logs и мгновенное обновление diagnostic-only rows не обеспечены. |
| Декомпозиция §9 | Роли сосредоточены в internal/core и facade/frontend. Имена каталогов в спеке ориентировочные; отсутствие backend/health/ и подобных папок не является нарушением. |

## Приёмка §8: что ещё требуется подтвердить

Существующий [VERIFICATION.md](../../docs/VERIFICATION.md) остаётся подробным сценарием
Windows-приёмки; ниже — связь со всеми сценариями §8, без дублирования инструкции запуска.

| Сценарий | Найденное свидетельство | Открытая проверка |
|---|---|---|
| Валидный/неоднозначный импорт | config/review test definitions | Новый фактический проход соответствующей suite. |
| 3 транспорта × HTTP/SOCKS | Сохранено 6 positive +6 stopped-server Linux PASS | Windows packaged client, контролируемые тестовые серверы, remote egress. |
| Win10/11 normal user, missing WebView2 | asInvoker + prerequisite docs | Обе ОС, normal/missing runtime; launch/ошибка/окно. |
| Fixed loopback ports/no bypass | builder + Linux ephemeral fixture | Реальные порты 10808/10809 на Windows, адреса listeners, отсутствие обхода. |
| Connecting → Connected после probe | service/UI test definitions | Native status/event timing. |
| Duplicate Connect | service/UI test definitions | Реальный child count при гонке. |
| Cancel preflight/start | service/helper/Windows test definitions | Parent/process creation race, оба этапа. |
| Late session events | service test definitions | Фактический regression pass. |
| Crash detection/single Wait | service/helper definitions | Native Xray exit, корректный UI и cleanup. |
| Network loss/recovery | service health definitions | Реальная сеть и отдельно недоступный контрольный URL; без restart. |
| Occupied localhost port | port check + service definition | Каждый из двух портов на Windows, cleanup. |
| Disconnect/forced parent death | Windows Job Object helper definition | Фактический native helper + packaged app, нет child/ports. |
| Window reopen/resync | frontend revision/focus definitions | Реальное окно/WebView2 и свежий GetStatus. |
| Secret-free logs/orphan cleanup | allowlist/store definitions | Windows DPAPI/DACL/другая учётная запись, crash/restart и sentinel secrets. |

При записи следующего прохода фиксировать commit приложения, ОС/build/архитектуру,
normal-user context, Go/Wails/WebView2/Xray версии и package digest, команду/сценарий,
PASS/FAIL/SKIP, cleanup и безопасные ссылки на результаты. Не включать реальные секреты.

## Минимальный комплект документов

Сохранены существующие spec, architecture, audit history, verification и transport evidence.
Добавлены этот отчёт, source research с provenance и [согласованный блок AGENTS.md](AGENTS.proposed.md), установленный в корне 2026-11-01.
Входная точка — [docs/BROWNFIELD.md](../../docs/BROWNFIELD.md).
Новый PRD, пересказ структуры файлов и дублирующие ADR не нужны: требования и решения
уже заданы спекой. F1/F2 — ограниченные следующие изменения; перед реализацией их можно
оформить как tickets. Новых архитектурных решений этот аудит не принимает.
