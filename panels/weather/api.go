package weather

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	geocodeURL = "https://geocoding-api.open-meteo.com/v1/search"
	forecastURL = "https://api.open-meteo.com/v1/forecast"
)

type geoResult struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timezone  string  `json:"timezone"`
}

type geoResponse struct {
	Results []geoResult `json:"results"`
}

type apiResponse struct {
	Current struct {
		Temperature float64 `json:"temperature_2m"`
		WeatherCode int     `json:"weathercode"`
		Humidity    int     `json:"relative_humidity_2m"`
	} `json:"current"`
	Daily struct {
		Time        []string  `json:"time"`
		WeatherCode []int     `json:"weathercode"`
		TempMax     []float64 `json:"temperature_2m_max"`
		TempMin     []float64 `json:"temperature_2m_min"`
	} `json:"daily"`
}

// weatherData holds parsed weather ready for display.
type weatherData struct {
	city        string
	temperature int
	condition   string
	humidity    int
	forecast    []ForecastDay
}

func fetchFromAPI(city string) (*weatherData, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	geo, err := geocodeCity(client, city)
	if err != nil {
		return nil, fmt.Errorf("geocoding %q: %w", city, err)
	}

	apiURL := fmt.Sprintf(
		"%s?latitude=%.4f&longitude=%.4f&current=temperature_2m,weathercode,relative_humidity_2m&daily=weathercode,temperature_2m_max,temperature_2m_min&timezone=%s&forecast_days=5",
		forecastURL,
		geo.Latitude,
		geo.Longitude,
		url.QueryEscape(geo.Timezone),
	)

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather API returned %d", resp.StatusCode)
	}

	var api apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&api); err != nil {
		return nil, err
	}

	data := &weatherData{
		city:        geo.Name,
		temperature: int(api.Current.Temperature),
		condition:   wmoCondition(api.Current.WeatherCode),
		humidity:    api.Current.Humidity,
	}

	for i := range api.Daily.Time {
		t, _ := time.Parse("2006-01-02", api.Daily.Time[i])
		data.forecast = append(data.forecast, ForecastDay{
			Day:       dayLabel(t),
			High:      int(api.Daily.TempMax[i]),
			Low:       int(api.Daily.TempMin[i]),
			Condition: wmoCondition(api.Daily.WeatherCode[i]),
		})
	}

	return data, nil
}

func geocodeCity(client *http.Client, city string) (*geoResult, error) {
	reqURL := fmt.Sprintf("%s?name=%s&count=1&language=en&format=json", geocodeURL, url.QueryEscape(city))
	resp, err := client.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var geo geoResponse
	if err := json.NewDecoder(resp.Body).Decode(&geo); err != nil {
		return nil, err
	}
	if len(geo.Results) == 0 {
		return nil, fmt.Errorf("city %q not found", city)
	}
	return &geo.Results[0], nil
}

func dayLabel(t time.Time) string {
	now := time.Now()
	switch {
	case sameDay(t, now):
		return "Today"
	case sameDay(t, now.Add(24*time.Hour)):
		return "Tmrw"
	default:
		return t.Format("Mon")
	}
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

// wmoCondition maps WMO weather interpretation codes to a short condition string.
func wmoCondition(code int) string {
	switch {
	case code == 0:
		return "Clear"
	case code <= 2:
		return "Partly Cloudy"
	case code == 3:
		return "Overcast"
	case code <= 48:
		return "Foggy"
	case code <= 55:
		return "Drizzle"
	case code <= 65:
		return "Rain"
	case code <= 75:
		return "Snow"
	case code <= 82:
		return "Showers"
	case code <= 99:
		return "Thunderstorm"
	default:
		return "Unknown"
	}
}
