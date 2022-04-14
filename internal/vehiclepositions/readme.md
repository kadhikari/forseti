# Vehicle positions

## Introduction

We use this service to collect information about position of vehicles.
The realtime information is read within an external api service.

## Api

Run Forseti and call `http://forseti:port/vehicle_positions`

Input parameters to inform Forseti:

- `--positions-service-uri` The path to external service (Required)
- `--positions-service-token` The token external service (Required)
- `--positions-refresh` The refresh time between 2 readings (Required)
- `--connector-type` The type of flow (Required). Possible values: [gtfsrt, oditi, fluctuo, citiz]
- `--positions-clean-vp` time between clean list of VehicleOccupancies (in hours)
- `--positions-service-refresh-active` active or deactivates the periodic refresh of data for api


Exemple:

```
./forseti  --positions-service-uri https://service_externe_position/VehicleLocations.pb --positions-service-token token_external_service --positions-refresh=300s --connector-type gtfsrt --positions-clean-vp 2h --positions-service-refresh-active true
```
