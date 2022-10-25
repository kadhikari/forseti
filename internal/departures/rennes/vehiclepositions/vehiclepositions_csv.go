package vehiclepositions

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/hove-io/forseti/internal/utils"
	"github.com/sirupsen/logrus"

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

type State string

const (
	State_ATE     State = State("ATE")  /* Atelier */
	State_INC           = State("INC")  /* Inconnu */
	State_HL            = State("HL")   /* Hors ligne */
	State_HS            = State("HS")   /* Hors service */
	State_LIGN          = State("LIGN") /* En Ligne */
	State_DEVP          = State("DEVP") /* Déviation programmée */
	State_DEV           = State("DEV")  /* Déviation non-programmée */
	State_GARE          = State("GARE") /* Garé */
	State_HLPS          = State("HLPS") /* HLP de sortie dépot */
	State_HLPR          = State("HLPR") /* HLP de rentrée dépot */
	State_HLP           = State("HLP")  /* HLP intercourse */
	State_SPEC          = State("SPEC") /* Service spécial */
	State_TDEP          = State("TDEP") /* Terminus départ */
	State_TARR          = State("TARR") /* Terminus arrivée */
	State_UNKNOWN       = State("UNKNOWN")
)

var (
	statesMap = map[string]State{
		string(State_ATE):  State_ATE,
		string(State_INC):  State_INC,
		string(State_HL):   State_HL,
		string(State_HS):   State_HS,
		string(State_LIGN): State_LIGN,
		string(State_DEVP): State_DEVP,
		string(State_DEV):  State_DEV,
		string(State_GARE): State_GARE,
		string(State_HLPS): State_HLPS,
		string(State_HLPR): State_HLPR,
		string(State_HLP):  State_HLP,
		string(State_SPEC): State_SPEC,
		string(State_TDEP): State_TDEP,
		string(State_TARR): State_TARR,
	}
)

func (s *State) IsInOperation() bool {
	isInOperation := false
	switch *s {
	case State_LIGN, State_DEVP, State_DEV, State_TDEP, State_TARR:
		isInOperation = true
	}
	return isInOperation
}

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
			state = State_UNKNOWN
			logrus.Warnf(
				"the state if the vehicle %s is unknown (=%s), it is replaced by %s",
				string(extVehicleId),
				stateStr,
				string(state),
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
