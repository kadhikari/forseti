package sirism

import (
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/hove-io/forseti/internal/departures"
	"github.com/stretchr/testify/assert"
)

var fixtureDir string

const SECONDS_PER_HOUR int = 3_600

var EXPECTED_LOCATION *time.Location = time.FixedZone("", 2*SECONDS_PER_HOUR)

func toTimePtr(t time.Time) *time.Time {
	return &t
}

var DEPARTURE_TIME_TESTS = []struct {
	name                  string
	departure             Departure
	expectedDepartureType departures.DepartureType
	expectedDepartureTime time.Time
}{
	{
		name: "check theoretical departure time",
		departure: Departure{
			Id:              ItemId("unused"),
			LineRef:         "unused",
			StopPointRef:    StopPointRef("unused"),
			DirectionType:   departures.DirectionTypeBackward,
			DestinationRef:  "unused",
			DestinationName: "unused",
			AimedDepartureTime: time.Date(
				2022, time.June, 15,
				5, 32, 0, 0,
				EXPECTED_LOCATION,
			),
			ExpectedDepartureTime: nil,
		},
		expectedDepartureType: departures.DepartureTypeTheoretical,
		expectedDepartureTime: time.Date(
			2022, time.June, 15,
			5, 32, 0, 0,
			EXPECTED_LOCATION,
		),
	},
	{
		name: "check estimated departure time 01",
		departure: Departure{
			Id:              ItemId("unused"),
			LineRef:         "unused",
			StopPointRef:    StopPointRef("unused"),
			DirectionType:   departures.DirectionTypeBackward,
			DestinationRef:  "unused",
			DestinationName: "unused",
			AimedDepartureTime: time.Date(
				2022, time.June, 15,
				5, 32, 0, 0,
				EXPECTED_LOCATION,
			),
			ExpectedDepartureTime: toTimePtr(time.Date(
				2022, time.June, 15,
				5, 31, 59, 999_999_999,
				EXPECTED_LOCATION,
			)),
		},
		expectedDepartureType: departures.DepartureTypeEstimated,
		expectedDepartureTime: time.Date(
			2022, time.June, 15,
			5, 31, 59, 999_999_999,
			EXPECTED_LOCATION,
		),
	},
	{
		name: "check estimated departure time 02",
		departure: Departure{
			Id:              ItemId("unused"),
			LineRef:         "unused",
			StopPointRef:    StopPointRef("unused"),
			DirectionType:   departures.DirectionTypeBackward,
			DestinationRef:  "unused",
			DestinationName: "unused",
			AimedDepartureTime: time.Date(
				2022, time.June, 15,
				5, 32, 0, 0,
				EXPECTED_LOCATION,
			),
			ExpectedDepartureTime: toTimePtr(time.Date(
				2022, time.June, 15,
				5, 32, 0, 1,
				EXPECTED_LOCATION,
			)),
		},
		expectedDepartureType: departures.DepartureTypeEstimated,
		expectedDepartureTime: time.Date(
			2022, time.June, 15,
			5, 32, 0, 1,
			EXPECTED_LOCATION,
		),
	},
	{
		name: "check estimated departure time 03 (timezones differ)",
		departure: Departure{
			Id:              ItemId("unused"),
			LineRef:         "unused",
			StopPointRef:    StopPointRef("unused"),
			DirectionType:   departures.DirectionTypeBackward,
			DestinationRef:  "unused",
			DestinationName: "unused",
			AimedDepartureTime: time.Date(
				2022, time.June, 15,
				5, 32, 0, 0,
				EXPECTED_LOCATION,
			),
			ExpectedDepartureTime: toTimePtr(time.Date(
				2022, time.June, 15,
				5, 32, 0, 0,
				time.UTC,
			)),
		},
		expectedDepartureType: departures.DepartureTypeEstimated,
		expectedDepartureTime: time.Date(
			2022, time.June, 15,
			5, 32, 0, 0,
			time.UTC,
		),
	},
}

func TestMain(m *testing.M) {

	fixtureDir = os.Getenv("FIXTUREDIR")
	if fixtureDir == "" {
		panic("$FIXTUREDIR isn't set")
	}

	os.Exit(m.Run())
}

