# Windows 11 transport acceptance — PARTIAL / BLOCKED

2026-11-01. [Ticket 12 native loopback matrix](native-windows-loopback-2026-11-01.md) passed with real pinned Windows Xray, real Service and Job/DPAPI paths. Three transports use both proxies; 8 positive requests, 6 stopped-peer and 2 wrong-pin rejections, 3 duplicate-Connect rejections and recovery without restart. This setup was extracted so useful Windows 11 work did not wait for unavailable Windows 10.

SKIP: authorized external peers, remote egress proof, ordinary OS TLS trust-chain acceptance, production external probe acceptance and complete system campaign. Local TLS uses an exact ephemeral leaf pin with localhost serverName, no allowInsecure or CA changes. Local probe interval/endpoint are explicit test hooks. Full task 10 remains blocked, as do its system/Windows 10 dependencies; no full MVP acceptance is claimed.
