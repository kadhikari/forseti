package forseti

import (
	"encoding/xml"
	"time"
)

// Temporary structures used only to read FLUX xml for equipments:
type Root struct {
	XMLName xml.Name   `xml:"root"`
	Info    Info       `xml:"infos_generales"`
	Data    Equipments `xml:"donnees"`
}

type Info struct {
	XMLName xml.Name `xml:"infos_generales"`
	Date    string   `xml:"date,attr"`
	Hour    string   `xml:"heure,attr"`
}

type Equipments struct {
	XMLName xml.Name `xml:"donnees"`
	Lines   []Line   `xml:"ligne"`
}

type Line struct {
	XMLName  xml.Name  `xml:"ligne"`
	Code     string    `xml:"code,attr"`
	Label    string    `xml:"libelle,attr"`
	Stations []Station `xml:"station"`
}

type Station struct {
	XMLName    xml.Name           `xml:"station"`
	Equipments []EquipementSource `xml:"equipement"`
}

type EquipementSource struct {
	XMLName xml.Name `xml:"equipement"`
	Type    string   `xml:"type,attr"`
	ID      string   `xml:"code_client,attr"`
	Name    string   `xml:"nom_client,attr"`
	Cause   string   `xml:"cause,attr"`
	Effect  string   `xml:"consequence,attr"`
	Start   string   `xml:"date_debut_indisponibilite,attr"`
	End     string   `xml:"date_remise_service,attr"`
	Hour    string   `xml:"heure_remise_service,attr"`
}

// Structure used to load date from Flucteo
//data.Data.Area.Vehicles
type Data struct{
	Data AreaNode `json:"data"`
}

type AreaNode struct{
	Area VehicleNode `json:"area"`
}

type VehicleNode struct {
	Vehicles []Vehicle `json:"vehicles"`
}

type ProviderNode struct {
	Name string `json:"name,omitempty"`
}

type Vehicle struct {
	PublicId string `json:"publicId,omitempty"`
	Provider ProviderNode `json:"provider,omitempty"`
	Id string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
	Latitude float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Propulsion string `json:"propulsion,omitempty"`
	Battery int `json:"battery,omitempty"`
	Deeplink string `json:"deeplink,omitempty"`
	Attributes []string `json:"attributes,omitempty"`
}

// Structure to load routes from navitia
type NavitiaRoutes struct {
	RouteSchedules []struct {
		Table struct {
			Rows []struct {
				StopPoint struct {
					ID string `json:"id"`
					Name string `json:"Name"`
				} `json:"stop_point"`
				DateTimes []struct {
					DateTime string `json:"date_time"`
					Links    []struct {
						Type  string `json:"type"`
						Value string `json:"value"`
					} `json:"links"`
				} `json:"date_times"`
			} `json:"rows"`
		} `json:"table"`
	} `json:"route_schedules"`
}

// Structure related to predictions
type PredictionData []PredictionNode
type PredictionNode struct {
	Line     	string    `json:"ligne"`
	Sens      	int       `json:"sens"`
	Date      	string    `json:"date"`
	Course    	string    `json:"course"`
	Order     	int       `json:"ordre"`
	StopName 	string    `json:"arret"`
	Charge    	float64   `json:"charge"`
	CreatedAt 	time.Time `json:"created_at"`
}