func TestGetDepartureType(t *testing.T) {
	assert := assert.New(t)

	for _, test := range DEPARTURE_TIME_TESTS {
		assert.Equalf(
			test.expectedDepartureType,
			test.departure.GetDepartureType(),
			"the unit test '%s' failed", test.name,
		)
	}
}

func TestGetDepartureTime(t *testing.T) {
	assert := assert.New(t)

	for _, test := range DEPARTURE_TIME_TESTS {
		assert.Equalf(
			test.expectedDepartureTime,
			test.departure.GetDepartureTime(),
			"the unit test '%s' failed", test.name,
		)
	}
}

func TestExtractDeparturesFromFilePath(t *testing.T) {

	assert := assert.New(t)
	expectedLocation := time.FixedZone("", 2*SECONDS_PER_HOUR)
	var tests = []struct {
		xmlFileName                         string
		expectedNumberOfUpdatedDepartures   int
		expectedFirstUpdatedDeparture       *Departure
		expectedLastUpdatedDeparture        *Departure
		expectedNumberOfCancelledDepartures int
		expectedFirstCancelledDeparture     *CancelledDeparture
		expectedLastCancelledDeparture      *CancelledDeparture
	}{
		{
			xmlFileName:                       "notif_siri_lille.xml",
			expectedNumberOfUpdatedDepartures: 138,
			expectedFirstUpdatedDeparture: &Departure{
				Id:              ItemId("SIRI:130784050"),
				LineRef:         "50",
				StopPointRef:    StopPointRef("CAS001"),
				DirectionType:   departures.DirectionTypeBackward,
				DestinationRef:  "LIG114",
				DestinationName: "GARE LILLE FLANDRES",
				AimedDepartureTime: time.Date(
					2022, time.June, 15,
					5, 32, 0, 0,
					expectedLocation,
				),
				ExpectedDepartureTime: toTimePtr(time.Date(
					2022, time.June, 15,
					5, 32, 0, 0,
					expectedLocation,
				)),
			},
			expectedLastUpdatedDeparture: &Departure{
				Id:              ItemId("SIRI:130827335"),
				LineRef:         "CO1",
				StopPointRef:    StopPointRef("CER001"),
				DirectionType:   departures.DirectionTypeForward,
				DestinationRef:  "CAL007",
				DestinationName: "CHU-EURASANTE",
				AimedDepartureTime: time.Date(
					2022, time.June, 15,
					6, 44, 34, 0,
					expectedLocation,
				),
				ExpectedDepartureTime: toTimePtr(time.Date(
					2022, time.June, 15,
					6, 44, 34, 0,
					expectedLocation,
				)),
			},
			expectedNumberOfCancelledDepartures: 0,
			expectedFirstCancelledDeparture:     nil,
			expectedLastCancelledDeparture:      nil,
		},
		{
			xmlFileName:                         "notif_siri_lille_cancellation.xml",
			expectedNumberOfUpdatedDepartures:   0,
			expectedFirstUpdatedDeparture:       nil,
			expectedLastUpdatedDeparture:        nil,
			expectedNumberOfCancelledDepartures: 1,
			expectedFirstCancelledDeparture: &CancelledDeparture{
				Id:           ItemId("SIRI:130784050"),
				StopPointRef: StopPointRef("CAS001"),
			},
			expectedLastCancelledDeparture: nil,
		},
	}
	for _, test := range tests {
		uri, err := url.Parse(fmt.Sprintf("file://%s/data_sirism/%s", fixtureDir, test.xmlFileName))
		assert.Nil(err)

		// Force the variable `time.Local` of the server while the run
		time.Local = time.UTC

		gotUpdatedDepartures, gotCancelledDepartures, err := LoadDeparturesFromFilePath(uri.Path)
		assert.Nil(err)

		// Check updated departures
		{
			expectedNumberOfUpdatedDepartures := test.expectedNumberOfUpdatedDepartures
			assert.Len(
				gotUpdatedDepartures,
				expectedNumberOfUpdatedDepartures,
			)
			// Check the first element
			if expectedNumberOfUpdatedDepartures > 0 {
				assert.Equal(
					*(test.expectedFirstUpdatedDeparture),
					gotUpdatedDepartures[0],
				)
				// Check the last element
				if expectedNumberOfUpdatedDepartures > 1 {
					assert.Equal(
						*(test.expectedLastUpdatedDeparture),
						gotUpdatedDepartures[expectedNumberOfUpdatedDepartures-1],
					)
				}
			}
		}

		// Check cancelled departures
		{
			expectedNumberOfCancelledDepartures := test.expectedNumberOfCancelledDepartures
			assert.Len(
				gotCancelledDepartures,
				expectedNumberOfCancelledDepartures,
			)
			// Check the first element
			if expectedNumberOfCancelledDepartures > 0 {
				assert.Equal(
					*(test.expectedFirstCancelledDeparture),
					gotCancelledDepartures[0],
				)
				// Check the last element
				if expectedNumberOfCancelledDepartures > 1 {
					assert.Equal(
						*(test.expectedLastCancelledDeparture),
						gotCancelledDepartures[expectedNumberOfCancelledDepartures-1],
					)
				}
			}
		}
	}
}

