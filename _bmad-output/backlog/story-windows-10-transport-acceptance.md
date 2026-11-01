---
id: 9
type: story
title: "Три транспорта подтверждены через оба proxy на Windows 10"
parent: none
covers: []
after: [7]
refined: true
hitl: true
risk: high
---

# Три транспорта подтверждены через оба proxy на Windows 10

## Description

Пользователь предоставляет разрешённые контролируемые test servers и синтетические
профили трёх транспортов, включая способ безопасно останавливать тестовый upstream.
Builder проверяет полный Windows 10 package, а не только engine/config fixture.
Эта задача первой фиксирует повторяемый сетевой сценарий и передаёт его задаче 10.

## Requirements

1. **Три транспорта и два пути проверены**
   **Given** прошедшая системную проверку Windows 10 поставка и разрешённые fixtures
   **When** RAW+REALITY, XHTTP+REALITY и XHTTP+TLS проверены через HTTP и SOCKS5
   **Then** все шесть запросов имеют ожидаемый ответ и подтверждённый тестовый egress;
   listeners используют только 127.0.0.1:10808/10809, direct bypass не обнаружен.
   При первоначальном Connect виден Connecting; Connected появляется только после
   успешного запроса через proxy, а не после открытия порта или запуска child.
2. **Отказы и восстановление честны**
   **Given** живой локальный target и возможность независимо остановить upstream
   **When** выполнены шесть stopped-server no-bypass проверок и потеря/возврат сети
   **Then** обхода нет; первая ошибка probe даёт unhealthy, вторая Degraded,
   успех возвращает Connected, проверки не перекрываются и Xray не перезапускается;
   отказ одного контрольного URL не объявлен доказанным отказом VLESS-сервера.
3. **Cleanup и свидетельства сохранены**
   **Given** каждый positive/negative сценарий
   **When** выполнены Disconnect/завершение и записан `_bmad-output/backlog/evidence/windows-10-transports.md`
   **Then** нет оставшихся child/listeners/runtime, результаты используют формат 5,
   секреты не записаны, FAIL/SKIP не превращены в PASS. Сценарий доступен задаче 10.

## Boundaries

Не менять системные proxy/маршруты/CA; TLS не отключать. Не использовать личные
профили либо посторонние серверы. Новые fixtures не должны затрагивать другие сессии.
Transport acceptance требует системного прохода того же package/commit; при изменении
поставки затронутые системные проверки повторяются.

## References

- parent — none
- specification — docs/xray-mini-spec-v0.2.ru.md, FR-04/06, ADR-007/011/012, §8
- verification — docs/VERIFICATION.md, Required native Windows acceptance, пункты 3/7
- research — _bmad-output/research-runtime-lifecycle-and-interfaces/research-runtime-lifecycle-and-interfaces.md, §4/6
- dependency — _bmad-output/backlog/story-windows-10-system-acceptance.md, Requirements
- dependency — _bmad-output/backlog/story-run-evidence-provenance.md, Requirements

## Notes

- Предложение риска: high — ложный PASS скрывает обход proxy или неподдерживаемый транспорт.
- Unknown: разрешённые серверы, synthetic profiles и контроль upstream/egress должны быть доступны до старта; сейчас их наличие не предполагается.
- Дополнительная проверка высокого риска: после остановки upstream target остаётся здоровым, но не получает запросов обходным путём.

- Decision: 2026-11-01 — пользователь утвердил план и поручил автономное выполнение; критерии и предложенные риски приняты. Доступность сред подтверждается фактически перед стартом.

- Dropped: 2026-11-01 — Windows 10 исключена пользователем из scope MVP; задача снята, не считается пройденной.
