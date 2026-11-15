# film-finder

CLI на Go: ищет фильм по полному или частичному названию и выводит его карточку с Letterboxd (режиссёр, рейтинг, жанры, длительность, описание) и таблицу основных актёров с ролями.

Поиск Letterboxd закрыт Cloudflare, поэтому фильм ищется через [TMDB API](https://developer.themoviedb.org/), а данные берутся со страницы фильма на Letterboxd.

```
$ film-finder -n 3 matrix
 1) The Matrix (1999)
 2) The Matrix Reloaded (2003)
 ...
choose 1-10: 1
┌──────────┬─────────────────────────────────────────┐
│ Title    │ The Matrix (1999)                       │
│ Director │ Lana Wachowski, Lilly Wachowski         │
│ Rating   │ 4.18 / 5 (3206785 ratings)              │
│ Genres   │ Science Fiction, Action                 │
│ Runtime  │ 136 min                                 │
│ URL      │ https://letterboxd.com/film/the-matrix/ │
└──────────┴─────────────────────────────────────────┘

Set in the 22nd century, The Matrix tells the story of a computer hacker who
joins a group of underground insurgents fighting the vast and powerful computers
who now rule the earth.

┌───┬────────────────────┬──────────┐
│ # │ ACTOR              │ ROLE     │
├───┼────────────────────┼──────────┤
│ 1 │ Keanu Reeves       │ Neo      │
│ 2 │ Laurence Fishburne │ Morpheus │
│ 3 │ Carrie-Anne Moss   │ Trinity  │
└───┴────────────────────┴──────────┘
```

## Запуск

Нужен Go 1.27+ и бесплатный ключ TMDB: https://www.themoviedb.org/settings/api

```sh
go build -o film-finder .
cp .env.example .env        # вписать TMDB_API_KEY=...
./film-finder the matrix
./film-finder -n 15 dune    # 15 актёров вместо 10
```

Ключ берётся из переменной окружения `TMDB_API_KEY` или из `.env` (в текущей папке или рядом с бинарником). Подходит ключ v3 и токен v4.

Если найдено несколько фильмов, программа показывает список до 10 вариантов и просит номер.

## Кэш

Ответы кэшируются с TTL, как в DNS: поиск TMDB — 1 час, страницы Letterboxd — 24 часа. Повторный запрос отвечает мгновенно, без сети. Кэш лежит в `~/Library/Caches/film-finder/` (macOS) или `~/.cache/film-finder/` (Linux), сбросить его можно удалением папки.

## Тесты

```sh
go test ./...
```

## Ограничения

- Данные Letterboxd берутся со страницы сайта. При смене вёрстки или усилении защиты Cloudflare инструмент перестанет работать.
- Правила Letterboxd запрещают скрейпинг, поэтому инструмент только для личного использования.
