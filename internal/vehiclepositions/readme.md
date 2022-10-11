# Vehicle positions

## Introduction

We use this service to collect information about position of vehicles.
The realtime information is read within an external api service.

## Api

Run Forseti and call `http://forseti:port/vehicle_positions`

Input parameters to inform Forseti:

- `--positions-files-uri`: The path of the external downloaded files (**optional** ,only used by the connector `rennes`)
- `--positions-files-refresh`: The refresh time between 2 attempts to download files (**optional** ,only used by the connector `rennes`)
- `--positions-service-uri`: The path to external service (**required**)
- `--positions-service-token`: The token external service (**required**)
- `--positions-service-refresh`: The refresh time between 2 attempts to download files through service
- `--positions-service-refresh-active` active or deactivates the periodic refresh of data for api
- `--positions-service-switch`: Time of the day used to reload daily base scheduled data (**optional** ,only used by the connector `rennes`)
- `--positions-clean-vp` time between clean list of VehicleOccupancies (in hours) (**optional** only used by the connector `gtfsrt`)
- `--connector-type`: The type of flow (**required**). Possible values: [`gtfsrt`, `rennes`]
- `--timezone-location`: Name of the location (default value `"Europe/Paris"`)


## Examples:

``` bash
./forseti \
    --positions-service-uri https://service_externe_position/VehicleLocations.pb \
    --positions-service-token token_external_service \
    --positions-refresh=300s \
    --connector-type gtfsrt \
    --positions-clean-vp 2h \
    --positions-service-refresh-active true
```

``` bash
./forseti \
    --positions-files-uri "sftp://username:password@sftp.files-address.com:22/foo" \
    --positions-files-refresh 5m \
    --positions-service-uri "https://service-address.com/bar" \
    --positions-service-token "token_external_service" \
    --positions-service-refresh 3s \
    --positions-service-switch "04:30:00" \
    --positions-service-refresh-active true \
    --connector-type rennes
```
