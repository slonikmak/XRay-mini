# Runtime lifecycle and interfaces — approved research brief

Approved: 2026-10-31. Repository: C:/Users/kp/Documents/projects/XRay-mini.
Commit: 910c353955f02b071fb545b0512b7585cca85189 (`910c353`), branch `main`.
Snapshot only: working-tree changes are excluded from research.

1. **How** are profiles imported, stored and transformed into runtime configuration?
2. **How** are connection commands, session events and cleanup handled?
3. **How** are child processes owned, Windows lifecycle handled and runtime binaries delivered?
4. **How** do health checks and diagnostic logs work?
5. **How** does UI/API synchronization, profile management and diagnostics display work?
6. **How** is this behavior tested and what result artifacts exist?

Limits per question: 15 source files; history 10 entries if needed. Validation: normal.
Up to three concurrent agents. Locators search first; analyzers read the immutable snapshot;
fresh verifiers check load-bearing claims. Researchers receive factual questions only.

Excluded from researcher searches: `_bmad/`, `_bmad-output/`, `.agents/`, all spec,
audit and planning documents under `docs/`, root `xray-mini-spec-v0.2.ru.md`,
generated dependency/build directories and any tickets or stories.

No prior research run or standalone ADR was found. No historical why-question is asked.
The parent will compare the resulting factual research against the supplied specification
in a separate document. Running application/system/network tests is outside this source research.
