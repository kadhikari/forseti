# Vehicle locations

## Introduction

We use this service to collect information about location of vehicles.
The realtime information is read within an external api service.

## Api

Run Forseti and call `http://forseti:port/vehicle_locations`

Input parameters to inform Forseti:

- `--location-service-uri` The path to external service (Required)
- `--location-service-token` The token external service (Required)
- `--location-refresh` The refresh time between 2 readings (Required)
- `--location-navitia-uri` The path to api navitia to get the coverages (Required)
- `--location-navitia-token` The token navitia (Required)
- `--connector-type` The type of flow (Required)
- `--locations-clean-vj` time between clean list of VehicleJourneys (in hours)
- `--locations-clean-vo` time between clean list of VehicleOccupancies (in hours)



Exemple:

```
./forseti  --location-service-uri https://service_externe_location/VehicleLocations.pb --location-service-token token_external_service --locations-refresh=300s --locations-navitia-uri https://api.navitia.io/v1/coverage/fr-idf --locations-navitia-token token_Navitia  --connector-type gtfsrt --locations-clean-vj 24 --locations-clean-vo 2
```
