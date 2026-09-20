# Content Factory

## 1. Назначение проекта

Content Factory — локальное приложение для автоматической подготовки коротких вертикальных видеороликов из одного длинного исходного видео.

Основная задача первой версии:

1. пользователь загружает исходное видео;
2. приложение анализирует его параметры;
3. пользователь указывает необходимое количество выходных роликов;
4. исходное видео равномерно делится на указанное количество частей;
5. каждый ролик преобразуется в формат TikTok 9:16;
6. в соответствии с правилами в ролик вставляется рекламный баннер;
7. во время баннера исходное видео и аудио полностью приостанавливаются;
8. после баннера воспроизведение продолжается с того же места;
9. пользователь получает готовые MP4-файлы.

Архитектура должна позволять в дальнейшем заменить простое равномерное деление видео на автоматическое определение интересных событий и смысловых фрагментов.

---

# 2. Scope MVP

В первую версию входят:

- загрузка локального видео;
- анализ видео через `ffprobe`;
- отображение метаданных;
- равномерная нарезка на N роликов;
- визуальное представление будущих сегментов на timeline;
- автоматический расчёт точек вставки баннера;
- удаление зелёного фона у баннера;
- остановка исходного видео и аудио во время баннера;
- воспроизведение аудио самого баннера;
- преобразование видео в вертикальный формат 9:16;
- blur-background для горизонтальных видео;
- сохранение исходного FPS;
- рендер конечных MP4;
- очередь обработки;
- повтор отдельной упавшей задачи;
- несколько версий результата из одного исходника;
- автоматическая генерация названий файлов;
- локальное хранение исходников и результатов.

В MVP не входят:

- автоматическая публикация в TikTok;
- управление TikTok-аккаунтами;
- proxy rotation;
- автоматическое получение видео с YouTube;
- AI-анализ смысловых частей;
- распознавание интересных моментов;
- автоматическая генерация субтитров;
- многопользовательская система;
- авторизация;
- облачное хранение;
- горизонтальное масштабирование workers.

---

# 3. Пользовательский сценарий

Пользователь открывает приложение и создаёт новый проект.

```text
New Project
    ↓
Upload source video
    ↓
Video analysis
    ↓
Choose number of clips
    ↓
Timeline preview
    ↓
Choose banner
    ↓
Render
    ↓
Processing
    ↓
Result clips
```

После загрузки приложение показывает:

```text
Название: interview.mp4
Длительность: 32:17
Разрешение: 1920×1080
FPS: 29.97
Codec: H.264
Audio: AAC
```

Пользователь указывает:

```text
Количество роликов

[-]  12  [+]
```

На timeline автоматически появляются 12 равных сегментов:

```text
00:00                                      32:17

|----|----|----|----|----|----|----|----|----|----|----|----|
 #1   #2   #3   #4   #5   #6   #7   #8   #9   #10  #11  #12
```

После этого пользователь запускает генерацию.

---

# 4. Нарезка исходного видео

## 4.1 Equal Split

В первой версии используется стратегия:

```text
EqualSplit
```

Пользователь задаёт количество выходных роликов:

```text
N
```

Длительность одного исходного сегмента:

```text
segmentDuration = sourceDuration / N
```

Например:

```text
source = 600 sec
N = 5

#1   0–120
#2 120–240
#3 240–360
#4 360–480
#5 480–600
```

Все сегменты должны иметь максимально одинаковую продолжительность.

Погрешности при дробных временных значениях должны компенсироваться последним сегментом, чтобы исходное видео было использовано полностью.

---

# 5. Архитектура планировщика нарезки

Механизм нарезки не должен быть связан напрямую с FFmpeg renderer.

Необходимо предусмотреть интерфейс:

```go
type ClipPlanner interface {
    Plan(video VideoMetadata, settings SplitSettings) ([]ClipPlan, error)
}
```

Первая реализация:

```text
EqualSplitPlanner
```

В дальнейшем:

```text
ManualSplitPlanner
EventSplitPlanner
SemanticSplitPlanner
YouTubeInterestPlanner
```

Renderer должен получать уже готовые timestamps и не знать, каким способом они были рассчитаны.

---

# 6. Будущая интеллектуальная нарезка

Архитектура должна допускать использование нескольких сигналов:

```text
YouTube replay graph
scene changes
transcript
audio activity
silence detection
speaker changes
LLM analysis
manual marks
```

Результатом интеллектуального анализа всё равно должен становиться стандартный:

```text
ClipPlan[]
```

