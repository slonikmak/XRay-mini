# Ручной импорт подписки — проверка 2026-11-01

**PASS:** предоставленная подписка загружена через packaged Windows UI, показаны пять XHTTP/TLS кандидатов, выбран и импортирован один профиль. Проверен первый кандидат; доступность остальных четырёх серверов не проверялась. URL, UUID, peer addresses и значения выходных IP в отчёты не записаны.

Среда: Windows 11 Home amd64 10.0.26300.9711, обычный пользователь; Go 1.26.10, Wails 3.0.0. WebView2 отрисовал реальное окно; версия этим прогоном не измерялась (unavailable в tool metadata не означает отсутствие runtime). Полный исходный HEAD `c0e412d67569c018b9c309cccdb559623f3a4d39`, dirty реализация. Точные source/artifact hashes — [manifest](subscription-acceptance-manifest-2026-11-01.json). GUI SHA256 `8dd510de3d9c6f451fdde08caf7623ac89b2ce45dc5b39af1563ea22fedc1ad7`; pinned Xray 26.10.29 SHA256 `07a332b714a3c0eee047a22a9c0ea749b64df33e4e1ff6d28a8c7f9ef943ce10`.

В отдельном процессе с временным APPDATA открыт Import profile → Subscription or VLESS link. Ссылка введена в защищённое поле. Load profiles показал пять кандидатов без автоматического импорта или Connect. Выбран первый, сохранён один .profile под нейтральным именем. Реальный runtime UUID отсутствовал в ciphertext файла ([storage inspection](subscription-live-storage-2026-11-01.json)); это не cross-user DPAPI проверка.

Connect показал Connecting/Starting, затем Running. OS inspection: один собственный Xray, два listeners только 127.0.0.1:10808/10809. Первый штатный probe не прошёл и UI не заявил Connected. Следующий probe примерно через 30 секунд прошёл; UI показал Connected/Reachable. Причина первой неудачи не устанавливалась.

Через каждый proxy получен HTTPS 204 от контрольного ресурса и корректный IP-ответ другого публичного ресурса. Оба egress IP отличались от прямого baseline ([proxy record](subscription-proxy-smoke-2026-11-01.json)); сами IP не сохранены. TLS verification не отключалась, системный proxy/routes/CA не менялись. Это один реальный XHTTP/TLS endpoint, не внешняя приёмка других транспортов.

После Disconnect UI показал Disconnected/Not running/Not verified, OS inspection: 0 собственных Xray children, 0 proxy listeners, 0 runtime directories. Повторно открытый импорт содержал пустой source и не содержал кандидатов. Видимые восемь diagnostic entries содержали только фиксированные категории/безопасные сообщения, без URL, UUID или endpoint. Alt+F4 завершил GUI с exit 0; controller подтвердил один профиль и удалил принадлежащее тесту временное хранилище ([GUI run 2](subscription-gui-2026-11-01-r2.json)). Личное хранилище не менялось.

Предыдущая попытка была остановлена пользователем физическим Esc до импорта; оставленный controller затем достиг timeout и очистился. Её [FAIL/timeout record](subscription-gui-2026-11-01.json) сохранён отдельно и не считается дефектом подключения. Второй проход самостоятельный.

Автоматические проверки после всех review fixes: production Go suite/vet PASS ([лог](subscription-go-checks-2026-11-01.log)); frontend typecheck и 30 tests PASS ([лог](subscription-frontend-checks-2026-11-01.log)); production Wails build PASS ([лог](subscription-wails-build-2026-11-01.log)); offline package verifier PASS. Все строки матрицы имеют исполнившиеся тесты, включая UTF-8, multibyte names, TLS/limits, canceled/replaced/expired previews и delayed-save/clear ordering. Пять findings независимого quick-review и координатора исправлены; [build plan](../../plan-manual-subscription-import.md).

Функция готова в BMAD built. Общая приёмка MVP остаётся открытой: cross-user DPAPI, оставшийся native GUI/stress checklist и внешние проверки RAW/REALITY и XHTTP/REALITY. Windows 10 и отсутствие WebView2 исключены пользователем.
