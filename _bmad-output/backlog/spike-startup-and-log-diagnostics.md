---
id: 6
type: spike
title: "Какие ограничения startup и log diagnostics требуют изменения"
parent: none
covers: []
after: [3]
refined: true
hitl: true
risk: medium
---

# Какие ограничения startup и log diagnostics требуют изменения

## Description

Установить поведение pinned Wails при ошибке ServiceStartup и отделить реальные
недоступные сообщения от условного Promise.all пути. Также зафиксировать решение
владельца продукта о допустимом обновлении диагностических строк без status events.
Пользователь предоставляет доступ к безопасной Windows-среде и принимает решение F4.

## Approach

1. На native Windows проверить ошибку собственного одноразового data directory,
   доступность окна/bridge и безопасное сообщение; в mock UI отдельно проверить
   rejected/pending profile/log calls и приход нового status event.
2. Показать фактическую задержку diagnostic-only строки до следующего refresh,
   не подавляя обычные health/status events. Не предполагать обязательность realtime UI.
3. Сохранить ответ в `_bmad-output/backlog/evidence/startup-and-log-diagnostics.md`.
   Объём — два указанных пути; пользователь подтверждает F4, builder фиксирует F3
   как «ошибка подтверждена», «не достижима» либо «не проверена» со свидетельствами.

## Acceptance Criteria

1. **Условие F3 установлено** — Verify: native/mock результаты различены, ошибка API
   не объявлена потерей всех status events; неизвестное поведение не объявлено PASS.
2. **Решение F4 записано** — Verify: пользователь подтвердил существующую модель либо
   потребность в изменении, без выдуманного численного SLA.
3. **Нужная работа оформлена отдельно** — Verify: доказано отсутствие необходимости
   изменения либо подготовлены конкретные tickets и зависимости затронутой приёмки.
   Spike не закрывает ещё не реализованные исправления.

## References

- parent — none
- specification — docs/xray-mini-spec-v0.2.ru.md, FR-07/08, ADR-008, §7 Диагностика
- finding — _bmad-output/research-runtime-lifecycle-and-interfaces/spec-v0.2-compliance.ru.md, F3/F4
- research — _bmad-output/research-runtime-lifecycle-and-interfaces/research-runtime-lifecycle-and-interfaces.md, §5

## Notes

- Предложение риска: medium — неверный вывод может скрыть причину запуска либо расширить UI без требования.
- Зависимость от 3: общий lifecycle должен быть определён перед проверкой startup/shutdown сообщений.
- Unknown: доступ к безопасной native Windows-среде должен быть подтверждён перед стартом; персональные профили не нужны.
- Open question: решение пользователя по F4 принимается в ходе spike после демонстрации фактического поведения.

- Decision: 2026-11-01 — пользователь утвердил план и поручил автономное выполнение; критерии и предложенные риски приняты. Доступность сред подтверждается фактически перед стартом.
