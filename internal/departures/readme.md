# Departuress

## Introduction

We use this service to collect some informations about the next departures for a stop (parameter `stop_id`).
The realtime information is readed within an **.txt** like a **Csv** Style (**;** delimiter).

## Api

Run Forseti and call `http://forseti:port/departures`

Input parameters to inform Forseti:

- `--departuress-uri` The file Path to read (Required)
- `--departures-refresh` The refresh time between 2 readings (Required)

Exemple:

```
./forseti --departures-uri file:///forseti/fixtures/extract_edylic.txt --departures-refresh=1s
```

