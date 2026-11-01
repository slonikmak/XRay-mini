---
id: 12
type: story
title: 'Повторяемая Windows fixture для реального core и обоих proxy'
parent: none
covers: []
after: [4, 5, 11]
refined: true
hitl: false
risk: high
---

# Повторяемая Windows fixture для реального core и обоих proxy

## Description

В доступной Windows11 среде создать изолированные loopback VLESS peers для трёх
транспортов и проверить настоящий core через production Service/worker/supervisor.
Это доступная часть подготовки/приёмки; она не заменяет Windows10, GUI и внешний egress.

## Requirements

1. Given pinned runtime и свободные fixed ports, When выполняются три комбинации
   через HTTP и SOCKS5, Then шесть реальных запросов получают ожидаемый ответ.
2. Given здоровый target, When upstream остановлен, Then шесть запросов не обходят
   VLESS; native health деградирует и восстанавливается без второго working client.
3. Given TLS peer, When точный leaf pin заменён неверным, Then запросы отклоняются;
   allowInsecure и изменения OS CA не используются.
4. Given любой исход, When проверка закончилась, Then все fixture children waited,
   Service закрыт, temporary profiles/configs удалены, fixed ports свободны.

## Boundaries

Только synthetic credentials и loopback; сертификат/ключи ephemeral. Health URL может
быть test-only loopback204 с реальным HTTP через proxy, интервал может ускоряться
для проверки переходов — эти отличия явно указываются в результате. Это не production
контрольный URL и не полная приёмка packaged UI или удалённых серверов.

## References

- specification — docs/xray-mini-spec-v0.2.ru.md, FR-04/06/09, ADR-012, §8
- verification — docs/VERIFICATION.md, Required native Windows acceptance
- dependency — _bmad-output/backlog/story-run-evidence-provenance.md, Requirements

## Notes

- Decision: 2026-11-01 — автономное выполнение включает подготовку доступной локальной fixture, не ожидание отсутствующей Windows10.
- Риск high: ложный PASS может скрыть bypass или незавершённый child; дополнительная проверка — неверный TLS pin отвергается при живом сервере/target.
