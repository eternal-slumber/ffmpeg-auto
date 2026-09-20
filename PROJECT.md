# Content Factory: устройство проекта и журнал разработки

Этот файл — живая документация проекта. После каждого этапа здесь фиксируются
цель, поток данных, ответственность функций, проверки и следующий шаг.

## Текущее состояние

Готовы этапы 0–3:

1. доказан FFmpeg pipeline с остановкой исходника и chroma key баннера;
2. Go-приложение получает метаданные source и banner через `ffprobe`;
3. исходная длительность делится на равные `ClipPlan`;
4. для каждого клипа рассчитываются точки вставки баннера.

Приложение пока строит план и выводит JSON. Оно ещё не рендерит итоговые клипы.

## Запуск

Требования:

- Go 1.22 или новее;
- `ffprobe` и `ffmpeg` в системном `PATH`;
- macOS или Linux.

Из корня проекта:

```sh
go test ./...
go vet ./...
go run ./cmd/api
```

Полная форма команды:

```sh
go run ./cmd/api /path/to/source.mp4 10 /path/to/banner.mp4
```

Если аргументы не указаны, используются:

```text
source: storage/incoming/source.mp4
parts:  10
banner: storage/incoming/banner.mp4
```

## Полный поток данных под капотом

```text
CLI arguments
    ↓
cmd/api/main.go
    ↓
FFProbe.Probe(source)
    ↓
exec.CommandContext → ffprobe → JSON
    ↓
parseProbeOutput
    ↓
source MediaMetadata
    ↓
FFProbe.Probe(banner)
    ↓
banner MediaMetadata
    ↓
EqualSplit(source.Duration, parts)
    ↓
ClipPlan[]
    ↓
PlanInterruptions(clip.Duration, banner.Duration)
    ↓
каждый ClipPlan получает Interruption[]
    ↓
encoding/json → stdout
```

На текущем этапе выход программы — декларативный план. В нём уже указано,
какую часть source брать и в какие моменты исходного времени вставлять banner.

## Структура исходного кода

```text
cmd/api/main.go                    точка входа и соединение компонентов
internal/model/media.go            метаданные медиафайла
internal/model/clip.go             план одного выходного клипа
internal/model/interruption.go     одна вставка баннера
internal/media/probe.go            запуск и разбор ffprobe
internal/media/probe_test.go       тесты разбора ffprobe JSON
internal/planner/equal_split.go    равномерная разбивка source
internal/planner/equal_split_test.go
internal/planner/interruption.go   правила расположения баннера
internal/planner/interruption_test.go
```

## Этап 0 — Technical spike

Цель этапа — проверить самый рискованный участок до написания backend.

Вручную был проверен единый FFmpeg pipeline:

```text
source
→ вертикальная композиция
→ остановка video и audio
→ freeze frame
→ удаление зелёного фона banner
→ overlay и banner audio
→ продолжение source с прежней позиции
→ MP4
```

Результат находится в `content-factory-spike/final-test.mp4`. Этот эксперимент
доказал техническую осуществимость, но его команда ещё не перенесена в Go.

## Этап 1 — Media Probe

### `model.MediaMetadata`

Единый результат анализа любого медиафайла:

- `Duration` — длительность в секундах;
- `Width`, `Height` — размер первого видеопотока;
- `FPS` — частота кадров;
- `VideoCodec`, `AudioCodec` — имена кодеков;
- `HasVideo`, `HasAudio` — наличие потоков.

Модель ничего не знает о FFmpeg и может использоваться планировщиками,
renderer или будущим HTTP API.

### `media.Prober`

Интерфейс задаёт контракт:

```go
Probe(context.Context, path) (MediaMetadata, error)
```

Контекст позволяет отменить внешний процесс. Путь передаётся отдельным
аргументом, а не частью shell-команды.

### `media.FFProbe.Probe`

Метод выбирает executable: значение `FFProbe.Command` либо `ffprobe` из
`PATH`. Затем `exec.CommandContext` запускает программу с `-of json` и
запрашивает только нужные поля.

`CombinedOutput` сохраняет диагностический текст при ошибке. При успехе байты
передаются в `parseProbeOutput`.

### `parseProbeOutput`

