# Vehicle positions

## Introduction

We use this service to collect information about position of vehicles.
The realtime information is read within an external api service.

## Api

Run Forseti and call `http://forseti:port/vehicle_positions`

Input parameters to inform Forseti:

- `--position-service-uri` The path to external service (Required)
- `--position-service-token` The token external service (Required)
- `--position-refresh` The refresh time between 2 readings (Required)
- `--connector-type` The type of flow (Required)
- `--positions-clean-vp` time between clean list of VehicleOccupancies (in hours)


Exemple:

```
./forseti  --position-service-uri https://service_externe_position/VehicleLocations.pb --position-service-token token_external_service --positions-refresh=300s --connector-type gtfsrt --positions-clean-vp 2
```