Поэтому изменение алгоритма нарезки не должно требовать изменения renderer.

---

# 7. Правила вставки баннера

Баннер представляет собой готовый видеофайл:

```text
banner.mp4
```

Он содержит:

- видеоряд;
- зелёный chroma-key фон;
- собственную аудиодорожку.

Продолжительность баннера не должна быть захардкожена.

Перед обработкой необходимо определить фактическую продолжительность баннера через:

```text
ffprobe
```

Например:

```text
bannerDuration = 5.027 sec
```

Эта длительность используется при формировании итогового видео.

---

# 8. Правила времени вставки

Точки вставки рассчитываются отдельно для каждого конечного клипа.

## Ролики длительностью до 90 секунд включительно

Добавляется один баннер:

```text
position = clipDuration / 2
```

Например:

```text
74 sec

0──────────────37──────────────74
               ↑
             banner
```

## Ролики длительностью больше 90 секунд

Баннер вставляется через каждые 30 секунд исходного контента:

```text
30
60
90
120
150
...
```

Например:

```text
130 sec

0────30────60────90────120──130
     ↑     ↑     ↑      ↑
     B     B     B      B
```

Точки рассчитываются исключительно относительно времени исходного контента.

Вставленные баннеры не должны сдвигать расчёт последующих точек.

---

# 9. Edge cases

Первоначальное правило:

```text
duration <= 90 sec
→ midpoint

duration > 90 sec
→ every 30 sec
```

Таким образом:

```text
90 sec
→ banner at 45

91 sec
→ banners at 30, 60, 90
```

Возможная проблема:

```text
91 sec

banner at 90 sec
→ после него остаётся только 1 sec исходного видео
```

В MVP правило сохраняется буквально.

Позже может быть добавлен параметр:

```text
minimumTailDuration
```

например:

```text
10 sec
```

Тогда баннер не вставляется, если после точки вставки остаётся слишком мало исходного контента.

---

# 10. Поведение видео во время баннера

Во время баннера исходный ролик полностью ставится на паузу.

Необходимо остановить одновременно:

```text
video
audio
```

Исходный timestamp не изменяется.

Схема:

```text
SOURCE

video ─────────────●───────────────────
audio ─────────────●───────────────────
                  pause


OUTPUT

video ─────────────●██████████●───────────────────
                    banner

audio ──────────────│BANNER AUDIO│─────────────────
```

После завершения баннера:

```text
source video resumes
source audio resumes
```

с той же позиции.

---

# 11. Freeze frame

Во время баннера последняя отображённая картинка исходного ролика должна оставаться на экране.

То есть создаётся:

```text
freeze-frame
```

на длительность баннера.

Поверх него проигрывается анимация баннера.

---

# 12. Chroma Key

У баннера зелёный фон.

Он должен удаляться при помощи chroma key.

Pipeline:

```text
banner.mp4
    ↓
chroma key
    ↓
transparent banner video
    ↓
overlay
    ↓
frozen source frame
```

Необходимо предусмотреть параметры:

```text
keyColor
similarity
blend
```

В MVP значения могут быть захардкожены в настройках приложения.

В дальнейшем они могут стать редактируемыми пользователем.

---

# 13. Размер баннера

Во время показа баннер должен занимать не менее:

```text
45%
```

площади итогового видео.

Условие:

```text
bannerArea / frameArea >= 0.45
```

При масштабировании необходимо:

- сохранить aspect ratio баннера;
- не допустить выхода изображения за границы видео;
- обеспечить минимальную требуемую площадь;
- расположить баннер по центру кадра.

Фактический алгоритм масштабирования может быть скорректирован после визуальных тестов.

---

# 14. Выходной видеоформат

Целевой формат:

```text
1080×1920
9:16
MP4
H.264
AAC
```

FPS должен сохраняться исходный.

Например:

```text
23.976 → 23.976
25 → 25
29.97 → 29.97
30 → 30
60 → 60
```

---

# 15. Горизонтальные исходники

Горизонтальное видео не должно просто растягиваться.

Используется схема:

```text
┌─────────────────────────┐
│     blurred source      │
│                         │
│ ┌─────────────────────┐ │
│ │                     │ │
│ │   original source   │ │
│ │                     │ │
│ └─────────────────────┘ │
│                         │
│     blurred source      │
└─────────────────────────┘
```

Один и тот же source используется двумя слоями:

```text
source
 ├─ background
 │    ↓
 │   scale
 │    ↓
 │   crop 9:16
 │    ↓
 │   blur
 │
 └─ foreground
      ↓
     fit
```

