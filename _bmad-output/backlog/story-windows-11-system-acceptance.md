---
id: 8
type: story
title: "Подтверждена системная приёмка Windows 11 x64"
parent: none
covers: []
after: [4, 5, 6]
refined: true
hitl: true
risk: high
---

# Подтверждена системная приёмка Windows 11 x64

## Description

Выполнить отдельную системную приёмку на Windows 11 x64. Результат Windows 10 не
переносится на эту ОС. Пользователь предоставляет изолированные WebView2-варианты
и вторую учётную запись; builder проверяет пакет обычным пользователем.

## Requirements

1. **Среда и поставка определены**
   **Given** полный пакет из 4 и схема результатов из 5
   **When** начинается проход Windows 11
   **Then** записаны OS/build, app revision/dirty, versions/hashes и normal-user context;
   отсутствующий/изменённый runtime не исполняется.
2. **Проверены native гарантии**
   **Given** безопасная одноразовая среда
   **When** выполнены пункты 1, 2, 4, 5, 6, 8 native checklist
   **Then** отдельно подтверждены Job Object/crash/cancel/Wait/lock/ports,
   DPAPI/DACL/другая учётная запись/recovery/второй экземпляр,
   WebView2 available/missing, окно/focus/reopen, bounded secret-free diagnostics.
   Packaged UI импортирует валидный synthetic профиль и объясняет отказ для
   невалидного/неоднозначного; повторный Connect не создаёт второй рабочий Xray.
   Принудительный выход рабочего Xray обнаруживается, UI отражает завершение cleanup.
   Допустим структурно валидный профиль с недоступным локальным endpoint: он проверяет
   процесс/UI без успешного транспорта, который относится к 10.
3. **Есть самостоятельное свидетельство Windows 11**
   **Given** завершённые и недоступные сценарии
   **When** сохранён `_bmad-output/backlog/evidence/windows-11-system.md`
   **Then** записаны PASS/FAIL/SKIP, наблюдения и cleanup; нельзя закрыть приёмку
   при SKIP или неустранённом дефекте. Transport/egress — задача 10.

## Boundaries

Не удалять WebView2 и не менять CA/системные proxy-настройки личной машины.
Использовать изолированные варианты среды. Исправления найденных дефектов оформлять
отдельно; Linux/cross-build/mock не заменяют этот проход.

## References

- parent — none
- specification — docs/xray-mini-spec-v0.2.ru.md, §7–8, ADR-006/009
- verification — docs/VERIFICATION.md, Required native Windows acceptance, пункты 1/2/4/5/6/8
- dependency — _bmad-output/backlog/story-package-completeness.md, Requirements
- dependency — _bmad-output/backlog/story-run-evidence-provenance.md, Requirements
- dependency — _bmad-output/backlog/spike-startup-and-log-diagnostics.md, Acceptance Criteria

## Notes

- Предложение риска: high — ошибки платформенной изоляции и lifecycle скрываются без native прохода.
- Unknown: доступ к Windows 11 x64, изолированным WebView2-вариантам и второй учётной записи подтверждается до старта.
- Все варианты среды, toolchain и synthetic fixtures готовятся до execution-сессии; отсутствующая подготовка получает отдельную setup-задачу.
- Результат 3 должен быть рассмотрен; необходимое исправление из spike становится отдельной зависимостью.
- Дополнительная проверка высокого риска: parent-death stress при создании child и nested jobs с записанными повторениями и cleanup.

- Decision: 2026-11-01 — пользователь утвердил план и поручил автономное выполнение; критерии и предложенные риски приняты. Доступность сред подтверждается фактически перед стартом.
