package internal

import "time"

type WeatherDescription struct {
	ID          int    `json:"id"`
	Main        string `json:"main"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

type WeatherMain struct {
	Temp      float64 `json:"temp"`
	FeelsLike float64 `json:"feels_like"`
	TempMin   float64 `json:"temp_min"`
	TempMax   float64 `json:"temp_max"`
	Pressure  float64 `json:"pressure"`
	Humidity  float64 `json:"humidity"`
	SeaLevel  float64 `json:"sea_level"`
	GrndLevel float64 `json:"grnd_level"`
	TempKf    float64 `json:"temp_kf"`
}

type Coordinate struct {
	Lon float64 `json:"lon"`
	Lat float64 `json:"lat"`
}

type Wind struct {
	Speed float64 `json:"speed"`
	Deg   float64 `json:"deg"`
}

type Clouds struct {
	All int `json:"all"`
}

type Rain struct {
	ThreeH float64 `json:"3h"`
}

type WeatherSys struct {
	Type    int    `json:"type"`
	ID      int    `json:"id"`
	Country string `json:"country"`
	Sunrise int    `json:"sunrise"`
	Sunset  int    `json:"sunset"`
	Pod     string `json:"pod"`
}

type WeatherApiResponse struct {
	WeatherInfo

	Cod        int        `json:"cod"`
	Message    int        `json:"message"`
	Cnt        int        `json:"cnt"`
	Coord      Coordinate `json:"coord"`
	Base       string     `json:"base"`
	Visibility int        `json:"visibility"`
	Timezone   int        `json:"timezone"`
	ID         int        `json:"id"`
	Name       string     `json:"name"`
}

type WeatherApi5DaysResponse struct {
	Cod     string        `json:"cod"`
	Message float64       `json:"message"`
	Cnt     float64       `json:"cnt"`
	City    WeatherCity   `json:"city"`
	List    []WeatherInfo `json:"list"`
}

type WeatherInfo struct {
	Dt                  int                  `json:"dt"`
	Main                WeatherMain          `json:"main"`
	WeatherDescriptions []WeatherDescription `json:"weather"`
	Clouds              Clouds               `json:"clouds"`
	Wind                Wind                 `json:"wind"`
	Sys                 WeatherSys           `json:"sys"`
	DtTxt               string               `json:"dt_txt"`
	Rain                Rain                 `json:"rain,omitempty"`
}

// this can be done better, but - no time for hobby :)
func (w *WeatherInfo) Timestamp() time.Time {
	return time.Unix(int64(w.Dt), 0)
}

type WeatherCity struct {
	ID       int        `json:"id"`
	Name     string     `json:"name"`
	State    string     `json:"state"`
	Country  string     `json:"country"`
	Coord    Coordinate `json:"coord"`
	Timezone int        `json:"timezone"`
	Sunrise  int        `json:"sunrise"`
	Sunset   int        `json:"sunset"`
}

// used to return to frontend
type WeatherInfoShort struct {
	Timestamp           int                  `json:"timestamp"`
	WeatherDescriptions []WeatherDescription `json:"descriptions"`
}
