package vehiclepositions

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/hove-io/forseti/internal/utils"

	"github.com/hove-io/forseti/google_transit"
)

// new type definitions
type ExternalVehiculeId string

const vehiclePositionsCsvNumOfFields int = 9

type VehiclePosition struct {
	// External (navitia) vehicle ID
	ExternalVehiculeId ExternalVehiculeId
	State              State
	Heading            int
	Latitude           float64
	Longitude          float64
	OccupancyStatus    google_transit.VehiclePosition_OccupancyStatus
}

type State int

const (
	HS State = iota
	HL
	INC
	LIGN
	TDEP
	HLP
	TARR
	DEVP
	SPEC
	ATE
	HLPS
	HLPR
	DEV
)

var (
	statesMap = map[string]State{
		"HS":   HS,
		"HL":   HL,
		"INC":  INC,
		"LIGN": LIGN,
		"TDEP": TDEP,
		"HLP":  HLP,
		"TARR": TARR,
		"DEVP": DEVP,
		"SPEC": SPEC,
		"ATE":  ATE,
		"HLPS": HLPS,
		"HLPR": HLPR,
		"DEV":  DEV,
	}
)

func parseStateFromString(stateStr string) (State, bool) {
	state, ok := statesMap[stateStr]
	return state, ok
}

func newVehiclePosition(record []string) (*VehiclePosition, error) {
	if len(record) < vehiclePositionsCsvNumOfFields {
		return nil, fmt.Errorf("missing field in VehiclePosition record")
	}

	// Column #1
	extVehicleId := ExternalVehiculeId(record[1])

	// Column #2
	var state State
	{
		stateStr := record[2]
		var parsingIsOk bool
		state, parsingIsOk = parseStateFromString(stateStr)
		if !parsingIsOk {
			return nil, fmt.Errorf(
				"the vehicle position state is not valid %s",
				stateStr,
			)
		}
	}

	// Parse the heading
	var heading int64
	{
		headingStr := record[6]
		var err error
		heading, err = strconv.ParseInt(headingStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf(
				"the vehicle position heading is not valid %s",
				headingStr,
			)
		}
	}

	// Parse the latitude
	var latitude float64
	{
		latitudeStr := record[7]
		var err error
		latitude, err = strconv.ParseFloat(latitudeStr, 64)
		if err != nil {
			return nil, fmt.Errorf(
				"the vehicle position latitude is not valid %s",
				latitudeStr,
			)
		}
	}

	// Parse the longitude
	var longitude float64
	{
		longitudeStr := record[8]
		var err error
		longitude, err = strconv.ParseFloat(longitudeStr, 64)
		if err != nil {
			return nil, fmt.Errorf(
				"the vehicle position longitude is not valid %s",
				longitudeStr,
			)
		}
	}

	return &VehiclePosition{
		ExternalVehiculeId: extVehicleId,
		State:              state,
		Heading:            int(heading),
		Latitude:           latitude,
		Longitude:          longitude,
		OccupancyStatus:    google_transit.VehiclePosition_NO_DATA_AVAILABLE,
	}, nil
}

type vehiclePositionCsvLineConsumer struct {
	vehiclesPositions []VehiclePosition
}

func makeVehiclePositionCsvLineConsumer() *vehiclePositionCsvLineConsumer {
	return &vehiclePositionCsvLineConsumer{
		vehiclesPositions: make([]VehiclePosition, 0),
	}
}

func (c *vehiclePositionCsvLineConsumer) Consume(csvLine []string, _ *time.Location) error {
	vehiclePosition, err := newVehiclePosition(csvLine)
	if err != nil {
		return err
	}
	c.vehiclesPositions = append(c.vehiclesPositions, *vehiclePosition)
	return nil
}

func (c *vehiclePositionCsvLineConsumer) Terminate() {
}

func loadVehiclePositionsUsingReader(
	reader io.Reader,
) ([]VehiclePosition, error) {

	loadDataOptions := utils.LoadDataOptions{
		Delimiter:     ';',
		NbFields:      vehiclePositionsCsvNumOfFields,
		SkipFirstLine: true, // First line is a header
	}

	vehiclePositionConsumer := makeVehiclePositionCsvLineConsumer()
	err := utils.LoadDataWithOptions(
		reader,
		vehiclePositionConsumer,
		loadDataOptions,
	)
	if err != nil {
		return nil, err
	}

	vehiclePosition := vehiclePositionConsumer.vehiclesPositions

	return vehiclePosition, nil
}
