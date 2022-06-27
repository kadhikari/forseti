package freefloatings

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hove-io/forseti/internal/utils"
)

type FreeFloatingsContext struct {
	freeFloatings          *[]FreeFloating
	lastFreeFloatingUpdate time.Time
	freeFloatingsMutex     sync.RWMutex
	loadFreeFloatingData   bool
	packageName            string
	RefreshTime            time.Duration
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
	return d.RefreshTime.String()
}

func (d *FreeFloatingsContext) GetPackageName() string {
	d.freeFloatingsMutex.Lock()
	defer d.freeFloatingsMutex.Unlock()
	return d.packageName
}

func (d *FreeFloatingsContext) SetPackageName(pathPackage string) {
	d.freeFloatingsMutex.Lock()
	defer d.freeFloatingsMutex.Unlock()

	paths := strings.Split(pathPackage, "/")
	size := len(paths)
	d.packageName = paths[size-1]
}

//nolint
func (d *FreeFloatingsContext) GetFreeFloatings(param *FreeFloatingRequestParameter) (freeFloatings []FreeFloating, paginate utils.Paginate, e error) {
	var paginate_freefloatings utils.Paginate
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

			// Select according to provider
			if len(param.ProviderName) > 0 && strings.ToLower(param.ProviderName) != strings.ToLower(ff.ProviderName) {
				continue
			}

			// Keep the wanted object
			if keep {
				resp = append(resp, ff)
			}
		}
		sort.Sort(ByDistance(resp))

		// Paginate
		paginate, indexS, indexE := utils.PaginateEndPoint(len(resp), param.Count, param.StartPage)
		paginate_freefloatings = paginate
		if indexS >= 0 {
			resp = resp[indexS:indexE]
		} else {
			resp = resp[:0]
		}
	}
	return resp, paginate_freefloatings, nil
}

type ByDistance []FreeFloating

func (ff ByDistance) Len() int           { return len(ff) }
func (ff ByDistance) Less(i, j int) bool { return ff[i].Distance < ff[j].Distance }
func (ff ByDistance) Swap(i, j int)      { ff[i], ff[j] = ff[j], ff[i] }
