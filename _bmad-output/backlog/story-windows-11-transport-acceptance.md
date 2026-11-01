---
id: 10
type: story
title: "Три транспорта подтверждены через оба proxy на Windows 11"
parent: none
covers: []
after: [8, 12]
refined: true
hitl: true
risk: high
---

# Три транспорта подтверждены через оба proxy на Windows 11

## Description

Использовать повторяемый сетевой сценарий задачи 12 на системно проверенной
Windows 11 поставке с установленным WebView2. Пользователь обеспечивает разрешённые fixtures/профили и контроль сервера;
builder проводит проверки отдельно, не перенося Linux PASS.

## Requirements

1. **Шесть успешных путей проверены нативно**
   **Given** Windows 11 package, прошедший задачу 8, и fixtures из 12
   **When** три требуемых транспорта проверены через HTTP и SOCKS5
   **Then** каждый из шести запросов имеет ожидаемый ответ/egress;
   оба fixed listeners привязаны только к 127.0.0.1.
   При первоначальном Connect виден Connecting; Connected возникает только после
   успешной проверки через proxy, а не по PID/открытому порту.
2. **Отказы не дают обхода или ложного статуса**
   **Given** отдельно контролируемые target, upstream и контрольный URL
   **When** выполнены шесть no-bypass проверок и сеть потеряна/восстановлена
   **Then** target не получает обходных запросов, сохраняется policy двух failures
   и recovery, нет overlap/restart; ограниченность диагностики отображается честно.
3. **Windows 11 результат полон**
   **Given** завершённые или недоступные сценарии
   **When** сохранён `_bmad-output/backlog/evidence/windows-11-transport.md`
   **Then** есть отдельные PASS/FAIL/SKIP и cleanup по формату 5, без секретов;
   после остановки нет child/listeners/runtime. Итоговая сводка ссылается на Windows 11
   system/transport отчёты и не закрывает MVP при пробелах или нерешённых дефектах.

## Boundaries

Сохранить TLS verification и изоляцию test fixtures, не менять CA/системный proxy/маршруты.
System и transport evidence должны относиться к одному Windows 11 package/commit;
при изменении поставки затронутые проверки повторяются.

## References

- parent — none
- specification — docs/xray-mini-spec-v0.2.ru.md, FR-04/06, ADR-007/011/012, §8
- verification — docs/VERIFICATION.md, Required native Windows acceptance, пункты 3/7
- dependency — _bmad-output/backlog/story-windows-11-system-acceptance.md, Requirements
- dependency — _bmad-output/backlog/story-native-loopback-fixture.md, Requirements
- dependency — _bmad-output/backlog/story-run-evidence-provenance.md, Requirements

## Notes

- Предложение риска: high — loopback не подтверждает внешний egress.
- Unknown: Windows 11 environment, разрешённые test profiles и серверный контроль подтверждаются до старта.
- Зависимость от 12: готовый проверяемый сценарий и локальный test-server setup; сетевые проходы не выполняются одновременно на одном fixture.
- Дополнительная проверка высокого риска: stopped upstream при здоровом target даёт отсутствие target hits для обоих proxy.

- Decision: 2026-11-01 — пользователь утвердил план и поручил автономное выполнение; критерии и предложенные риски приняты. Доступность сред подтверждается фактически перед стартом.

- Decision: 2026-11-01 — пользователь исключил Windows 10 и тестирование без WebView2. Текущая приёмка: Windows 11 x64 с установленным WebView2; остальные проверки сохраняются. Зависимость от снятой задачи 9 заменена готовой fixture 12; внешний egress остаётся обязательным.
