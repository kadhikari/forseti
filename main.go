package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	file, err := getFileWithSftp(SftpFileConfig{
		Host:     "172.17.0.3:22",
		User:     "sytral",
		Password: "pass",
		File:     "extract_edylic.txt",
	})
	if err != nil {
		panic(err)
	}
	var manager DataManager
	manager.UpdateDepartures(LoadData(file))

	r := gin.Default()
	r.GET("/departures", func(c *gin.Context) {
		stopID := c.Query("stop_id")
		departures, err := manager.GetDeparturesByStop(stopID)
		if err != nil {
			c.JSON(503, departures) //TODO: return a pretty error response
			return
		}
		c.JSON(200, departures)

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
