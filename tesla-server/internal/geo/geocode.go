package geo

import (
	"encoding/json"
	"fmt"
	"math"
	"tesla-server/config"
	"tesla-server/internal/database"
	"tesla-server/models"

	"github.com/go-resty/resty/v2"
)

var client = resty.New()

type GeoResult struct {
	Address  string
	City     string
	District string
	PoiName  string
}

func ReverseGeocode(lat, lng float64) string {
	result := ReverseGeocodeDetail(lat, lng)
	return result.Address
}

func ReverseGeocodeDetail(lat, lng float64) *GeoResult {
	roundedLat := math.Round(lat*1000) / 1000
	roundedLng := math.Round(lng*1000) / 1000

	var cache models.GeoCache
	if err := database.DB.Where("latitude = ? AND longitude = ?", roundedLat, roundedLng).First(&cache).Error; err == nil {
		return &GeoResult{
			Address: cache.Address,
		}
	}

	cfg := config.Load()
	if cfg.Map.TencentKey == "" {
		return &GeoResult{}
	}

	resp, err := client.R().
		SetQueryParams(map[string]string{
			"location": fmt.Sprintf("%.6f,%.6f", lat, lng),
			"key":      cfg.Map.TencentKey,
			"output":   "json",
			"get_poi":  "1",
		}).
		Get("https://apis.map.qq.com/ws/geocoder/v1/")

	if err != nil {
		return &GeoResult{}
	}

	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Result  struct {
			Address     string `json:"address"`
			AddressComponent struct {
				City     string `json:"city"`
				District string `json:"district"`
			} `json:"address_component"`
			Pois []struct {
				Title string `json:"title"`
			} `json:"pois"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp.Body(), &result); err != nil || result.Status != 0 {
		return &GeoResult{}
	}

	geoResult := &GeoResult{
		Address:  result.Result.Address,
		City:     result.Result.AddressComponent.City,
		District: result.Result.AddressComponent.District,
	}

	if len(result.Result.Pois) > 0 {
		geoResult.PoiName = result.Result.Pois[0].Title
	}

	if geoResult.Address != "" {
		database.DB.Where("latitude = ? AND longitude = ?", roundedLat, roundedLng).
			FirstOrCreate(&models.GeoCache{
				Latitude:  roundedLat,
				Longitude: roundedLng,
				Address:   geoResult.Address,
			})
	}

	return geoResult
}

func CalculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	cfg := config.Load()
	if cfg.Map.TencentKey == "" {
		return calculateDistanceSimple(lat1, lng1, lat2, lng2)
	}

	resp, err := client.R().
		SetQueryParams(map[string]string{
			"from":    fmt.Sprintf("%.6f,%.6f", lat1, lng1),
			"to":      fmt.Sprintf("%.6f,%.6f", lat2, lng2),
			"key":     cfg.Map.TencentKey,
			"output":  "json",
		}).
		Get("https://apis.map.qq.com/ws/distance/v1/")

	if err != nil {
		return calculateDistanceSimple(lat1, lng1, lat2, lng2)
	}

	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Result  struct {
			Elements []struct {
				Distance int `json:"distance"`
			} `json:"elements"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp.Body(), &result); err != nil || result.Status != 0 {
		return calculateDistanceSimple(lat1, lng1, lat2, lng2)
	}

	if len(result.Result.Elements) > 0 {
		return float64(result.Result.Elements[0].Distance) / 1000.0
	}

	return calculateDistanceSimple(lat1, lng1, lat2, lng2)
}

func calculateDistanceSimple(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371.0
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func transformLat(x, y float64) float64 {
	ret := -100.0 + 2.0*x + 3.0*y + 0.2*y*y + 0.1*x*y + 0.2*math.Abs(math.Sqrt(x*x+y*y))
	ret += (20.0*math.Sin(6.0*x*math.Pi) + 20.0*math.Sin(2.0*x*math.Pi)) * 2.0 / 3.0
	ret += (20.0*math.Sin(y*math.Pi) + 40.0*math.Sin(y/3.0*math.Pi)) * 2.0 / 3.0
	ret += (160.0*math.Sin(y/12.0*math.Pi) + 320.0*math.Sin(y*math.Pi/30.0)) * 2.0 / 3.0
	return ret
}

func transformLng(x, y float64) float64 {
	ret := 300.0 + x + 2.0*y + 0.1*x*x + 0.1*x*y + 0.1*math.Abs(x)
	ret += (20.0*math.Sin(6.0*x*math.Pi) + 20.0*math.Sin(2.0*x*math.Pi)) * 2.0 / 3.0
	ret += (20.0*math.Sin(x*math.Pi) + 40.0*math.Sin(x/3.0*math.Pi)) * 2.0 / 3.0
	ret += (150.0*math.Sin(x/12.0*math.Pi) + 300.0*math.Sin(x/30.0*math.Pi)) * 2.0 / 3.0
	return ret
}

func outOfChina(lng, lat float64) bool {
	return lng < 72.004 || lng > 137.8347 || lat < 0.8293 || lat > 55.8271
}

func WGS84ToGCJ02(wgsLat, wgsLng float64) (gcjLat, gcjLng float64) {
	if outOfChina(wgsLng, wgsLat) {
		return wgsLat, wgsLng
	}
	dLat := transformLat(wgsLng-105.0, wgsLat-35.0)
	dLng := transformLng(wgsLng-105.0, wgsLat-35.0)
	radLat := wgsLat / 180.0 * math.Pi
	magic := math.Sin(radLat)
	magic = 1 - 0.00669342162296594323*magic*magic
	sqrtMagic := math.Sqrt(magic)
	dLat = (dLat * 180.0) / ((6378245.0 * (1 - 0.00669342162296594323)) / (magic * sqrtMagic) * math.Pi)
	dLng = (dLng * 180.0) / (6378245.0 / sqrtMagic * math.Cos(radLat) * math.Pi)
	return wgsLat + dLat, wgsLng + dLng
}
