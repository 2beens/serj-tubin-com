package internal

type GeoIpResponse struct {
	Ip          string `json:"ip"`
	CountryCode string `json:"country_code"`
	CountryName string `json:"country_name"`
	RegionCode  string `json:"region_code"`
	RegionName  string `json:"region_name"`
	City        string `json:"city"`
	ZipCode     string `json:"zip_code"`
	TimeZone    string `json:"time_zone"`
	Latitude    string `json:"latitude"`
	Longitude   string `json:"longitude"`
	MetroCode   string `json:"metro_code"`
}
