---
id: 5
type: story
title: "Результаты прогонов указывают ревизию приложения и среду"
parent: none
covers: []
after: []
refined: true
hitl: false
risk: medium
---

# Результаты прогонов указывают ревизию приложения и среду

## Description

Следующие integration и Windows-проходы должны однозначно указывать, какой код и
комплект проверялись. Исторический Linux JSON сохраняется как историческое свидетельство,
без добавления задним числом неизвестного app commit.

## Requirements

1. **Новый результат идентифицирует запуск**
   **Given** новый проход integration либо приёмки
   **When** сохраняется результат
   **Then** указаны app commit, признак dirty tree, ОС/build/архитектура, версии
   toolchain/Wails/WebView2 по применимости, Xray/package digests, время, команда,
   сценарии PASS/FAIL/SKIP и результат cleanup; неприменимые поля явно отмечены.
2. **Область результата честна**
   **Given** dirty tree, Linux fixture, fake/helper test либо недоступный шаг
   **When** формируется отчёт
   **Then** область и ограничение записаны; engine commit не выдаётся за app commit,
   SKIP не становится PASS, dirty-прогон не объявлен проверкой точного clean snapshot.
3. **История и секреты сохранены корректно**
   **Given** старый результат и synthetic sentinel credentials
   **When** проверяется новый формат и запись ошибочного/прерванного прохода
   **Then** старый JSON не переписан; секреты и исходные конфигурации не сохраняются;
   доступная информация об отказе и cleanup не теряется.

## Boundaries

Сохранить смысл transport/no-bypass проверок и cleanup. В этой задаче не выполнять
платформенную или сетевую приёмку, не считать подготовленный формат новым PASS.

## References

- parent — none
- finding — _bmad-output/research-runtime-lifecycle-and-interfaces/spec-v0.2-compliance.ru.md, E1; Приёмка §8
- research — _bmad-output/research-runtime-lifecycle-and-interfaces/research-runtime-lifecycle-and-interfaces.md, §6
- verification — docs/VERIFICATION.md, Actual environment and verified checks; Required native Windows acceptance

## Notes

- Предложение риска: medium — неверная provenance создаёт ложное основание приёмки.
- Эта задача владеет общим форматом результатов; задачи 7–10 используют его.

- Decision: 2026-11-01 — пользователь утвердил план и поручил автономное выполнение; критерии и предложенные риски приняты. Доступность сред подтверждается фактически перед стартом.