Foreground накладывается поверх blurred background.

---

# 16. Вертикальные исходники

Вертикальный исходник приводится к `1080×1920`.

В зависимости от aspect ratio допускается:

```text
fit
```

или небольшой crop.

Контент не должен заметно деформироваться.

---

# 17. Рендер

Необходимо избегать промежуточных MP4.

Неправильно:

```text
source
↓
clip.mp4
↓
vertical.mp4
↓
banner.mp4
↓
final.mp4
```

Правильно:

```text
source.mp4
+
banner.mp4
        ↓
single FFmpeg render pipeline
        ↓
final.mp4
```

Каждый выходной ролик рендерится непосредственно из исходного файла.

---

# 18. Render Plan

Перед запуском FFmpeg backend создаёт RenderPlan.

Пример:

```json
{
  "clip": {
    "index": 3,
    "start": 240.5,
    "duration": 121.8
  },
  "interruptions": [
    {
      "at": 30,
      "media": "banner.mp4"
    },
    {
      "at": 60,
      "media": "banner.mp4"
    },
    {
      "at": 90,
      "media": "banner.mp4"
    },
    {
      "at": 120,
      "media": "banner.mp4"
    }
  ]
}
```

Renderer работает только с RenderPlan.

Он не должен содержать правила вроде:

```text
if duration > 90
```

Это ответственность `InterruptionPlanner`.

---

# 19. Очередь рендера

Каждый конечный ролик является отдельной задачей:

```text
RenderJob
```

Статусы:

```text
queued
processing
completed
failed
cancelled
```

Пример:

```text
#001 completed
#002 completed
#003 processing
#004 queued
#005 failed
```

При ошибке:

```text
Retry
```

должен повторяться только конкретный job.

Остальные успешно обработанные ролики повторно рендерить нельзя.

---

# 20. Worker Pool

Для первой локальной версии:

```text
maxWorkers = 1 или 2
```

Например:

```text
50 clips
   ↓
Render Queue
   ↓
┌───────────┐
│ worker #1 │
│ worker #2 │
└───────────┘
```

Количество workers должно задаваться конфигурацией.

В дальнейшем его можно увеличивать без изменения основной архитектуры.

---

# 21. Версии обработки

Один исходник может иметь несколько вариантов обработки.

Например:

```text
Project
│
├── Source
│
├── Version #1
│     parts = 10
│
├── Version #2
│     parts = 15
│
└── Version #3
      semantic split
```

При изменении настроек исходный файл повторно загружать не требуется.

Новая обработка создаёт:

```text
RenderVersion
```

и не перезаписывает предыдущую.

---

# 22. Имена файлов

По умолчанию имя генерируется автоматически:

```text
interview_001.mp4
interview_002.mp4
interview_003.mp4
```

Пользователь должен иметь возможность изменить отображаемое имя.

Физическое имя/ключ хранения и отображаемое имя желательно разделить:

```text
storageKey
displayName
```

---

# 23. Хранение

В MVP используется локальное файловое хранилище.

Пример:

```text
storage/
├── projects/
│   └── {project-id}/
│       ├── source/
│       ├── banner/
│       └── versions/
│           ├── {version-id}/
│           │   └── clips/
```

Однако domain/business layer не должен непосредственно работать с локальными путями.

Нужна абстракция:

```go
type Storage interface {
    Put(...)
    Open(...)
    Delete(...)
}
```

Первая реализация:

```text
LocalStorage
```

В будущем:

```text
S3Storage
MinIOStorage
```

---

# 24. Основная доменная модель

```text
Project
│
├── SourceVideo
├── Banner
│
└── RenderVersion
      │
      ├── SplitSettings
      ├── RenderSettings
      ├── ClipPlan[]
      └── RenderJob[]
```

Основные сущности:

```go
type Project struct {
    ID UUID
}

type SourceVideo struct {
    ID       UUID
    ProjectID UUID

    Duration float64
    Width    int
    Height   int
    FPS      float64

    VideoCodec string
    AudioCodec string
}

type Banner struct {
    ID       UUID
    Duration float64
    Path     string
}

type RenderVersion struct {
    ID        UUID
    ProjectID UUID

    SplitMode string
    Parts     int
}

type ClipPlan struct {
    Index    int
    Start    float64
    Duration float64

    Interruptions []Interruption
}

type Interruption struct {
    At       float64
    Duration float64
}

type RenderJob struct {
    ID     UUID
    Status RenderStatus
}
```

