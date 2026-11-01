---
id: 7
type: story
title: "Подтверждена системная приёмка Windows 10 x64"
parent: none
covers: []
after: [4, 5, 6, 11]
refined: true
hitl: true
risk: high
---

# Подтверждена системная приёмка Windows 10 x64

## Description

Проверить реальное приложение и native helper tests на Windows 10 x64 обычным
пользователем. Пользователь предоставляет одноразовую среду, вторую учётную запись
и вариант без WebView2; builder выполняет доступные проверки и сохраняет результат.

## Requirements

1. **Проверена указанная поставка**
   **Given** полный пакет из задачи 4 и формат свидетельств из 5
   **When** запускается приёмка Windows 10
   **Then** записаны точные OS/build, app revision/dirty, package/runtime hashes,
   normal-user context и версии; missing/modified runtime отвергаются до исполнения.
2. **Пройден системный сценарий**
   **Given** изолированная среда с контролируемыми файлами и helper processes
   **When** выполнены пункты 1, 2, 4, 5, 6, 8 native checklist из References
   **Then** подтверждены Job Object/parent death, executable lock, cancel/late-start,
   once-only Wait, оба занятых порта, DPAPI/DACL и другая учётная запись, orphan recovery,
   второй экземпляр, WebView2 available/missing, окно/focus/reopen и secret-free bounded logs.
   В packaged UI валидный синтетический профиль импортируется, невалидный/неоднозначный
   отклоняется с объяснением; повторный Connect не создаёт второй рабочий Xray.
   Принудительный выход рабочего Xray обнаруживается, UI после cleanup показывает
   корректное состояние. Для этих process-проверок достаточно структурно валидного
   профиля с недоступным локальным endpoint; успешный транспорт проверяется в 9.
3. **Результат не скрывает пробелы**
   **Given** доступные и недоступные проверки
   **When** сохранён `_bmad-output/backlog/evidence/windows-10-system.md`
   **Then** каждый сценарий имеет PASS/FAIL/SKIP со свидетельством и cleanup;
   SKIP или неустранённый дефект не позволяет назвать системную приёмку завершённой.
   Реальные transport/egress проверки отнесены к задаче 9.

## Boundaries

Не менять рабочие настройки Windows, доверенные CA или установленный WebView2 на
личной машине; варианты отсутствующего runtime и второй учётной записи проверять
в предоставленной одноразовой среде. Не исправлять приложение в этой acceptance-задаче:
подтверждённый дефект получает отдельный ticket и повторную проверку.

## References

- parent — none
- specification — docs/xray-mini-spec-v0.2.ru.md, §7–8, ADR-006/009
- verification — docs/VERIFICATION.md, Required native Windows acceptance, пункты 1/2/4/5/6/8
- dependency — _bmad-output/backlog/story-package-completeness.md, Requirements
- dependency — _bmad-output/backlog/story-run-evidence-provenance.md, Requirements
- dependency — _bmad-output/backlog/spike-startup-and-log-diagnostics.md, Acceptance Criteria

## Notes

- Предложение риска: high — ложная приёмка скрывает утечки процессов/секретов.
- Unknown: доступ к Windows 10 x64, изолированным WebView2-вариантам и второй учётной записи должен быть подтверждён перед стартом.
- Все варианты среды, toolchain и синтетические fixtures должны быть подготовлены до execution-сессии; если подготовки нет, сначала оформить отдельную setup-задачу.
- Результат 3 должен быть рассмотрен: найденное там необходимое исправление добавляется в зависимости до старта этой задачи.
- Дополнительная проверка высокого риска: stress forced parent death в момент создания child и nested-job execution; сохранить число повторов, наблюдения и cleanup.

- Decision: 2026-11-01 — пользователь утвердил план и поручил автономное выполнение; критерии и предложенные риски приняты. Доступность сред подтверждается фактически перед стартом.

- Dropped: 2026-11-01 — Windows 10 исключена пользователем из scope MVP; задача снята, не считается пройденной.
