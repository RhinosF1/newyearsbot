package nyb

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

//TZ holds infor for Time Zone
type TZ struct {
	Countries []struct {
		Name   string   `json:"name"`
		Cities []string `json:"cities"`
	} `json:"countries"`
	Offset string `json:"offset"`
}

func (t TZ) String() (x string) {
	for i, country := range t.Countries {
		x += fmt.Sprintf("%s", country.Name)
		for i, city := range country.Cities {
			if city == "" {
				continue
			}
			if i == 0 {
				x += " ("
			}
			x += fmt.Sprintf("%s", city)
			if i < len(country.Cities)-1 {
				x += ", "
			}
			if i == len(country.Cities)-1 {
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

//IrcChans is a custom flag
type IrcChans []string

func (i *IrcChans) String() string {
	return fmt.Sprint(*i)
}

//Set satisfies flag Interface?
func (i *IrcChans) Set(value string) error {
	if len(*i) > 0 {
		return errors.New("interval flag already set")
	}
	for _, dt := range strings.Split(value, ",") {
		*i = append(*i, strings.TrimSpace(dt))
	}
	return nil
}

//Timer struct
type Timer struct {
	C      chan bool
	Target time.Time
	stop   chan bool
	ticker *time.Ticker
}

//NewTimer returns ticker based timer
//We need this to take into account time taken in suspend and hibernation
//Golang time.Timers suck
func NewTimer(dur time.Duration) *Timer {
	t := &Timer{}
	t.C = make(chan bool)
	t.stop = make(chan bool)
	t.Target = time.Now().UTC().Add(dur)
	t.ticker = time.NewTicker(time.Millisecond * 100)
	go func(t *Timer) {
		defer t.ticker.Stop()
		for range t.ticker.C {
			select {
			case <-t.stop:
				return
			default:
				if time.Now().UTC().After(t.Target) {
					close(t.C)
					return
				}
			}
		}
	}(t)
	return t
}

//Stop stops the timer
func (t *Timer) Stop() {
	select {
	case <-t.stop:
		return
	default:
		close(t.stop)
	}
}

//NominatimResult ...
type NominatimResult struct {
	Lat         string
	Lon         string
	DisplayName string `json:"Display_name"`
}

//NominatimResults ...
type NominatimResults []NominatimResult

//NominatimGeoCode const
const NominatimGeoCode = "/search?"

//Avoid allocations, reuse stuff
var nominatim = struct {
	cache map[string][]byte
	req   *http.Request
	sync.RWMutex
	http.Client
}{
	cache: make(map[string][]byte),
}

func init() {
	nominatim.req, _ = http.NewRequest("GET", "", nil)
	nominatim.req.Header.Set("User-Agent", "newyearsbot (irc bot) https://github.com/ugjka/newyearsbot")
}

//NominatimGetter make an api request
func NominatimGetter(urlString string) (data []byte, err error) {
	nominatim.RLock()
	if v, ok := nominatim.cache[urlString]; ok {
		nominatim.RUnlock()
		log.Println("Nominatim: using cached result")
		return v, nil
	}
	nominatim.RUnlock()
	url, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}
	nominatim.req.URL = url
	get, err := nominatim.Do(nominatim.req)
	if err != nil {
		return
	}
	defer get.Body.Close()
	if get.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Status: %d", get.StatusCode)
	}
	data, err = ioutil.ReadAll(get.Body)
	if err != nil {
		return
	}
	nominatim.Lock()
	nominatim.cache[urlString] = data
	nominatim.Unlock()
	return
}