---

# 25. Компоненты backend

Backend разделяется на независимые компоненты:

```text
HTTP/API
   ↓
Project Service
   ↓
Media Probe
   ↓
Clip Planner
   ↓
Interruption Planner
   ↓
Render Planner
   ↓
Render Queue
   ↓
FFmpeg Worker
   ↓
Storage
```

Ответственность:

### Media Probe

Использует:

```text
ffprobe
```

и получает:

- duration;
- width;
- height;
- FPS;
- codec;
- audio codec;
- audio presence.

### Clip Planner

Определяет:

```text
какие части исходного видео станут роликами
```

### Interruption Planner

Определяет:

```text
где будут баннеры
```

### Render Planner

Объединяет:

```text
ClipPlan
+
Interruptions
+
video settings
+
banner settings
```

в RenderPlan.

### Renderer

Преобразует RenderPlan в команду FFmpeg.

### Worker

Запускает FFmpeg и отслеживает выполнение.

---

# 26. Технологический стек MVP

Backend:

```text
Go
```

Media processing:

```text
FFmpeg
FFprobe
```

Frontend:

может быть выбран позднее.

Для MVP достаточно:

```text
React/Vite
```

либо минимального web UI.

Database:

```text
PostgreSQL
```

или SQLite на самом первом прототипе.

Если планируется перенос на VPS без переделки architecture, предпочтительно сразу PostgreSQL.

Хранилище:

```text
Local filesystem
```

через Storage abstraction.

---

# 27. Локальный запуск

MVP должен работать:

```text
macOS
```

Но приложение не должно зависеть от:

```text
/Users/username/...
```

или других macOS-специфичных путей.

Все зависимости задаются через config/environment.

Например:

```text
DATABASE_URL=
STORAGE_PATH=
FFMPEG_PATH=
FFPROBE_PATH=
MAX_RENDER_WORKERS=2
```

После переноса на Linux/VPS должна требоваться преимущественно смена configuration, а не business logic.

---

# 28. API первой версии

Предварительно:

```text
POST /api/projects
```

создать проект.

```text
POST /api/projects/{id}/source
```

загрузить исходное видео.

```text
GET /api/projects/{id}/source
```

получить metadata.

```text
POST /api/projects/{id}/versions
```

создать вариант нарезки.

Пример:

```json
{
  "split_mode": "equal",
  "parts": 10
}
```

```text
GET /api/versions/{id}/plan
```

получить timeline/ClipPlan.

```text
POST /api/versions/{id}/render
```

запустить обработку.

```text
GET /api/versions/{id}/jobs
```

получить состояния jobs.

```text
POST /api/jobs/{id}/retry
```

повторить ошибочную задачу.

```text
GET /api/clips/{id}/download
```

скачать ролик.

---

# 29. Критерии готовности MVP

MVP считается работоспособным, если можно:

```text
1. загрузить длинный MP4;

2. получить его metadata;

3. указать количество выходных роликов;

4. увидеть равномерную разбивку timeline;

5. создать RenderVersion;

6. автоматически определить точки баннера;

7. обработать banner chroma key;

8. заморозить исходное видео и звук;

9. проиграть banner + banner audio;

10. продолжить исходное видео без потери синхронизации;

11. получить 1080×1920 MP4;

12. корректно обработать горизонтальный source через blurred background;

13. сохранить исходный FPS;

14. обработать несколько clips через очередь;

15. повторить отдельно failed job;

16. создать новую версию проекта без повторной загрузки source;

17. скачать готовые ролики.
```

---

# ROADMAP

## Этап 0 — Technical spike

Цель: доказать, что ключевой media pipeline вообще работает.

Сделать вручную одну FFmpeg-команду:

```text
source.mp4
↓
vertical conversion
↓
freeze at timestamp
↓
green-screen banner
↓
banner audio
↓
resume source
↓
final.mp4
```

Никакого UI, API и базы.

Результат этапа:

```text
один корректный final.mp4
```

Это самый важный первый эксперимент.

---

## Этап 1 — Media Probe

Создать Go-проект.

Реализовать wrapper над:

```text
ffprobe
```

Получать:

```text
duration
width
height
fps
video codec
audio codec
```

Добавить unit tests парсинга.

---

## Этап 2 — Equal Split Planner

Реализовать:

```text
duration + numberOfParts
        ↓
ClipPlan[]
```

Проверить:

- обычное деление;
- дробные duration;
- 1 часть;
- большое N;
- отсутствие потери конца видео.

