# Автономное выполнение плана — 2026-11-01

В доступной среде завершены реализация, независимое ревью и проверки задач 1–6, 11 и 12. Четыре задачи полной приёмки 7–10 заблокированы; частичные результаты Windows 11 сохранены. **MVP целиком ещё не принят.**

## Что изменилось

- Ошибка удаления runtime сохраняет ownership сессии. Connect запрашивает одну попытку cleanup; после её успеха следующий Connect запускает Xray. Нет фонового цикла повторов и второго child.
- Close ограничен пятью секундами ожидания session/storage и 250 мс публикации. Ошибка не выдаётся за успешную остановку; один уже исполняемый произвольный callback может жить до своего возврата. Остановка child использует принадлежащий сессии Job Object и единственный Wait.
- Безопасная ошибка startup видна в native окне; snapshot применяется независимо от зависших/неудачных profile/log запросов и позднего появления host bridge.
- `python scripts/package_runtime.py --verify bin` проверяет GUI PE, pinned runtime и notices без сети. GUI hash информационный, подпись/происхождение GUI этим не доказываются.
- Новые результаты получают commit/dirty, OS/build, версии, artifact hashes, FAIL/SKIP и cleanup. Старый Linux JSON сохранён побайтно. Добавлен opt-in native loopback fixture.

## Проверки

Среда: Windows 11 Home amd64 10.0.26300.9711/26H2, обычный пользователь; Go 1.26.10, Wails 3.0.0, WebView2 156.0.4314.58. Исполнялся dirty checkout на полном HEAD `ee4832faa679f632b0bda850fe6c6f288dedcd24`; это не результат чистого исходного commit. Хеши исходников и артефактов — [manifest](execution-source-manifest.json). После исполнения менялись документация и планы, а не проверенный production-код.

| Проверка | Результат и границы |
|---|---|
| `go test -tags production -count=1 -timeout=180s ./...` | PASS; facade 3.833s, core 7.150s; opt-in transport case отдельно |
| `go vet -tags production ./...` | PASS |
| frontend typecheck / Vitest | PASS, 23 теста |
| Python packaging/provenance | PASS, 25 тестов, [лог](python-checks-2026-11-01.log) |
| `wails3 build` | PASS, [финальный лог](wails-build-startup-fix-2026-11-01.log) |
| Offline package verifier | PASS; весь bin, включая notices; проверочный test exe не является частью поставки |
| Real-core Windows loopback | PASS, 18.14s, [матрица и ограничения](native-windows-loopback-2026-11-01.md) |
| Native GUI / crash / profile recovery | Частичный PASS, [системные наблюдения](windows-11-system.md) |
| Native race instrumentation | SKIP: нет Windows C toolchain/gcc; обычный PASS не заменяет `-race` |

Независимое ревью нашло недоступную из UI cleanup-попытку, слабую GUI-проверку, два пробела provenance, платформенную изоляцию facade-теста и два пробела fixture cleanup/negative control. Все исправлены с повторными проверками. Живое окно дополнительно выявило неверный browser-preview banner при позднем bridge; исправлено и перепроверено. Неудачная первая попытка crash-controller сохранена отдельно, не переименована в PASS.

## Что остаётся

| Задача | Статус | Что требуется |
|---|---|---|
| 7: Windows 10 system | BLOCKED / SKIP | Windows 10 x64, второй аккаунт, WebView2-варианты |
| 8: Windows 11 system | PARTIAL / BLOCKED | Второй аккаунт, изолированная missing-WebView2 среда и оставшийся полный native GUI/stress checklist |
| 9: Windows 10 transport | BLOCKED / SKIP | Проверенная Windows 10 и разрешённые внешние peers/egress |
| 10: Windows 11 transport | PARTIAL / BLOCKED | Полная system acceptance, внешние peers/egress и Windows 10 handoff |

Реальные локальные проверки не доказывают внешний egress, обычную системную TLS trust chain, Windows 10 или cross-user DPAPI. Решение по diagnostic-only обновлениям — сохранить существующую модель без нового realtime SLA — принято в пределах делегированной автономии ([spike 6](startup-and-log-diagnostics.md)).

Планы завершённых реализаций имеют BMAD `built` (board: review); это отделяет готовую проверенную работу от окончательной приёмки владельцем. Для задач 7–10 записаны точные blockers, автоматический выбор новой реализации не обходит их. Локальный commit предусмотрен bmad-build; push не выполнялся. Прежние generated bindings сохранены [побайтно](generated-bindings-preserved.json), сторонние dirty-файлы не включены.
