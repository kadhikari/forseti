package sytralrt

import "encoding/xml"

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
