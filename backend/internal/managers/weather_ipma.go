package managers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"od/internal/httpclient"
	"od/internal/parse"
	"od/internal/schedule"
)

// IPMA sends most numbers as strings ("12.3") and a few as numbers, so those fields are `any`
// and converted with parseNumber.

type IPMALocationRecord struct {
	IDAreaAviso   string `json:"idAreaAviso"`
	GlobalIDLocal int    `json:"globalIdLocal"`
	IDDistrito    int    `json:"idDistrito"`
	Latitude      any    `json:"latitude"`
	Longitude     any    `json:"longitude"`
	Local         string `json:"local"`
}

type IPMAAggregateRecord struct {
	DataPrev               string `json:"dataPrev"`
	DataUpdate             string `json:"dataUpdate"`
	DdVento                any    `json:"ddVento"`
	FfVento                any    `json:"ffVento"`
	GlobalIDLocal          int    `json:"globalIdLocal"`
	IDPeriodo              int    `json:"idPeriodo"`
	IDTipoTempo            int    `json:"idTipoTempo"`
	ProbabilidadePrecipita any    `json:"probabilidadePrecipita"`
	TMax                   any    `json:"tMax"`
	TMed                   any    `json:"tMed"`
	TMin                   any    `json:"tMin"`
}

type IPMADailyForecast struct {
	Date                     string   `json:"date"`
	TemperatureMin           *float64 `json:"temperatureMin"`
	TemperatureMax           *float64 `json:"temperatureMax"`
	PrecipitationProbability *float64 `json:"precipitationProbability"`
	WeatherTypeID            int      `json:"weatherTypeId"`
	WindDirection            *string  `json:"windDirection"`
}

type IPMAHourlyForecastEntry struct {
	Time                     string   `json:"time"`
	PeriodHours              int      `json:"periodHours"`
	Temperature              *float64 `json:"temperature"`
	WindSpeed                *float64 `json:"windSpeed"`
	WindDirection            *string  `json:"windDirection"`
	PrecipitationProbability *float64 `json:"precipitationProbability"`
	WeatherTypeID            int      `json:"weatherTypeId"`
}

type IPMAHourlyForecast struct {
	Entries        []IPMAHourlyForecastEntry `json:"entries"`
	TemperatureMin *float64                  `json:"temperatureMin"`
	TemperatureMax *float64                  `json:"temperatureMax"`
}

