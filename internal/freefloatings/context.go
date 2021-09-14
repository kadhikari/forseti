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
	start_page             int
	items_on_page          int
	items_per_page         int
	total_result           int
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

		// Paginate
		d.total_result = len(resp)
		d.items_per_page = param.Count
		d.start_page = param.StartPage

		if param.Count >= 0 && param.StartPage >= 0 {
			first_item := param.StartPage * param.Count
			last_item := first_item + param.Count
			if first_item < len(resp) {
				if last_item < len(resp) {
					resp = resp[first_item:last_item]
				} else {
					resp = resp[first_item:]
				}
			} else {
				resp = nil
			}
			d.items_on_page = len(resp)
		} else {
			resp = nil
			d.items_on_page = 0
		}
	}
	return resp, nil
}

func (d *FreeFloatingsContext) NewPaginate() Paginate {
	return Paginate{
		Start_page:     d.start_page,
		Items_on_page:  d.items_on_page,
		Items_per_page: d.items_per_page,
		Total_result:   d.total_result,
	}
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
