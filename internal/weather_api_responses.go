package internal

type Weather struct {
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
	Pressure  int     `json:"pressure"`
	Humidity  int     `json:"humidity"`
	SeaLevel  int     `json:"sea_level"`
	GrndLevel int     `json:"grnd_level"`
	TempKf    float64 `json:"temp_kf"`
}

type Coordinate struct {
	Lon float64 `json:"lon"`
	Lat float64 `json:"lat"`
}

type Wind struct {
	Speed float64 `json:"speed"`
	Deg   int     `json:"deg"`
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
	Cod        int         `json:"cod"`
	Message    int         `json:"message"`
	Cnt        int         `json:"cnt"`
	Coord      Coordinate  `json:"coord"`
	Base       string      `json:"base"`
	Visibility int         `json:"visibility"`
	Timezone   int         `json:"timezone"`
	ID         int         `json:"id"`
	Name       string      `json:"name"`
	Dt         int         `json:"dt"`
	Main       WeatherMain `json:"main"`
	Weather    []Weather   `json:"weather"`
	Clouds     Clouds      `json:"clouds"`
	Wind       Wind        `json:"wind"`
	Sys        WeatherSys  `json:"sys"`
}

type WeatherApi5DaysResponse struct {
	Cod     string      `json:"cod"`
	Message int         `json:"message"`
	Cnt     int         `json:"cnt"`
	City    WeatherCity `json:"city"`
	List    []struct {
		Dt      int         `json:"dt"`
		Main    WeatherMain `json:"main"`
		Weather []Weather   `json:"weather"`
		Clouds  Clouds      `json:"clouds"`
		Wind    Wind        `json:"wind"`
		Sys     WeatherSys  `json:"sys"`
		DtTxt   string      `json:"dt_txt"`
		Rain    Rain        `json:"rain,omitempty"`
	} `json:"list"`
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