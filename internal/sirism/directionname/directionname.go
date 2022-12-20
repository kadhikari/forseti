package directionname

import (
	"encoding/xml"
	"fmt"
	"strings"
)

type DirectionName int

const (
	DirectionNameAller  DirectionName = 0
	DirectionNameRetour DirectionName = 1
)

func (dn *DirectionName) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var innerText string
	err := d.DecodeElement(&innerText, &start)
	if err != nil {
		return err
	}

	if strings.EqualFold(innerText, "ALLER") || strings.EqualFold(innerText, "A") {
		*dn = DirectionNameAller
		return nil
	} else if strings.EqualFold(innerText, "RETOUR") || strings.EqualFold(innerText, "R") {
		*dn = DirectionNameRetour
		return nil
	}

	return &UnexpectedDirectionNameError{
		UnexpectedDirectionName: innerText,
	}
}

type UnexpectedDirectionNameError struct {
	UnexpectedDirectionName string
}

// Overriding of the interface `Error`
func (err *UnexpectedDirectionNameError) Error() string {
	errMessage := fmt.Sprintf(
		"the direction name '%s' is not expected",
		err.UnexpectedDirectionName,
	)
	return errMessage
}
