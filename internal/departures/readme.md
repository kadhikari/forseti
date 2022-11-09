# Departures

## Introduction

We use this service to collect some informations about the next departures for a stop (parameter `stop_id`).
The realtime information is read within an **.txt** like a **Csv** Style (**;** delimiter).

## Api

Run Forseti and call `http://forseti:port/departures`

Input parameters to inform Forseti:

- `--departures-type` The type of departures (Required)(default value = `sytralrt`, possible values = [`sytralrt`, `rennes`, `sirism`])
- `--departures-files-uri` The file Path to read (Required)
- `--departures-files-refresh` The refresh time between 2 readings (Required)
- `--departures-service-uri` The path to the external service (Required for `rennes`)
- `--departures-service-refresh` The refresh time between 2 request of the  external service(Required for `rennes`)
- `--departures-token` The token for the external service (Required for `rennes`)
- `--departures-notifications-stream-name` The name of the AWS Kinesis Data Stream (Required `sirism`)

Exemple:

SERVICE SYTRALRT
``` bash
./forseti --departures-type=sytralrt \
    --departures-files-uri file:///forseti/fixtures/extract_edylic.txt \
    --departures-files-refresh=10s 
```

SERVICE RENNES
``` bash
./forseti --departures-type=rennes \
    --departures-files-uri file:///forseti/fixtures/data_rennes/referential \
    --departures-files-refresh=300s \
    --departures-service-uri https://path/to/external_service \
    --departures-service-refresh=20s \
    --departures-token=12345
```

SERVICE SIRI-SM
``` bash
./forseti --departures-type=sirism \
    --departures-type="sirism" \
    --departures-files-uri="IDontCare" \
    --departures-notifications-stream-name="siri-sm-notif-stream" \
    --timezone-location="Europe/Paris"
```