---

## Этап 3 — Interruption Planner

Реализовать правила:

```text
<= 90 sec
→ midpoint

> 90 sec
→ every 30 sec
```

Длительность баннера получать из metadata файла.

Добавить unit tests для:

```text
30 sec
60 sec
89 sec
90 sec
91 sec
120 sec
130 sec
180 sec
```

---

## Этап 4 — Single Clip Renderer

Научиться рендерить:

```text
один ClipPlan
```

непосредственно из исходного source.

Пока без очереди.

---

## Этап 5 — TikTok video normalization

Добавить:

```text
9:16
1080×1920
```

Горизонтальное:

```text
blur background + foreground
```

Вертикальное:

```text
fit/crop
```

Сохранить source FPS.

---

## Этап 6 — Banner Renderer

Добавить:

```text
freeze source
chroma key
overlay banner
banner audio
resume source
```

Проверить несколько вставок в одном ролике.

Особенно проверить отсутствие audio/video desync.

---

## Этап 7 — Render Plan

Убрать business rules из renderer.

Добавить структуру:

```text
RenderPlan
```

Renderer должен полностью зависеть только от неё.

---

## Этап 8 — Job Queue

Добавить:

```text
queued
processing
completed
failed
```

Сначала можно сделать in-memory queue.

Добавить worker pool:

```text
1–2 workers
```

---

## Этап 9 — Persistence

Добавить PostgreSQL.

Хранить:

```text
projects
source_videos
banners
render_versions
clip_plans
render_jobs
```

---

## Этап 10 — Local Storage abstraction

Добавить:

```text
Storage interface
```

и:

```text
LocalStorage
```

Подготовить architecture под будущий S3/MinIO.

---

## Этап 11 — HTTP API

Добавить endpoints:

```text
projects
upload
versions
plans
render
jobs
download
```

---

## Этап 12 — Первый UI

Минимальная страница:

```text
Upload video

Video info

Number of clips
[-] 10 [+]

Timeline

[Generate]
```

---

## Этап 13 — Timeline

Сделать полноценную визуализацию:

```text
0:00                            20:00
|-----|-----|-----|-----|-----|
  #1    #2    #3    #4    #5
```

При изменении количества:

```text
5 → 8
```

timeline автоматически перестраивается.

---

## Этап 14 — Processing UI

Добавить:

```text
Rendering 4 / 12

#1 ✓
#2 ✓
#3 ✓
#4 processing
#5 queued
...
```

Для failed:

```text
Retry
```

---

## Этап 15 — Results

Показать:

```text
preview
filename
duration
download
```

Автоматически генерировать имена.

Позволить изменить display name.

Позже добавить:

```text
Download all ZIP
```

---

# V1.1

После стабильного MVP:

- ручное перемещение границ timeline;
- настройка размера баннера;
- chroma-key controls;
- minimum tail после баннера;
- выбор качества;
- ZIP;
- удаление проектов;
- progress FFmpeg;
- cancel job;
- drag-and-drop upload.

---

# V2 — Smart clipping

Добавить анализ:

```text
scene detection
silence detection
speech transcription
```

После этого:

```text
SemanticSplitPlanner
```

Пользователь вместо:

```text
10 одинаковых роликов
```

может выбрать:

```text
10 лучших фрагментов
```

---

# V2.1 — YouTube signals

Исследовать возможность получения:

```text
Most replayed
```

или аналогичных сигналов интереса с timeline YouTube.

Эти данные не должны использоваться как единственный источник.

Они становятся дополнительным score:

```text
replayScore
```

для определения потенциально интересных моментов.

---

# V3 — Publishing subsystem

Только после стабильного media pipeline.

Отдельный модуль:

```text
Publisher
```

Архитектура:

```text
Ready Clip
   ↓
Publish Queue
   ↓
Platform Adapter
   ├── TikTok
   ├── YouTube Shorts
   └── Instagram Reels
```

Автопубликация не должна быть встроена в Renderer.

---

# Порядок разработки

Наиболее важный порядок:

```text
FFmpeg spike

→ Probe

→ EqualSplitPlanner

→ InterruptionPlanner

→ single clip render

→ banner/freeze/audio

→ vertical normalization

→ RenderPlan

→ Queue

→ persistence

→ API

→ UI

→ smart clipping

→ publishing
```

Главный принцип проекта:

```text
Сначала сделать надёжный media engine.
Потом строить вокруг него интерфейс и автоматизацию.
```