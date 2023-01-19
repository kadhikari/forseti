package sirism

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/hove-io/forseti/internal/departures"

	sirism_directionname "github.com/hove-io/forseti/internal/sirism/directionname"
	sirism_xml "github.com/hove-io/forseti/internal/sirism/xml"
)

// Type aliases
type ItemId string
type StopPointRef string

type Departure struct {
	Id                    ItemId
	LineRef               string
	StopPointRef          StopPointRef
	DirectionType         departures.DirectionType
	DestinationRef        string
	DestinationName       string
	AimedDepartureTime    time.Time
	ExpectedDepartureTime *time.Time /* Optional attribut */
}

type CancelledDeparture struct {
	Id           ItemId
	StopPointRef StopPointRef
}

func (d *Departure) GetDepartureTime() time.Time {
	if d.ExpectedDepartureTime != nil {
		return *d.ExpectedDepartureTime
	}
	return d.AimedDepartureTime
}

func (d *Departure) GetDepartureType() departures.DepartureType {
	departureType := departures.DepartureTypeTheoretical
	if d.ExpectedDepartureTime != nil {
		departureType = departures.DepartureTypeEstimated
	}
	return departureType
}

func LoadDeparturesFromFilePath(xmlFilePath string) ([]Departure, []CancelledDeparture, error) {
	xmlFile, err := os.Open(xmlFilePath)
	if err != nil {
		return nil, nil, err
	}
	return LoadDeparturesFromOpenedFile(xmlFile)
}

func LoadDeparturesFromOpenedFile(xmlFile *os.File) ([]Departure, []CancelledDeparture, error) {
	xmlBytes, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		return nil, nil, err
	}
	return LoadDeparturesFromByteArray(xmlBytes)
}

func LoadDeparturesFromByteArray(xmlBytes []byte) ([]Departure, []CancelledDeparture, error) {
	var envelope sirism_xml.Envelope
	err := xml.Unmarshal(xmlBytes, &envelope)
	if err != nil {
		return nil, nil, err
	}
	updatedDepartures := make([]Departure, 0)
	cancelledDepartures := make([]CancelledDeparture, 0)

	for _, smDelivery := range envelope.Notification.StopMonitoringDeliveries {
		monitoringRef := string(smDelivery.MonitoringRef)
		// Loop over tags `<MonitoredStopVisit>`
		for _, monitoredStopVisit := range smDelivery.MonitoredStopVisits {
			monitoredVehicleJourney := &monitoredStopVisit.MonitoredVehicleJourney
			innerMonitoringRef := string(monitoredStopVisit.MonitoringRef)
			if monitoringRef != innerMonitoringRef {
				err := fmt.Errorf(
					"the both XML tags <MonitoringRef> mismatch: %s != %s",
					monitoringRef,
					innerMonitoringRef,
				)
				return nil, nil, err
			}
			var updatedDeparture Departure
			{
				lineRef := string(monitoredVehicleJourney.LineRef)
				directionName := monitoredVehicleJourney.DirectionName
				destinationRef := string(monitoredVehicleJourney.DestinationRef)
				destinationName := string(monitoredVehicleJourney.DestinationName)
				monitoredCall := &monitoredVehicleJourney.MonitoredCall
				stopPointRef := string(monitoredCall.StopPointRef)
				departureStatus := strings.ToLower(monitoredCall.DepartureStatus)

				// Manage cancelledDepartures with DepartureStatus == Cancelled or cancelled
				if departureStatus == "cancelled" {
					cancelledDeparture := CancelledDeparture{
						Id:           ItemId(monitoredStopVisit.ItemIdentifier),
						StopPointRef: StopPointRef(monitoringRef),
					}
					cancelledDepartures = append(cancelledDepartures, cancelledDeparture)
					continue
				}

				if monitoringRef != stopPointRef {
					err := fmt.Errorf(
						"the XML tags <MonitoringRef> and <StopPointRef> mismatch: %s != %s",
						monitoringRef,
						stopPointRef,
					)
					return nil, nil, err
				}
				aimedDepartureTime := time.Time(monitoredCall.AimedDepartureTime)
				var expectedDepartureTime *time.Time = nil
				if monitoredCall.ExpectedDepartureTime != nil {
					// A temporary variable is mandatory to get its pointer
					expectedDepartureTimeTmpVar := time.Time(*monitoredCall.ExpectedDepartureTime)
					expectedDepartureTime = &expectedDepartureTimeTmpVar
				}
				var directionType departures.DirectionType
				if directionName == sirism_directionname.DirectionNameAller {
					directionType = departures.DirectionTypeForward
				} else {
					directionType = departures.DirectionTypeBackward
				}
				updatedDeparture = Departure{
					Id:                    ItemId(monitoredStopVisit.ItemIdentifier),
					LineRef:               lineRef,
					StopPointRef:          StopPointRef(monitoringRef),
					DirectionType:         directionType,
					DestinationRef:        destinationRef,
					DestinationName:       destinationName,
					AimedDepartureTime:    aimedDepartureTime,
					ExpectedDepartureTime: expectedDepartureTime,
				}
			}
			updatedDepartures = append(updatedDepartures, updatedDeparture)
		}

		// Loop over tags `<MonitoredStopVisitCancellation>`
		for _, monitoredStopVisitCanc := range smDelivery.MonitoredStopVisitCancellations {
			innerMonitoringRef := string(monitoredStopVisitCanc.MonitoringRef)
			if monitoringRef != innerMonitoringRef {
				err := fmt.Errorf(
					"the both XML tags <MonitoringRef> mismatch: %s != %s",
					monitoringRef,
					innerMonitoringRef,
				)
				return nil, nil, err
			}
			var cancelledDeparture CancelledDeparture
			{

				cancelledDeparture = CancelledDeparture{
					Id:           ItemId(monitoredStopVisitCanc.ItemRef),
					StopPointRef: StopPointRef(monitoringRef),
				}
			}
			cancelledDepartures = append(cancelledDepartures, cancelledDeparture)
		}
	}
	return updatedDepartures, cancelledDepartures, nil
}
