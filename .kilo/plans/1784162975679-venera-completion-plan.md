# План завершения проекта Venera

## Шаг 1: Устранение критических уязвимостей многопоточности и паник
1.  **Защита от отправки в закрытый канал**:
    *   Заменить прямую отправку в `signalChan` в `runTsharkCommand` на использование `sync.Cond` или атомарного счетчика.
2.  **Изоляция WaitGroup по процессам**:
    *   Создать структуру `RunningProcess` внутри `processes/manager.go`, которая будет инкапсулировать `context.CancelFunc`, `sync.WaitGroup` и каналы/сигналы конкретного процесса.
3.  **Безопасный конкурентный доступ к мапам фильтров**:
    *   В `data/filter.go` заменить `sync.RWMutex` на `atomic.Value` для хранения мап фильтрации и контроля, что исключит блокировки при перезагрузке списков.

## Шаг 2: Реализация надежного жизненного цикла Tshark в Windows
1.  **Корректный запуск и убийство процессов (Process Trees)**:
    *   В `processes/tshark.go` использовать Windows Job Objects (через `golang.org/x/sys/windows`) для привязки дочерних процессов `tshark.exe` к процессу Venera.
2.  **Парсинг потокового вывода (Tshark Stream Parsing)**:
    *   Изменить аргументы Tshark на `-T ek` (чтобы Tshark выдавал NDJSON - один JSON-объект на строку).
    *   В `runTsharkCommand` корректно обрабатывать построчный JSON.
3.  **Обработка ошибок Tshark**:
    *   Читать `Stderr` процесса Tshark и логировать ошибки.

## Шаг 3: Исправление архитектуры DragonflyDB & PostgreSQL
1.  **Исправление логики Sorted Set**:
    *   В `data/dragonfly.go` и `processes/manager.go` исключить очистку Sorted Set при каждом переносе в Postgres. Sorted Set будет служить витриной. Добавить функцию периодической очистки устаревших записей (например, раз в час).
2.  **Оптимизация пакетной вставки в PostgreSQL**:
    *   В `data/postgres.go` использовать `CopyFrom` для быстрой вставки.
    *   Точно сохранять миллисекунды.

## Шаг 4: Реализация недостающих модулей
1.  **Модуль диагностики (`-diagnose`)**:
    *   Создать `diagnose/diagnose.go` для проверки доступности Tshark, портов БД и прав на запись.
2.  **Завершение Windows Service и Tray**:
    *   Реализовать `services/win_service.go` на базе `github.com/kardianos/service` или `golang.org/x/sys/windows/svc`.
    *   Реализовать `tray/tray.go` на базе `github.com/getlantern/systray`.

## Шаг 5: Веб-интерфейс
1.  **Реализация API и статики**:
    *   В `web/server.go` и `web/handlers.go` реализовать выдачу базового UI и API для получения данных из Sorted Set DragonflyDB и управления процессами.