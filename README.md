# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Bench 

```text
File: agent.test
Type: alloc_space
Time: Nov 27, 2026 at 2:08am (PST)
Showing nodes accounting for -1566426.84kB, 95.08% of 1647438.22kB total
Dropped 99 nodes (cum <= 8237.19kB)
      flat  flat%   sum%        cum   cum%
-1294704kB 78.59% 78.59% -1574174.25kB 95.55%  compress/flate.NewWriter (inline)
 -271728kB 16.49% 95.08%  -271728kB 16.49%  compress/flate.(*compressor).initDeflate (inline)
    5.16kB 0.00031% 95.08% -787979.51kB 47.83%  github.com/kheguy/collector/internal/agent.(*RetryClient).Do
         0     0% 95.08% -279470.25kB 16.96%  compress/flate.(*compressor).init
         0     0% 95.08% -793830.38kB 48.19%  compress/gzip.(*Writer).Close
         0     0% 95.08% -1574174.25kB 95.55%  compress/gzip.(*Writer).Write
         0     0% 95.08% -1582095.05kB 96.03%  github.com/kheguy/collector/internal/agent.(*Agent).sendMetricsBatch
         0     0% 95.08% -787984.66kB 47.83%  github.com/kheguy/collector/internal/agent.(*RetryClient).Do.func1
         0     0% 95.08% -1582095.05kB 96.03%  github.com/kheguy/collector/internal/agent.BenchmarkAgent_Report
         0     0% 95.08% -787935.43kB 47.83%  github.com/kheguy/collector/internal/agent.BenchmarkAgent_Report.WithGzip.func1
         0     0% 95.08% -787984.66kB 47.83%  github.com/kheguy/collector/internal/agent.benchmarkReportClient.Do
         0     0% 95.08% -793982.83kB 48.19%  github.com/kheguy/collector/internal/agent.compress
         0     0% 95.08% -787984.66kB 47.83%  github.com/kheguy/collector/internal/agent.sendRequest (inline)
         0     0% 95.08% -787087.12kB 47.78%  github.com/kheguy/collector/internal/middlewares.(*gzipResponseWriter).Close
         0     0% 95.08% -787984.66kB 47.83%  github.com/kheguy/collector/internal/retry.Do
         0     0% 95.08% -787984.66kB 47.83%  github.com/kheguy/collector/internal/retry.do
         0     0% 95.08% -787935.43kB 47.83%  net/http.HandlerFunc.ServeHTTP
         0     0% 95.08% -1582095.05kB 96.03%  testing.(*B).run1.func1
         0     0% 95.08% -1582095.05kB 96.03%  testing.(*B).runN
```
