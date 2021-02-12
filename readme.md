forseti
===========
This project aims to provide jormungandr with realtime data provided by external API.
Realtime data is available as csv files that needs to be downloaded by FTP.

At this time only realtime departures are being handled.

Build
=====
To build this project you need at least [go 1.11](https://golang.org/dl)
Dependencies are handled by go modules as such it is recommended to not checkout this in your *GOPATH*.

To build the project you just need to run the following command:
```
make
```

If you want to run the tests you can run this:
```
make test
```

Finally the linter is available with `make lint` but it requirement to install [golangci-lint v1.11.2](https://github.com/golangci/golangci-lint)
The command `make linter-install` will install golangci-lint by piping the untrusted output of an url into a shell, be careful.


Run
===
Once you have build it it's fairly easy to run it:
```
./forseti --departures-uri file:///PATHTO/extract_edylic.txt --departures-refresh=1s --parkings-uri file:///PATH_TO/parkings.txt --parkings-refresh=2s --equipments-uri file:///home/kadhikari/dev/forseti/fixtures/NET_ACCESS.XML --equipments-refresh=2s --free-floatings-uri <freefloating source url> --free-floatings-token <token> --free-floatings-refresh=60s

```

You can also use the pre-built docker image: navitia/forseti

How does it work
================
The web api is powered by [gin](https://github.com/gin-gonic/gin)
Two routes are provided:
  - `/status` exposes general information about the webservice  
  - `/metrics` exposes metrics in the prometheus text format
  - `/departures` returns the next departures for a stop (parameter `stop_id`)
  - `/parkings/P+R` returns real time parkings data. (with an optional list parameter of `ids[]`)
  - `/equipments` returns informations on Equipments in StopAreas.
  - `/free_floatings?coord=2.37715%3B48.846781` returns informations on freefloatings  within a certain radius as a crow flies from the point.
One goroutine is handling the refresh of the data by downloading them every refresh-interval (default: 30s)
and load them. Once these data have been loaded there is swap of pointer being done so that every new requests
will get the new dataset.

General Architecture
================
Forseti is a webservice that is meant to be integrated as part of [Navitia](https://www.navitia.io) as follow: 

![artchitecture](doc/architecture.png)
