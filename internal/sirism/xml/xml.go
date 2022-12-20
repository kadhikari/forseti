package xml

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/hove-io/forseti/internal/sirism/directionname"
	"github.com/sirupsen/logrus"
)

type Envelope struct {
	XMLName      xml.Name     `xml:"Envelope"`
	Notification Notification `xml:"Body>NotifyStopMonitoring>Notification"`
}

func (e *Envelope) GetTotalNumberOfMonitoredStopVisits() int {
	return e.Notification.getTotalNumberOfMonitoredStopVisits()
}

func (e *Envelope) GetTotalNumberOfMonitoredStopVisitCancellations() int {
	return e.Notification.getTotalNumberOfMonitoredStopVisitCancellations()
}

type Notification struct {
	XMLName                  xml.Name                 `xml:"Notification"`
	StopMonitoringDeliveries []StopMonitoringDelivery `xml:"StopMonitoringDelivery"`
}

func (n *Notification) getTotalNumberOfMonitoredStopVisits() int {
	sum := 0
	for _, stopMonitoringDelivery := range n.StopMonitoringDeliveries {
		sum += len(stopMonitoringDelivery.MonitoredStopVisits)
	}
	return sum
}

func (n *Notification) getTotalNumberOfMonitoredStopVisitCancellations() int {
	sum := 0
	for _, stopMonitoringDelivery := range n.StopMonitoringDeliveries {
		sum += len(stopMonitoringDelivery.MonitoredStopVisitCancellations)
	}
	return sum
}

type StopMonitoringDelivery struct {
	XMLName                         xml.Name                         `xml:"StopMonitoringDelivery"`
	MonitoringRef                   StopPointRef                     `xml:"MonitoringRef"`
	MonitoredStopVisits             []MonitoredStopVisit             `xml:"MonitoredStopVisit"`
	MonitoredStopVisitCancellations []MonitoredStopVisitCancellation `xml:"MonitoredStopVisitCancellation"`
}

func (smd *StopMonitoringDelivery) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {

	var monitoringRef StopPointRef
	monitoredStopVisits := make([]MonitoredStopVisit, 0)
	monitoredStopVisitCancellations := make([]MonitoredStopVisitCancellation, 0)

	for token, err := d.Token(); err == nil; token, err = d.Token() {
		if err != nil {
			return err
		}
		switch start := token.(type) {
		case xml.StartElement:
			tag := start.Name.Local
			switch tag {
			case "MonitoringRef":
				err = d.DecodeElement(&monitoringRef, &start)
				if err != nil {
					return err
				}
			case "MonitoredStopVisit":
				var monitoredStopVisit MonitoredStopVisit
				err = d.DecodeElement(&monitoredStopVisit, &start)
				switch e := err.(type) {
				case *directionname.UnexpectedDirectionNameError:
					// skip the current token
					logrus.Warnf(
						"unexpected DirectionName %s, the current MonitoredStopVisit is skipped",
						e.UnexpectedDirectionName,
					)
				case nil:
					monitoredStopVisits = append(
						monitoredStopVisits,
						monitoredStopVisit,
					)
				default:
					logrus.Errorf(
						"an unexpected error occurred while the XML parsing, the current MonitoredStopVisit is skipped: %v",
						err,
					)
				}
			case "MonitoredStopVisitCancellation":
				var monitoredStopVisitCancellation MonitoredStopVisitCancellation
				err = d.DecodeElement(&monitoredStopVisitCancellation, &start)
				if err != nil {
					logrus.Errorf(
						"an unexpected error occurred while the XML parsing, the current MonitoredStopVisitCancellation is skipped: %v",
						err,
					)
				} else {
					monitoredStopVisitCancellations = append(
						monitoredStopVisitCancellations,
						monitoredStopVisitCancellation,
					)
				}
			}
		}
	}
	smd.XMLName = start.Name
	smd.MonitoringRef = monitoringRef
	{
		// Default behaviour of the function `UnmarshalXML()`:
		// the slice is not allocated if no matching element have been found
		if len(monitoredStopVisits) == 0 {
			monitoredStopVisits = nil
		}
		smd.MonitoredStopVisits = monitoredStopVisits
	}
	{
		// Default behaviour of the function  `UnmarshalXML()`:
		// the slice is not allocated if not matching element have been found
		if len(monitoredStopVisitCancellations) == 0 {
			monitoredStopVisitCancellations = nil
		}
		smd.MonitoredStopVisitCancellations = monitoredStopVisitCancellations
	}
	return nil
}