type IPMAWeatherLocation struct {
	District      string  `json:"district"`
	Local         string  `json:"local"`
	GlobalIDLocal int     `json:"globalIdLocal"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	IDAreaAviso   string  `json:"idAreaAviso"`
}

type IPMAWeatherForecast struct {
	SourceURL   string              `json:"sourceUrl"`
	Location    IPMAWeatherLocation `json:"location"`
	Daily10     []IPMADailyForecast `json:"daily10"`
	Hourly24    IPMAHourlyForecast  `json:"hourly24"`
	LastUpdated time.Time           `json:"lastUpdated"`
}

const defaultIPMASourceURL = "https://www.ipma.pt/pt/otempo/prev.localidade.hora/index.jsp#Coimbra&Miranda%20do%20Corvo"

// WeatherIPMAManager keeps the IPMA forecast of one place (the "#District&Local" of the source url)
// up to date, refreshing it every hour
type WeatherIPMAManager struct {
	http      *httpclient.Client
	sourceURL string

	mutex       sync.RWMutex // weatherData is written by the refresh job and read by the REST route
	weatherData *IPMAWeatherForecast
	refreshJob  *schedule.Job
}

func NewWeatherIPMAManager(http *httpclient.Client, sourceURL string) *WeatherIPMAManager {
	if sourceURL == "" {
		sourceURL = defaultIPMASourceURL
	}
	return &WeatherIPMAManager{http: http, sourceURL: sourceURL}
}

func (manager *WeatherIPMAManager) Start() error {
	slog.Info("weather: starting", "source", manager.sourceURL)
	job, err := schedule.New("0 * * * *", manager.refresh)
	if err != nil {
		return err
	}
	manager.refreshJob = job
	go manager.refresh()
	return nil
}

func (manager *WeatherIPMAManager) Stop() {
	manager.refreshJob.Cancel()
	manager.refreshJob = nil
	manager.setWeatherData(nil)
}

// WeatherData returns the forecast, or nil while nothing was fetched yet
func (manager *WeatherIPMAManager) WeatherData() *IPMAWeatherForecast {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()
	return manager.weatherData
}

func (manager *WeatherIPMAManager) setWeatherData(weatherData *IPMAWeatherForecast) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	manager.weatherData = weatherData
}

func (manager *WeatherIPMAManager) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	weatherData, err := manager.fetch(ctx)
	if err != nil {
		slog.Error("weather: fetching the forecast failed", "err", err)
		return
	}
	manager.setWeatherData(weatherData)
	slog.Info("weather: forecast updated", "at", weatherData.LastUpdated)
}

// parseLocationFromSourceURL reads "#District&Local" from the source url
func (manager *WeatherIPMAManager) parseLocationFromSourceURL() (district string, local string) {
	parsedURL, err := url.Parse(manager.sourceURL)
	if err != nil {
		return "", ""
	}
	district, local, _ = strings.Cut(parsedURL.Fragment, "&") // Fragment is already percent-decoded
	return district, local
}

// parseNumber returns nil for missing values and for the -99 IPMA uses when a value is not available
func parseNumber(value any) *float64 {
	text := parse.String(value)
	if text == "" {
		return nil
	}
	number, err := parse.Float(value)
	if err != nil || (number < 0 && strings.HasPrefix(text, "-99")) {
		return nil
	}
	return &number
}

// stringOrNil mirrors `value || null`
func stringOrNil(value any) *string {
	text := parse.String(value)
	if text == "" {
		return nil
	}
	return &text
}

func (manager *WeatherIPMAManager) resolveLocation(ctx context.Context) (*IPMALocationRecord, error) {
	_, requestedLocal := manager.parseLocationFromSourceURL()
	body, _, err := manager.http.Get(ctx, "https://api.ipma.pt/public-data/forecast/locations.json")
	if err != nil {
		return nil, err
	}
	var locations []IPMALocationRecord
	if err := json.Unmarshal(body, &locations); err != nil {
		return nil, err
	}
	for index := range locations {
		if locations[index].Local == requestedLocal {
			return &locations[index], nil
		}
	}
	return nil, fmt.Errorf("IPMA location not found for %s", requestedLocal)
}

func buildDaily10(records []IPMAAggregateRecord) []IPMADailyForecast {
	dailyRecords := []IPMAAggregateRecord{}
	for _, record := range records {
		if record.IDPeriodo == 24 {
			dailyRecords = append(dailyRecords, record)
		}
	}
	sort.SliceStable(dailyRecords, func(left int, right int) bool {
		return dailyRecords[left].DataPrev < dailyRecords[right].DataPrev
	})
	if len(dailyRecords) > 10 {
		dailyRecords = dailyRecords[:10]
	}

	daily10 := make([]IPMADailyForecast, 0, len(dailyRecords))
	for _, record := range dailyRecords {
		daily10 = append(daily10, IPMADailyForecast{
			Date:                     record.DataPrev,
			TemperatureMin:           parseNumber(record.TMin),
			TemperatureMax:           parseNumber(record.TMax),
			PrecipitationProbability: parseNumber(record.ProbabilidadePrecipita),
			WeatherTypeID:            record.IDTipoTempo,
			WindDirection:            stringOrNil(record.DdVento),
		})
	}
	return daily10
}

func buildHourly24(records []IPMAAggregateRecord) IPMAHourlyForecast {
	now := time.Now().UTC()
	nowUTC := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
	endUTC := nowUTC.Add(24 * time.Hour)

	dailyProbabilityByDate := map[string]*float64{}
	for _, record := range records {
		if record.IDPeriodo == 24 && len(record.DataPrev) >= 10 {
			dailyProbabilityByDate[record.DataPrev[:10]] = parseNumber(record.ProbabilidadePrecipita)
		}
	}

	type recordWithDate struct {
		record IPMAAggregateRecord
		date   time.Time
	}
	items := []recordWithDate{}
	for _, record := range records {
		if record.IDPeriodo != 1 && record.IDPeriodo != 3 {
			continue
		}
		date, err := time.Parse("2006-01-02T15:04:05", record.DataPrev)
		if err != nil {
			continue
		}
		if !date.Before(nowUTC) && date.Before(endUTC) {
			items = append(items, recordWithDate{record: record, date: date})
		}
	}
	sort.SliceStable(items, func(left int, right int) bool {
		return items[left].date.Before(items[right].date)
	})

	entries := make([]IPMAHourlyForecastEntry, 0, len(items))
	for _, item := range items {
		// IPMA uses -99 for unavailable hourly precipitation: fall back to the daily probability
		precipitationProbability := parseNumber(item.record.ProbabilidadePrecipita)
		if precipitationProbability == nil && len(item.record.DataPrev) >= 10 {
			precipitationProbability = dailyProbabilityByDate[item.record.DataPrev[:10]]
		}
		entries = append(entries, IPMAHourlyForecastEntry{
			Time:                     item.record.DataPrev,
			PeriodHours:              item.record.IDPeriodo,
			Temperature:              parseNumber(item.record.TMed),
			WindSpeed:                parseNumber(item.record.FfVento),
			WindDirection:            stringOrNil(item.record.DdVento),
			PrecipitationProbability: precipitationProbability,
			WeatherTypeID:            item.record.IDTipoTempo,
		})
	}

	var temperatureMin, temperatureMax *float64
	for _, entry := range entries {
		if entry.Temperature == nil {
			continue
		}
		if temperatureMin == nil || *entry.Temperature < *temperatureMin {
			temperatureMin = entry.Temperature
		}
		if temperatureMax == nil || *entry.Temperature > *temperatureMax {
			temperatureMax = entry.Temperature
		}
	}
	return IPMAHourlyForecast{Entries: entries, TemperatureMin: temperatureMin, TemperatureMax: temperatureMax}
}

func (manager *WeatherIPMAManager) fetch(ctx context.Context) (*IPMAWeatherForecast, error) {
	district, _ := manager.parseLocationFromSourceURL()
	location, err := manager.resolveLocation(ctx)
	if err != nil {
		return nil, err
	}

	body, _, err := manager.http.Get(ctx, fmt.Sprintf("https://api.ipma.pt/public-data/forecast/aggregate/%d.json", location.GlobalIDLocal))
	if err != nil {
		return nil, err
	}
	var records []IPMAAggregateRecord
	if err := json.Unmarshal(body, &records); err != nil {
		return nil, err
	}

	latitude, _ := parse.Float(location.Latitude)
	longitude, _ := parse.Float(location.Longitude)
	return &IPMAWeatherForecast{
		SourceURL: manager.sourceURL,
		Location: IPMAWeatherLocation{
			District:      district,
			Local:         location.Local,
			GlobalIDLocal: location.GlobalIDLocal,
			Latitude:      latitude,
			Longitude:     longitude,
			IDAreaAviso:   location.IDAreaAviso,
		},
		Daily10:     buildDaily10(records),
		Hourly24:    buildHourly24(records),
		LastUpdated: time.Now(),
	}, nil
}
