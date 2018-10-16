package main

import (
	"encoding/csv"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"os"
	"sort"
	"time"
)

type Departure struct {
	Line      string
	Stop      string
	Type      string
	VJ        string
	Direction string
	Datetime  time.Time
	Route     string
}

type DataManager struct {
	data *map[string][]Departure
}

func (d *DataManager) UpdateData(data map[string][]Departure) {
	d.data = &data
}

func (d *DataManager) GetData() map[string][]Departure {
	return *d.data
}

func LoadData(filename string) map[string][]Departure {
	location, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		panic(err)
	}

	// Open CSV file
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	data := make(map[string][]Departure)

	// Read File into a Variable
	reader := csv.NewReader(f)
	reader.Comma = ';'
	reader.FieldsPerRecord = 8

	// Loop through lines & turn into object
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		dt, err := time.ParseInLocation("2006-01-02 15:04:05", line[5], location) // aaaa-mm-jjhh:mi:ss
		if err != nil {
			panic(err)
		}

		departure := Departure{
			Stop:      line[0],
			Line:      line[1],
			Type:      line[4],
			Datetime:  dt,
			Direction: line[6],
			VJ:        line[7],
		}
		data[departure.Stop] = append(data[departure.Stop], departure)
	}

	for _, v := range data {
		sort.Slice(v, func(i, j int) bool {
			return v[i].Datetime.Before(v[j].Datetime)
		})
	}
	return data
}

func main() {
	fmt.Println("finish")
	manager := DataManager{}
	manager.UpdateData(LoadData("extract_edylic.txt"))

	r := gin.Default()
	r.GET("/departures", func(c *gin.Context) {
		stopID := c.Query("stop_id")
		c.JSON(200, manager.GetData()[stopID])

	})

	/*
		go func() {
			for {
				manager.UpdateData(LoadData("extract_edylic.txt"))
			}
		}()
	*/

	r.Run()

}
