package weather

func MockWeather() *WeatherModel {
	m := &WeatherModel{
		city:        "San Francisco",
		temperature: 62,
		condition:   "Partly Cloudy",
		humidity:    78,
		forecast: []ForecastDay{
			{Day: "Today", High: 64, Low: 52, Condition: "Partly Cloudy"},
			{Day: "Tue", High: 67, Low: 54, Condition: "Sunny"},
			{Day: "Wed", High: 59, Low: 50, Condition: "Foggy"},
			{Day: "Thu", High: 61, Low: 51, Condition: "Overcast"},
			{Day: "Fri", High: 65, Low: 53, Condition: "Sunny"},
		},
	}
	return m
}
