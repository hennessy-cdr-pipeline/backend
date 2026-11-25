# CDR generator

Запустите из каталога `backend`:

```bash
go run ./cmd/generator
```

По умолчанию генератор создаёт 10 записей в `cdr_report.csv`, используя справочник `data/subscribers.csv`.

Параметры можно задать так:

```bash
go run ./cmd/generator -count 100 -out result.csv -subscribers data/subscribers.csv
```

На данный момент генерация абсолютно тупая. Пойдёт для начала работы над медиацией