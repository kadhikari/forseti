# Vehicle occupancies

## Introduction

We use this service to collect information about occupancy of a vehicles at a stop.
The realtime information is readed within Csv files and an external api service.

## Api

Run Forseti and call `http://forseti:port/vehicle_occupancies`

Input parameters to inform Forseti:

- `--occupancy-files-uri` The files Path to read courses and stop (Required)
- `--occupancy-refresh` The refresh time between 2 readings (Required)
- `--occupancy-navitia-uri` The path to api navitia to get the coverages (Required)
- `--occupancy-navitia-token` The token navitia (Required)
- `--occupancy-service-uri` The path to external service (Required)
- `--occupancy-service-token` The token external service (Required)
- `--occupancy-service-refresh-active` active or deactivates the periodic refresh of data for api

Exemple:

```
./forseti --occupancy-files-uri file:///path/to/extract_courses_and_stop.csv --occupancy-refresh=300s --occupancy-navitia-uri https://path/to/api_navitia --occupancy-navitia-token token_navitia --occupancy-service-uri https://path/to/external_service --occupancy-service-token token_external_service --occupancy-service-refresh-active true
```
