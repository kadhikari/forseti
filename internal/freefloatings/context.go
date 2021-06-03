package freefloatings

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/CanalTP/forseti/internal/data"
	"github.com/CanalTP/forseti/internal/utils"
)

type FreeFloatingsContext struct {
	freeFloatings          *[]FreeFloating
	lastFreeFloatingUpdate time.Time
	freeFloatingsMutex     sync.RWMutex
	loadFreeFloatingData   bool
	refreshTime            time.Duration
}

func (d *FreeFloatingsContext) ManageFreeFloatingsStatus(activate bool) {
	d.freeFloatingsMutex.Lock()
	defer d.freeFloatingsMutex.Unlock()

	d.loadFreeFloatingData = activate
}

func (d *FreeFloatingsContext) LoadFreeFloatingsData() bool {
	d.freeFloatingsMutex.RLock()
	defer d.freeFloatingsMutex.RUnlock()
	return d.loadFreeFloatingData
}

func (d *FreeFloatingsContext) UpdateFreeFloating(freeFloatings []FreeFloating) {
	d.freeFloatingsMutex.Lock()
	defer d.freeFloatingsMutex.Unlock()

	d.freeFloatings = &freeFloatings
	d.lastFreeFloatingUpdate = time.Now()
}

func (d *FreeFloatingsContext) GetLastFreeFloatingsDataUpdate() time.Time {
	d.freeFloatingsMutex.RLock()
	defer d.freeFloatingsMutex.RUnlock()

	return d.lastFreeFloatingUpdate
}

func (d *FreeFloatingsContext) GetRereshTime() string {
	d.freeFloatingsMutex.Lock()
	defer d.freeFloatingsMutex.Unlock()
	return d.refreshTime.String()
}

//nolint
func (d *FreeFloatingsContext) GetFreeFloatings(param *FreeFloatingRequestParameter) (freeFloatings []FreeFloating, e error) {
	resp := make([]FreeFloating, 0)
	{
		d.freeFloatingsMutex.RLock()
		defer d.freeFloatingsMutex.RUnlock()

		if d.freeFloatings == nil {
			e = fmt.Errorf("No free-floatings in the data")
			return
		}

		// Implementation of filters: distance, type[]
		for _, ff := range *d.freeFloatings {
			// Filter on type[]
			keep := keepIt(ff, param.Types)

			if !keep {
				continue
			}

			// Calculate distance from coord in the request
			distance := utils.CoordDistance(param.Coord.Lat, param.Coord.Lon, ff.Coord.Lat, ff.Coord.Lon)
			ff.Distance = math.Round(distance)
			if int(distance) > param.Distance {
				continue
			}

			// Keep the wanted object
			if keep {
				resp = append(resp, ff)
			}
		}
		sort.Sort(ByDistance(resp))
		if len(resp) > param.Count {
			resp = resp[:param.Count]
		}
	}
	return resp, nil
}

type ByDistance []FreeFloating

func (ff ByDistance) Len() int           { return len(ff) }
func (ff ByDistance) Less(i, j int) bool { return ff[i].Distance < ff[j].Distance }
func (ff ByDistance) Swap(i, j int)      { ff[i], ff[j] = ff[j], ff[i] }

// NewFreeFloating creates a new FreeFloating object from the object Vehicle
func NewFreeFloating(ve data.Vehicle) *FreeFloating {
	return &FreeFloating{
		PublicId:     ve.PublicId,
		ProviderName: ve.Provider.Name,
		Id:           ve.Id,
		Type:         ve.Type,
		Coord:        Coord{Lat: ve.Latitude, Lon: ve.Longitude},
		Propulsion:   ve.Propulsion,
		Battery:      ve.Battery,
		Deeplink:     ve.Deeplink,
		Attributes:   ve.Attributes,
	}
}
