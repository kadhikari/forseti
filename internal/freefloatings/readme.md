# Free-floatings

## Introduction

We use this service to collect some informations about Vehicles free floatings.
The realtime information is readed within external api service.

## Api

Run Forseti and call `http://forseti:port/free_floatings`

Input parameters to inform Forseti:

- `--free-floatings-files-uri` The files Path to read cities coord (Required for Citiz)
- `--free-floatings-uri` The path to api free-floatings to get vehicles (Required)
- `--free-floatings-token` The token external service (Required for Fluctuo)
- `--free-floatings-refresh` The refresh time between 2 readings (Required)
- `--free-floatings-refresh-active` active or deactivates the periodic refresh of data for api
- `--free-floatings-type` The type of flow (Required)(default value = fluctuo, possible values = [fluctuo, citiz])
- `--free-floatings-username` Username to get token (Required for Citiz)
- `--free-floatings-password` Password to get token (Required for Citiz)
- `--free-floatings-area-id` city id for free floating source (Required for Fluctuo)

Exemple: 

SERVICE FLUCTUO
```
./forseti --free-floatings-uri "https://path/to/external_service" --free-floatings-token <token_fluctuo> 
--free-floatings-refresh=100s --free-floatings-refresh-active true --free-floatings-area-id=6
```


SERVICE CITIZ
```
./forseti --free-floatings-files-uri file:///path/to/extract_citiz.json --free-floatings-uri "https://path/to/external_service" 
--free-floatings-refresh=100s --free-floatings-refresh-active true --free-floatings-type <type_free_floqtings> 
```