type StopPointRef string

func (mr *StopPointRef) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var innerText string
	err := d.DecodeElement(&innerText, &start)
	if err != nil {
		return err
	}

	splittedInnerText := strings.Split(innerText, ":")
	const EXPECTED_NUM_OF_PARTS int = 5
	if len(splittedInnerText) != EXPECTED_NUM_OF_PARTS {
		return fmt.Errorf("the `MonitoringRef` is not well formatted: %s", innerText)
	}
	if splittedInnerText[1] != "StopPoint" {
		return fmt.Errorf("the `MonitoringRef` is not well formatted: %s", innerText)
	}
	*mr = StopPointRef(splittedInnerText[3])
	return nil
}

type MonitoredStopVisit struct {
	XMLName                 xml.Name                `xml:"MonitoredStopVisit"`
	ItemIdentifier          string                  `xml:"ItemIdentifier"`
	MonitoringRef           StopPointRef            `xml:"MonitoringRef"`
	MonitoredVehicleJourney MonitoredVehicleJourney `xml:"MonitoredVehicleJourney"`
}

type MonitoredVehicleJourney struct {
	XMLName         xml.Name                    `xml:"MonitoredVehicleJourney"`
	LineRef         LineRef                     `xml:"LineRef"`
	DirectionName   directionname.DirectionName `xml:"DirectionName"`
	DestinationRef  StopPointRef                `xml:"DestinationRef"`
	DestinationName string                      `xml:"DestinationName"`
	MonitoredCall   MonitoredCall               `xml:"MonitoredCall"`
}

type MonitoredStopVisitCancellation struct {
	XMLName       xml.Name     `xml:"MonitoredStopVisitCancellation"`
	ItemRef       string       `xml:"ItemRef"`
	MonitoringRef StopPointRef `xml:"MonitoringRef"`
}

type LineRef string

func (lr *LineRef) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var innerText string
	err := d.DecodeElement(&innerText, &start)
	if err != nil {
		return err
	}

	splittedInnerText := strings.Split(innerText, ":")
	const EXPECTED_NUM_OF_PARTS int = 5
	if len(splittedInnerText) != EXPECTED_NUM_OF_PARTS {
		return fmt.Errorf("the `LineRef` is not well formatted: %s", innerText)
	}
	if splittedInnerText[1] != "Line" {
		return fmt.Errorf("the `LineRef` is not well formatted: %s", innerText)
	}
	*lr = LineRef(splittedInnerText[3])
	return nil
}

type MonitoredCall struct {
	XMLName               xml.Name     `xml:"MonitoredCall"`
	StopPointRef          StopPointRef `xml:"StopPointRef"`
	AimedDepartureTime    customTime   `xml:"AimedDepartureTime"`
	ExpectedDepartureTime *customTime  `xml:"ExpectedDepartureTime,omitempty"`
}

type customTime time.Time

func (ct *customTime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	const CUSTUM_TIME_LAYOUT string = "2006-01-02T15:04:05.000Z07:00"
	var s string
	err := d.DecodeElement(&s, &start)
	if err != nil {
		return err
	}

	t, err := time.Parse(CUSTUM_TIME_LAYOUT, s)
	if err != nil {
		return err
	}
	*ct = customTime(t)
	return nil
}
