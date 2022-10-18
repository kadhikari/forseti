# Vehicle positions

## Introduction

We use this service to collect information about position of vehicles.
The realtime information is read within an external api service.

## Api

Run Forseti and call `http://forseti:port/vehicle_positions`

Input parameters to inform Forseti:

- `--positions-files-uri`: The path of the external downloaded files (**optional**)
- `--positions-files-refresh`: The refresh time between 2 attempts to download files (**optional** ,only used by the connector `rennes`)
- `--positions-service-uri`: The path to external service (**required**)
- `--positions-service-token`: The token external service (**required**)
- `--positions-service-refresh`: The refresh time between 2 attempts to download files through service
- `--positions-service-refresh-active` active or deactivates the periodic refresh of data for api
- `--positions-clean-vp` time between clean list of VehicleOccupancies (in hours) (**optional** only used by the connector `gtfsrt`)
- `--positions-navitia-uri`: The path of the *Navitia*  service  format: [scheme:][//[userinfo@]host][/]path
- `--positions-navitia-token`:  The token of the *Navitia* service
- `--positions-navitia-coverage` : The name of a *Navitia* coverage
- `--connector-type`: The type of flow (**required**). Possible values: [`gtfsrt`, `rennes`]
- `--timezone-location`: Name of the location (default value `"Europe/Paris"`)


## Examples:

### Connector `gtfsrt`

``` bash
./forseti \
    --positions-service-uri https://service_externe_position/VehicleLocations.pb \
    --positions-service-token token_external_service \
    --positions-service-refresh=300s \
    --connector-type gtfsrt \
    --positions-clean-vp 2h \
    --positions-service-refresh-active true
```

### Connector `rennes`

``` bash
./forseti \
    --positions-service-uri "https://service-address.com/foo" \
    --positions-service-token "1234_token_external_service" \
    --positions-service-refresh 3s \
    --positions-service-refresh-active true \
    --positions-navitia-uri "http://navitia-ws-address.com/bar" \
    --positions-navitia-token "5678_token_navitia" \
    --positions-navitia-coverage "coverage-name" \
    --connector-type rennes
```
