# Equipments

## Introduction

We use this service to collect some informations about Stop areas equipments like **Escalators/Elevators**.
The realtime information is readed inside a Ftp server provider inside an **Xml file**.

## Api

Run Forseti and call `http://forseti:port/equipments`

Input parameters to inform Forseti:

- `--equipments-uri` The file Path to read (Required)
- `--equipments-refresh` The refresh time between 2 readings (Required)

Exemple:

```
./forseti --equipments-uri file:///forseti/fixtures/NET_ACCESS.XML --equipments-refresh=1s
```

