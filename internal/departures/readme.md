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
- `--departures-notifications-stream-name` The name of the AWS Kinesis Data Stream (Required for `sirism`)
- `--departures-notifications-reload-period` The duration since 
- `--departures-stream-read-only-role-arn` The ARN of the role to assume on AWS to read Kinesis Data Stream (Optional but only for 
`sirism`)
- `--departures-service-switch` Required by several connectors as follows:
    - the connector `rennes`: the time of day when the operating day starts
    - the connector `sirism`: the time of day when the departures are deleted

Exemple:

SERVICE SYTRALRT
``` bash
./forseti --departures-type=sytralrt \
    --departures-files-uri file:///forseti/fixtures/extract_edylic.txt \
    --departures-files-refresh=10s 
```

SERVICE RENNES
``` bash
./forseti --departures-type="rennes" \
    --departures-files-uri="file:///forseti/fixtures/data_rennes/referential" \
    --departures-files-refresh="300s" \
    --departures-service-uri="https://path/to/external_service" \
    --departures-service-refresh="20s" \
    --departures-token="12345" \
    --departures-service-switch="04:30:00"
```

SERVICE SIRI-SM
``` bash
./forseti --departures-type=sirism \
    --departures-type="sirism" \
    --departures-files-uri="IDontCare" \
    --departures-notifications-stream-name="siri-sm-notif-stream" \
    --departures-notifications-reload-period="75m" \
    --departures-service-switch="03:00:00" \
    --timezone-location="Europe/Paris"
```