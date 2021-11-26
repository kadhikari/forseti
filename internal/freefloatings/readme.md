# Free-floatings

## Introduction

We use this service to collect some informations about Vehicles free floatings.
The realtime information is readed within external api service.

## Api

Run Forseti and call `http://forseti:port/free_floatings`

Input parameters to inform Forseti:

   --free-floatings-type citiz --free-floatings-providers 19,127,111,8

- `--free-floatings-uri` The path to api free-floatings to get vehicles (Required)
- `--free-floatings-refresh` The refresh time between 2 readings (Required)
- `--free-floatings-refresh-active` active or deactivates the periodic refresh of data for api
- `--free-floatings-type` The type of flow (Required)
- `--free-floatings-providers` List of providers

Exemple:

```
./forseti --free-floatings-uri "https://path/to/external_service" --free-floatings-refresh=100s 
--free-floatings-refresh-active true --free-floatings-type <type_free_floqtings> --free-floatings-providers 19,127,8
```
