package equipments

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/text/encoding/charmap"

	"github.com/hove-io/forseti/internal/data"
	"github.com/hove-io/forseti/internal/utils"
)

var location = "Europe/Paris"

// Main loop
func RefreshEquipmentLoop(context *EquipmentsContext, equipmentsURI url.URL,
	equipmentsRefresh, connectionTimeout time.Duration) {
	if len(equipmentsURI.String()) == 0 || equipmentsRefresh.Seconds() <= 0 {
		logrus.Debug("Equipment data refreshing is disabled")
		return
	}
	for {
		err := RefreshEquipments(context, equipmentsURI, connectionTimeout)
		if err != nil {
			logrus.Error("Error while reloading equipment data: ", err)
		} else {
			logrus.Debug("Equipment data updated")
		}
		time.Sleep(equipmentsRefresh)
	}
}

func RefreshEquipments(context *EquipmentsContext, uri url.URL, connectionTimeout time.Duration) error {
	begin := time.Now()
	file, err := utils.GetFile(uri, connectionTimeout)

	if err != nil {
		EquipmentsLoadingErrors.Inc()
		return err
	}

	equipments, err := loadXmlEquipments(file)
	if err != nil {
		EquipmentsLoadingErrors.Inc()
		return err
	}
	context.UpdateEquipments(equipments)
	EquipmentsLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}

func loadXmlEquipments(file io.Reader) ([]EquipmentDetail, error) {

	location, err := time.LoadLocation(location)
	if err != nil {
		return nil, err
	}

	XMLdata, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	decoder := xml.NewDecoder(bytes.NewReader(XMLdata))
	decoder.CharsetReader = getCharsetReader

	var root data.Root
	err = decoder.Decode(&root)
	if err != nil {
		return nil, err
	}

	equipments := make(map[string]EquipmentDetail)
	//Calculate updated_at from Info.Date and Info.Hour
	updatedAt, err := CalculateDate(root.Info, location)
	if err != nil {
		return nil, err
	}
	// for each root.Data.Lines.Stations create an object Equipment
	for _, l := range root.Data.Lines {
		for _, s := range l.Stations {
			for _, e := range s.Equipments {
				ed, err := NewEquipmentDetail(e, updatedAt, location)
				if err != nil {
					return nil, err
				}
				equipments[ed.ID] = *ed
			}
		}
	}

	// Transform the map to array of EquipmentDetail
	equipmentDetails := make([]EquipmentDetail, 0)
	for _, ed := range equipments {
		equipmentDetails = append(equipmentDetails, ed)
	}

	return equipmentDetails, nil
}

func getCharsetReader(charset string, input io.Reader) (io.Reader, error) {
	if charset == "ISO-8859-1" {
		return charmap.ISO8859_1.NewDecoder().Reader(input), nil
	}

	return nil, fmt.Errorf("unknown Charset")
}

// NewEquipmentDetail creates a new EquipmentDetail object from the object EquipementSource
func NewEquipmentDetail(es data.EquipementSource, updatedAt time.Time,
	location *time.Location) (*EquipmentDetail, error) {
	start, err := time.ParseInLocation("2006-01-02", es.Start, location)
	if err != nil {
		return nil, err
	}

	end, err := time.ParseInLocation("2006-01-02", es.End, location)
	if err != nil {
		return nil, err
	}

	hour, err := time.ParseInLocation("15:04:05", es.Hour, location)
	if err != nil {
		return nil, err
	}

	// Add time part to end date
	end = hour.AddDate(end.Year(), int(end.Month())-1, end.Day()-1)

	etype, err := EmbeddedType(es.Type)
	if err != nil {
		return nil, err
	}
	now := time.Now()

	return &EquipmentDetail{
		ID:           es.ID,
		Name:         es.Name,
		EmbeddedType: etype,
		CurrentAvailability: CurrentAvailability{
			Status:    GetEquipmentStatus(start, end, now),
			Cause:     Cause{Label: es.Cause},
			Effect:    Effect{Label: es.Effect},
			Periods:   []Period{Period{Begin: start, End: end}},
			UpdatedAt: updatedAt},
	}, nil
}

// GetEquipmentStatus returns availability of equipment
func GetEquipmentStatus(start time.Time, end time.Time, now time.Time) string {
	if now.Before(start) || now.After(end) {
		return "available"
	} else {
		return "unavailable"
	}

}

// CalculateDate adds date and hour parts
func CalculateDate(info data.Info, location *time.Location) (time.Time, error) {
	date, err := time.ParseInLocation("2006-01-02", info.Date, location)
	if err != nil {
		return time.Now(), err
	}

	hour, err := time.ParseInLocation("15:04:05", info.Hour, location)
	if err != nil {
		return time.Now(), err
	}

	// Add time part to end date
	date = hour.AddDate(date.Year(), int(date.Month())-1, date.Day()-1)

	return date, nil
}
