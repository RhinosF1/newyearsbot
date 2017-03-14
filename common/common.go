package common

import (
	"fmt"
	"log"
	"strconv"
)

//Geocode is Google maps api
const Geocode = "http://maps.googleapis.com/maps/api/geocode/json?"

//Timezone is Google maps time api
const Timezone = "https://maps.googleapis.com/maps/api/timezone/json?"

//Gmap holds map json
type Gmap struct {
	Results []struct {
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			}
		}
	}
	Status string
}

//Gtime holds location tz info
type Gtime struct {
	Status    string
	RawOffset int
	DstOffset int
}

//TZ holds infor for Time Zone
type TZ struct {
	Countries []struct {
		Name   string   `json:"name"`
		Cities []string `json:"cities"`
	} `json:"countries"`
	Offset string `json:"offset"`
}

func (t TZ) String() (x string) {
	for i, k := range t.Countries {
		x += fmt.Sprintf("%s", k.Name)
		for i, k1 := range k.Cities {
			if k1 == "" {
				continue
			}
			if i == 0 {
				x += " ("
			}
			x += fmt.Sprintf("%s", k1)
			if i >= 0 && i < len(k.Cities)-1 {
				x += ", "
			}
			if i == len(k.Cities)-1 {
				x += ")"
			}
		}
		if i < len(t.Countries)-1 {
			x += ", "
		}
	}
	return
}

//TZS is a slice of TZ
type TZS []TZ

func (t TZS) Len() int {
	return len(t)
}

func (t TZS) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t TZS) Less(i, j int) bool {
	x, err := strconv.ParseFloat(t[i].Offset, 64)
	if err != nil {
		log.Fatal(err)
	}
	y, err := strconv.ParseFloat(t[j].Offset, 64)
	if err != nil {
		log.Fatal(err)
	}
	if x < y {
		return true
	}
	return false
}