Функция:

1. декодирует JSON во внутреннюю структуру `probeOutput`;
2. переводит строковую длительность в `float64`;
3. находит первый video stream;
4. находит первый audio stream;
5. получает FPS из `avg_frame_rate`, а при его отсутствии — из `r_frame_rate`;
6. возвращает `MediaMetadata`.

### `parseRate`

FFprobe отдаёт FPS как дробь, например `30000/1001`. Функция разбирает
числитель и знаменатель, запрещает деление на ноль и возвращает `29.97...`.

## Этап 2 — Equal Split Planner

### `model.ClipPlan`

`ClipPlan` описывает будущий ролик:

- `Number` — номер с единицы;
- `Start` — начало в source;
- `Duration` — сколько исходного контента использовать;
- `Interruptions` — рассчитанные на этапе 3 вставки.

### `planner.EqualSplit`

Вход:

```text
source duration + number of parts
```

Выход:

```text
[]ClipPlan
```

Функция отклоняет нулевую, отрицательную, бесконечную и `NaN` длительность,
а также неположительное число частей.

Для каждого клипа границы считаются от полной длительности:

```text
start = duration × i / parts
end   = duration × (i + 1) / parts
```

Продолжительности предыдущих клипов не складываются, поэтому ошибка округления
не накапливается. Конец последнего клипа явно приравнивается к концу source.

## Этап 3 — Interruption Planner

### `model.Interruption`

Одна вставка содержит:

- `At` — позицию относительно исходного контента конкретного клипа;
- `Duration` — фактическую длительность banner из `ffprobe`.

Позиции не включают длительность предыдущих баннеров. Например, точки `30` и
`60` означают 30 и 60 секунд source-контента, даже если первая вставка удлинила
готовый файл.

### `planner.PlanInterruptions`

Функция принимает длительности клипа и баннера и проверяет, что обе конечны и
положительны.

Правила MVP:

```text
clip <= 90 секунд → одна вставка посередине
clip > 90 секунд  → вставки на 30, 60, 90, ... пока точка меньше clip duration
```

Точка, совпадающая с концом клипа, не создаётся: после неё не осталось бы
исходного кадра. Поэтому для 120 секунд точки равны `30, 60, 90`, а для 130 —
`30, 60, 90, 120`.

Длительность banner не захардкожена. `main` отдельно вызывает
`FFProbe.Probe(bannerPath)` и передаёт найденную длительность планировщику.

### Связывание этапов в `main`

`main` читает аргументы, одним экземпляром `FFProbe` анализирует source и
banner, вызывает `EqualSplit`, затем проходит по полученным клипам и добавляет
результат `PlanInterruptions`.

Последним шагом стандартный `encoding/json` печатает source metadata, banner
metadata и все планы. На любой ошибке программа пишет причину в stderr и
завершается с ненулевым кодом.

## Что проверяют тесты

Media Probe:

- video/audio codecs и наличие потоков;
- разрешение и длительность;
- дробный FPS;
- fallback на `r_frame_rate`;
- ошибки некорректной длительности и FPS.

Equal Split:

- нужное количество клипов;
- последовательную нумерацию;
- отсутствие промежутков и пересечений;
- точное использование конца source;
- отклонение некорректных аргументов.

Interruption Planner:

- граничные длительности 30, 60, 89, 90, 91, 120, 130 и 180 секунд;
- одинаковую длительность banner во всех вставках;
- отклонение нулевых, отрицательных, бесконечных и `NaN` значений.

## Переносимость macOS → Linux/VPS

В коде нет абсолютных пользовательских путей. Go вызывает `ffprobe` через
`PATH`, работает с относительными путями проекта и использует только стандартную
библиотеку. На VPS достаточно установить Go и FFmpeg и перенести каталог.

## Следующий этап

Stage 4 — Single Clip Renderer:

1. взять один `ClipPlan`;
2. сформировать безопасные аргументы FFmpeg;
3. прочитать участок напрямую из source;
4. создать один MP4 без очереди и без баннера;
5. проверить продолжительность результата через уже готовый `FFProbe`.

Вертикализация, chroma key и фактическая вставка banner остаются следующими
отдельными этапами согласно roadmap.
