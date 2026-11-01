---
id: 11
type: bug
title: 'Ошибка backend startup должна быть видна в окне'
parent: none
covers: []
after: [6]
refined: true
hitl: false
risk: medium
severity: P2
---

# Ошибка backend startup должна быть видна в окне

## Description

Pinned Wails прекращает запуск окна при ошибке ServiceStartup. Безопасная причина
например конфликта владельца каталога остаётся только в stderr GUI-процесса.
Приложение должно показывать причину, сохраняя запрет операций недоступного backend.

## Reproduction

Изолировать APPDATA только у тестового процесса, удерживать его .instance.lock,
запустить готовый пакет. Нативный baseline-проход startup-before.json: процесс вышел
с кодом1 за0.282с, fixed ownership error только в stderr; источник GUI commit не встроен.
Source SDK3.0.0 независимо подтверждает abort до запуска pending windows.

## Cause Hypothesis

Фатальная ошибка Wails startup завершает запуск до UI; даже при нефатальном startup
общий Promise.all может отбросить полезный snapshot при отказе ListProfiles.

## Acceptance Criteria

1. **Причина доступна**: Given backend не создан из-за безопасной startup-категории,
   When приложение запускается, Then окно остаётся доступным, GetStatus показывает
   безопасную причину, не раскрывая пути/профили, а guarded операции отказывают.
2. **Snapshot независим**: Given profile/log запрос отказал либо задержался,
   When свежий GetStatus завершился, Then его статус применяется без ожидания
   несвязанного I/O и старые revisions не перезаписывают новые events.
3. **Проверка**: Given temp data owner и mock UI failure,
   When проверки выполнены, Then façade и frontend regression проходят; native
   window/ownership failure проверяется на доступной Windows11 отдельно.
4. **Либо без изменения**: все требуемые результаты доказаны без изменения продукта;
   опровержение одной гипотезы причины не закрывает задачу.

## Boundaries

Backend не считается работоспособным при startup error. Не подменять ошибку fake
профилями или Connected. Сохранить secret-free errors, status revisions и UI cancellation.

## References

- finding — _bmad-output/research-runtime-lifecycle-and-interfaces/spec-v0.2-compliance.ru.md, F3
- evidence — _bmad-output/backlog/evidence/startup-before.json
- specification — docs/xray-mini-spec-v0.2.ru.md, FR-07/08, ADR-008

## Notes

- Decision: 2026-11-01 — необходимый follow-up из spike6 включён в авторизованное автономное выполнение.
- Risk medium: отсутствие видимой причины делает исправление запуска недоступным пользователю; P2.