func TestLoadDeparturesFromByteArray(t *testing.T) {
	notificationXml := `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
<soap:Body>
	<ns1:NotifyStopMonitoring xmlns:ns1="http://wsdl.siri.org.uk">
		<ServiceDeliveryInfo
			xmlns:ns2="http://www.ifopt.org.uk/acsb"
			xmlns:ns3="http://www.ifopt.org.uk/ifopt"
			xmlns:ns4="http://datex2.eu/schema/2_0RC1/2_0"
			xmlns:ns5="http://www.siri.org.uk/siri"
			xmlns:ns6="http://wsdl.siri.org.uk/siri">
			<ns5:ResponseTimestamp>2022-06-15T04:36:07.881+02:00</ns5:ResponseTimestamp>
			<ns5:ProducerRef>ILEVIA</ns5:ProducerRef>
			<ns5:ResponseMessageIdentifier>ILEVIA:ResponseMessage::2136076:LOC</ns5:ResponseMessageIdentifier>
			<ns5:RequestMessageRef>SUBREQ</ns5:RequestMessageRef>
		</ServiceDeliveryInfo>
		<Notification
			xmlns:ns2="http://www.ifopt.org.uk/acsb"
			xmlns:ns3="http://www.ifopt.org.uk/ifopt"
			xmlns:ns4="http://datex2.eu/schema/2_0RC1/2_0"
			xmlns:ns5="http://www.siri.org.uk/siri"
			xmlns:ns6="http://wsdl.siri.org.uk/siri">
			<ns5:StopMonitoringDelivery version="2.0:FR-IDF-2.4">
			<ns5:ResponseTimestamp>2022-06-15T04:36:07.881+02:00</ns5:ResponseTimestamp>
			<ns5:RequestMessageRef>SUBHOR_ILEVIA:StopPoint:BP:CAS001:LOC</ns5:RequestMessageRef>
			<ns5:SubscriberRef>KISIO2</ns5:SubscriberRef>
			<ns5:SubscriptionRef>SUBHOR_ILEVIA:StopPoint:BP:CAS001:LOC</ns5:SubscriptionRef>
			<ns5:Status>true</ns5:Status>
			<ns5:MonitoringRef>ILEVIA:StopPoint:BP:CAS001:LOC</ns5:MonitoringRef>
			<ns5:MonitoredStopVisit>
				<ns5:RecordedAtTime>2022-06-15T04:35:56.969+02:00</ns5:RecordedAtTime>
				<ns5:ItemIdentifier>SIRI:130784050</ns5:ItemIdentifier>
				<ns5:MonitoringRef>ILEVIA:StopPoint:BP:CAS001:LOC</ns5:MonitoringRef>
				<ns5:MonitoredVehicleJourney>
					<ns5:LineRef>ILEVIA:Line::50:LOC</ns5:LineRef>
					<ns5:FramedVehicleJourneyRef>
						<ns5:DataFrameRef>ILEVIA:DataFrame::1.0:LOC</ns5:DataFrameRef>
						<ns5:DatedVehicleJourneyRef>ILEVIA:VehicleJourney::50_1_052900_2_LAMJV_8:LOC</ns5:DatedVehicleJourneyRef>
					</ns5:FramedVehicleJourneyRef>
					<ns5:JourneyPatternRef>ILEVIA:JourneyPattern::50_STA001_LIG114_1:LOC</ns5:JourneyPatternRef>
					<ns5:JourneyPatternName>50_STA001_LIG114_1</ns5:JourneyPatternName>
					<ns5:VehicleMode>bus</ns5:VehicleMode>
					<ns5:PublishedLineName>LIGNE 50</ns5:PublishedLineName>
					<ns5:DirectionName>RETOUR</ns5:DirectionName>
					<ns5:OperatorRef>ILEVIA:Operator::ILEVIA:LOC</ns5:OperatorRef>
					<ns5:DestinationRef>ILEVIA:StopPoint:BP:LIG114:LOC</ns5:DestinationRef>
					<ns5:DestinationName>GARE LILLE FLANDRES</ns5:DestinationName>
					<ns5:Monitored>true</ns5:Monitored>
					<ns5:MonitoredCall>
						<ns5:StopPointRef>ILEVIA:StopPoint:BP:CAS001:LOC</ns5:StopPointRef>
						<ns5:Order>3</ns5:Order>
						<ns5:StopPointName>CASSEL</ns5:StopPointName>
						<ns5:VehicleAtStop>false</ns5:VehicleAtStop>
						<ns5:DestinationDisplay>LILLE FLANDRES</ns5:DestinationDisplay>
						<ns5:AimedDepartureTime>2022-06-15T05:32:00.000+02:00</ns5:AimedDepartureTime>
						<ns5:ExpectedDepartureTime>2022-06-15T05:32:30.000+02:00</ns5:ExpectedDepartureTime>
						<ns5:DepartureStatus>onTime</ns5:DepartureStatus>
					</ns5:MonitoredCall>
				</ns5:MonitoredVehicleJourney>
			</ns5:MonitoredStopVisit>
			<ns5:MonitoredStopVisit>
				<ns5:RecordedAtTime>2022-06-15T04:35:56.969+02:00</ns5:RecordedAtTime>
				<ns5:ItemIdentifier>SIRI:130784224</ns5:ItemIdentifier>
				<ns5:MonitoringRef>ILEVIA:StopPoint:BP:CAS001:LOC</ns5:MonitoringRef>
				<ns5:MonitoredVehicleJourney>
					<ns5:LineRef>ILEVIA:Line::50:LOC</ns5:LineRef>
					<ns5:FramedVehicleJourneyRef>
						<ns5:DataFrameRef>ILEVIA:DataFrame::1.0:LOC</ns5:DataFrameRef>
						<ns5:DatedVehicleJourneyRef>ILEVIA:VehicleJourney::50_2_055900_2_LAMJV_8:LOC</ns5:DatedVehicleJourneyRef>
					</ns5:FramedVehicleJourneyRef>
					<ns5:JourneyPatternRef>ILEVIA:JourneyPattern::50_STA001_LIG114_1:LOC</ns5:JourneyPatternRef>
					<ns5:JourneyPatternName>50_STA001_LIG114_1</ns5:JourneyPatternName>
					<ns5:VehicleMode>bus</ns5:VehicleMode>
					<ns5:PublishedLineName>LIGNE 50</ns5:PublishedLineName>
					<ns5:DirectionName>RETOUR</ns5:DirectionName>
					<ns5:OperatorRef>ILEVIA:Operator::ILEVIA:LOC</ns5:OperatorRef>
					<ns5:DestinationRef>ILEVIA:StopPoint:BP:LIG114:LOC</ns5:DestinationRef>
					<ns5:DestinationName>GARE LILLE FLANDRES</ns5:DestinationName>
					<ns5:Monitored>true</ns5:Monitored>
					<ns5:MonitoredCall>
						<ns5:StopPointRef>ILEVIA:StopPoint:BP:CAS001:LOC</ns5:StopPointRef>
						<ns5:Order>3</ns5:Order>
						<ns5:StopPointName>CASSEL</ns5:StopPointName>
						<ns5:VehicleAtStop>false</ns5:VehicleAtStop>
						<ns5:DestinationDisplay>LILLE FLANDRES</ns5:DestinationDisplay>
						<ns5:AimedDepartureTime>2022-06-15T06:02:00.000+02:00</ns5:AimedDepartureTime>
						<ns5:DepartureStatus>onTime</ns5:DepartureStatus>
					</ns5:MonitoredCall>
				</ns5:MonitoredVehicleJourney>
			</ns5:MonitoredStopVisit>
			</ns5:StopMonitoringDelivery>
			<ns5:StopMonitoringDelivery version="2.0:FR-IDF-2.4">
				<ns5:ResponseTimestamp>2022-08-16T04:36:15.640+02:00</ns5:ResponseTimestamp>
				<ns5:RequestMessageRef>SUBHOR_ILEVIA:StopPoint:BP:CAS002:LOC</ns5:RequestMessageRef>
				<ns5:SubscriberRef>KISIO2</ns5:SubscriberRef>
				<ns5:SubscriptionRef>SUBHOR_ILEVIA:StopPoint:BP:CAS002:LOC</ns5:SubscriptionRef>
				<ns5:Status>true</ns5:Status>
				<ns5:MonitoringRef>ILEVIA:StopPoint:BP:CAS002:LOC</ns5:MonitoringRef>
				<ns5:MonitoredStopVisitCancellation>
					<ns5:RecordedAtTime>2022-08-16T04:36:15.591+02:00</ns5:RecordedAtTime>
					<ns5:ItemRef>SIRI:130784051</ns5:ItemRef>
					<ns5:MonitoringRef>ILEVIA:StopPoint:BP:CAS002:LOC</ns5:MonitoringRef>
					<ns5:LineRef>ILEVIA:Line::50:LOC</ns5:LineRef>
					<ns5:VehicleJourneyRef>
						<ns5:DataFrameRef>ILEVIA:DataFrame::1.0:LOC</ns5:DataFrameRef>
						<ns5:DatedVehicleJourneyRef>ILEVIA:VehicleJourney::50_1_052900_2_LAMJV_8:LOC</ns5:DatedVehicleJourneyRef>
					</ns5:VehicleJourneyRef>
				</ns5:MonitoredStopVisitCancellation>
			</ns5:StopMonitoringDelivery>
			<ns5:StopMonitoringDelivery version="2.0:FR-IDF-2.4">
                <ns5:ResponseTimestamp>2023-01-10T12:54:47.270+02:00</ns5:ResponseTimestamp>
                <ns5:RequestMessageRef>SUBHOR_ILEVIA:StopPoint:BP:PRR004:LOC</ns5:RequestMessageRef>
                <ns5:SubscriberRef>KISIO2</ns5:SubscriberRef>
                <ns5:SubscriptionRef>SUBHOR_ILEVIA:StopPoint:BP:PRR004:LOC</ns5:SubscriptionRef>
                <ns5:Status>true</ns5:Status>
                <ns5:MonitoringRef>ILEVIA:StopPoint:BP:CER001:LOC</ns5:MonitoringRef>
                <ns5:MonitoredStopVisit>
                    <ns5:RecordedAtTime>2023-01-10T12:54:47.270+02:00</ns5:RecordedAtTime>
                    <ns5:ItemIdentifier>SIRI:130784050</ns5:ItemIdentifier>
                    <ns5:MonitoringRef>ILEVIA:StopPoint:BP:CER001:LOC</ns5:MonitoringRef>
                    <ns5:MonitoredVehicleJourney>
                        <ns5:LineRef>ILEVIA:Line::L1:LOC</ns5:LineRef>
                        <ns5:FramedVehicleJourneyRef>
                            <ns5:DataFrameRef>ILEVIA:DataFrame::1.0:LOC</ns5:DataFrameRef>
                        </ns5:FramedVehicleJourneyRef>
                        <ns5:JourneyPatternRef>ILEVIA:JourneyPattern::18_NOT001_HDV018_1:LOC</ns5:JourneyPatternRef>
                        <ns5:JourneyPatternName>18_NOT001_HDV018_1</ns5:JourneyPatternName>
                        <ns5:VehicleMode>bus</ns5:VehicleMode>
                        <ns5:PublishedLineName>LIGNE 1</ns5:PublishedLineName>
                        <ns5:DirectionName>ALLER</ns5:DirectionName>
                        <ns5:OperatorRef>ILEVIA:Operator::ILEVIA:LOC</ns5:OperatorRef>
                        <ns5:DestinationRef>ILEVIA:StopPoint:BP:HDV018:LOC</ns5:DestinationRef>
                        <ns5:DestinationName>V. D'ASCQ HOTEL DE VILLE</ns5:DestinationName>
                        <ns5:Monitored>true</ns5:Monitored>
                        <ns5:MonitoredCall>
                            <ns5:StopPointRef>ILEVIA:StopPoint:BP:CER001:LOC</ns5:StopPointRef>
                            <ns5:Order>18</ns5:Order>
                            <ns5:StopPointName>LE CERF</ns5:StopPointName>
                            <ns5:VehicleAtStop>false</ns5:VehicleAtStop>
                            <ns5:DestinationDisplay>HOTEL DE VILLE</ns5:DestinationDisplay>
                            <ns5:AimedDepartureTime>2023-01-10T17:54:47.270+02:00</ns5:AimedDepartureTime>
                            <ns5:ExpectedDepartureTime>2023-01-10T17:54:47.270+02:00</ns5:ExpectedDepartureTime>
                            <ns5:DepartureStatus>Cancelled</ns5:DepartureStatus>
                        </ns5:MonitoredCall>
                    </ns5:MonitoredVehicleJourney>
                </ns5:MonitoredStopVisit>
            </ns5:StopMonitoringDelivery>
		</Notification>
		<SiriExtension
			xmlns:ns2="http://www.ifopt.org.uk/acsb"
			xmlns:ns3="http://www.ifopt.org.uk/ifopt"
			xmlns:ns4="http://datex2.eu/schema/2_0RC1/2_0"
			xmlns:ns5="http://www.siri.org.uk/siri"
			xmlns:ns6="http://wsdl.siri.org.uk/siri"/>
	</ns1:NotifyStopMonitoring>
</soap:Body>
</soap:Envelope>`

	// Force the variable `time.Local` of the server while the run
	time.Local = time.UTC

	assert := assert.New(t)
	getDepartures, getCancelledDepartures, err := LoadDeparturesFromByteArray([]byte(notificationXml))
	assert.Nil(err)

	// Ckeck parsed departures
	{
		var expectedDepartures = []Departure{
			{
				Id:              ItemId("SIRI:130784050"),
				LineRef:         "50",
				StopPointRef:    StopPointRef("CAS001"),
				DirectionType:   departures.DirectionTypeBackward,
				DestinationRef:  "LIG114",
				DestinationName: "GARE LILLE FLANDRES",
				AimedDepartureTime: time.Date(
					2022, time.June, 15,
					5, 32, 00, 000_000_000,
					EXPECTED_LOCATION,
				),
				ExpectedDepartureTime: toTimePtr(time.Date(
					2022, time.June, 15,
					5, 32, 30, 0,
					EXPECTED_LOCATION,
				)),
			},
			{
				Id:              ItemId("SIRI:130784224"),
				LineRef:         "50",
				StopPointRef:    StopPointRef("CAS001"),
				DirectionType:   departures.DirectionTypeBackward,
				DestinationRef:  "LIG114",
				DestinationName: "GARE LILLE FLANDRES",
				AimedDepartureTime: time.Date(
					2022, time.June, 15,
					6, 02, 00, 0,
					EXPECTED_LOCATION,
				),
				ExpectedDepartureTime: nil,
			},
		}

		assert.Equal(
			expectedDepartures,
			getDepartures,
		)
		// Check the departures type (theoretical or estimated)
		assert.Equal(
			departures.DepartureTypeEstimated,
			getDepartures[0].GetDepartureType(),
		)
		assert.Equal(
			departures.DepartureTypeTheoretical,
			getDepartures[1].GetDepartureType(),
		)
	}

	// Ckeck parsed cancelled departures
	{
		var expectedCancelledDepartures = []CancelledDeparture{
			{
				Id:           ItemId("SIRI:130784051"),
				StopPointRef: StopPointRef("CAS002"),
			},
			{
				Id:           ItemId("SIRI:130784050"),
				StopPointRef: StopPointRef("CER001"),
			},
		}
		assert.Equal(
			expectedCancelledDepartures[0],
			getCancelledDepartures[0],
		)
		assert.Equal(
			expectedCancelledDepartures[1],
			getCancelledDepartures[1],
		)
	}

}

func BenchmarkLoadDeparturesFromFilePath(b *testing.B) {
	uri, _ := url.Parse(fmt.Sprintf("file://%s/data_sirism/notif_siri_lille.xml", fixtureDir))
	for n := 0; n < b.N; n++ {
		updatedDepartures, cancelledDepartures, err := LoadDeparturesFromFilePath(uri.Path)
		_ = updatedDepartures
		_ = cancelledDepartures
		_ = err
	}
}
