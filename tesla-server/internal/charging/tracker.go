package charging

import (
	"fmt"
	"log"
	"tesla-server/internal/ai"
	"tesla-server/internal/database"
	"tesla-server/internal/fleet"
	"tesla-server/internal/geo"
	"tesla-server/internal/redis"
	"tesla-server/models"
	"time"
)

const chargingKeyPrefix = "tesla:charging:"

type ChargingState struct {
	VIN        string    `json:"vin"`
	StartTime  time.Time `json:"start_time"`
	SocStart   int       `json:"soc_start"`
	MaxPower   float64   `json:"max_power"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	IsCharging bool      `json:"is_charging"`
	ChargeType string    `json:"charge_type"`
}

func ProcessChargingState(vin string, data *fleet.SimpleVehicleData) {
	var state ChargingState
	key := chargingKeyPrefix + vin

	err := redis.Get(key, &state)
	isCharging := data.ChargingState == "Charging"

	if err != nil {
		if isCharging {
			startCharging(vin, data)
		}
		return
	}

	if state.IsCharging && !isCharging {
		endCharging(vin, data)
	} else if !state.IsCharging && isCharging {
		startCharging(vin, data)
	} else if isCharging && data.ChargePower > state.MaxPower {
		state.MaxPower = data.ChargePower
		redis.Set(key, state, 24*time.Hour)
	}
}

func startCharging(vin string, data *fleet.SimpleVehicleData) {
	chargeType := "AC"
	if data.Supercharging || data.ChargePower > 50 {
		chargeType = "DC"
	}

	state := ChargingState{
		VIN:        vin,
		StartTime:  time.Now(),
		SocStart:   data.Soc,
		MaxPower:   data.ChargePower,
		Latitude:   data.Latitude,
		Longitude:  data.Longitude,
		IsCharging: true,
		ChargeType: chargeType,
	}

	key := chargingKeyPrefix + vin
	redis.Set(key, state, 24*time.Hour)
}

func endCharging(vin string, data *fleet.SimpleVehicleData) {
	key := chargingKeyPrefix + vin

	var state ChargingState
	if err := redis.Get(key, &state); err != nil {
		return
	}

	now := time.Now()
	socAdded := data.Soc - state.SocStart

	if socAdded < 1 {
		redis.Delete(key)
		return
	}

	chargeKwh := estimateChargeKwh(socAdded, vin)

	// 充电结束时调用一次逆地理编码，获取详细信息
	geoResult := geo.ReverseGeocodeDetail(state.Latitude, state.Longitude)

	location := fmt.Sprintf("%.6f,%.6f", state.Latitude, state.Longitude)

	charge := models.ChargingLog{
		VIN:        vin,
		StartTime:  state.StartTime,
		EndTime:    &now,
		SocStart:   state.SocStart,
		SocEnd:     data.Soc,
		ChargeKwh:  chargeKwh,
		MaxPower:   state.MaxPower,
		ChargeType: state.ChargeType,
		Location:   location,
		Address:    geoResult.Address,
		City:       geoResult.City,
		District:   geoResult.District,
		PoiName:    geoResult.PoiName,
		Latitude:   state.Latitude,
		Longitude:  state.Longitude,
	}

	database.DB.Create(&charge)
	redis.Delete(key)

	go func() {
		var v models.TeslaVehicle
		if err := database.DB.Where("vin = ? AND bind_status = 1", vin).First(&v).Error; err == nil {
			refID := fmt.Sprintf("charging:%d", charge.ID)
			go ai.RunChargingAnalysis(vin, v.UserID, refID)
			log.Printf("[ChargingEngine] Auto AI analysis triggered for charge %d", charge.ID)
		}
	}()
}

func estimateChargeKwh(socAdded int, vin string) float64 {
	batteryCapacity := getBatteryCapacity(vin)
	return batteryCapacity * float64(socAdded) / 100.0
}

func getBatteryCapacity(vin string) float64 {
	if len(vin) < 4 {
		return 60.0
	}
	capacities := map[string]float64{
		"LRW": 60.0,
		"5YJ": 75.0,
		"7SA": 78.0,
		"XP7": 100.0,
	}
	prefix := vin[0:3]
	if cap, ok := capacities[prefix]; ok {
		return cap
	}
	return 60.0
}

func GetChargingLogs(vin string, limit int, startDate, endDate time.Time) ([]models.ChargingLog, error) {
	var logs []models.ChargingLog
	q := database.DB.Where("vin = ?", vin)
	if !startDate.IsZero() && !endDate.IsZero() {
		q = q.Where("start_time >= ? AND start_time < ?", startDate, endDate)
	}
	err := q.Order("start_time DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

func GetChargingStats(vin string, startDate, endDate time.Time) (map[string]interface{}, error) {
	var logs []models.ChargingLog
	err := database.DB.Where("vin = ? AND start_time >= ? AND start_time <= ?", vin, startDate, endDate).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}

	var totalKwh, maxPower float64
	var chargeCount int
	var totalDuration time.Duration
	acCount, dcCount := 0, 0
	// 按城市统计充电次数
	cityCount := make(map[string]int)
	// 家充、超充、第三方统计
	homeCount, superCount, thirdCount := 0, 0, 0

	for _, log := range logs {
		totalKwh += log.ChargeKwh
		chargeCount++
		if log.MaxPower > maxPower {
			maxPower = log.MaxPower
		}
		if log.EndTime != nil {
			totalDuration += log.EndTime.Sub(log.StartTime)
		}
		if log.ChargeType == "AC" {
			acCount++
		} else {
			dcCount++
		}

		// 城市统计
		if log.City != "" {
			cityCount[log.City]++
		}

		// 充电地点类型识别
		if log.PoiName != "" {
			if contains(log.PoiName, "特斯拉") || contains(log.PoiName, "Tesla") || contains(log.PoiName, "超充") {
				superCount++
			} else if contains(log.PoiName, "家") || contains(log.Address, "住宅") {
				homeCount++
			} else {
				thirdCount++
			}
		}
	}

	return map[string]interface{}{
		"total_kwh":      totalKwh,
		"charge_count":   chargeCount,
		"total_duration": totalDuration.Hours(),
		"max_power":      maxPower,
		"ac_count":       acCount,
		"dc_count":       dcCount,
		"city_count":     cityCount,
		"home_count":     homeCount,
		"super_count":    superCount,
		"third_count":    thirdCount,
	}, nil
}

type MonthlyChargingItem struct {
	Month           string   `json:"month"`
	ChargeCount     int      `json:"charge_count"`
	TotalKwh        float64  `json:"total_kwh"`
	MaxPower        float64  `json:"max_power"`
	AvgKwhPerCharge float64  `json:"avg_kwh_per_charge"`
	TotalCost       *float64 `json:"total_cost"` // 总费用(元)，可能为null
}

func GetMonthlyChargingList(vin string) ([]MonthlyChargingItem, error) {
	var logs []models.ChargingLog
	err := database.DB.Where("vin = ?", vin).
		Select("vin, start_time, charge_kwh, max_power, charge_type, total_cost").
		Find(&logs).Error
	if err != nil {
		return nil, err
	}

	monthMap := make(map[string]*MonthlyChargingItem)
	var monthOrder []string

	for _, log := range logs {
		monthKey := log.StartTime.Format("2006-01")
		if _, ok := monthMap[monthKey]; !ok {
			monthMap[monthKey] = &MonthlyChargingItem{Month: monthKey}
			monthOrder = append(monthOrder, monthKey)
		}
		m := monthMap[monthKey]
		m.ChargeCount++
		m.TotalKwh += log.ChargeKwh
		if log.MaxPower > m.MaxPower {
			m.MaxPower = log.MaxPower
		}
		// 累加总费用（只计算有价格的记录）
		if log.TotalCost != nil {
			if m.TotalCost == nil {
				zero := 0.0
				m.TotalCost = &zero
			}
			*m.TotalCost += *log.TotalCost
		}
	}

	for _, m := range monthMap {
		if m.ChargeCount > 0 {
			m.AvgKwhPerCharge = m.TotalKwh / float64(m.ChargeCount)
		}
	}

	result := make([]MonthlyChargingItem, 0, len(monthOrder))
	for i := len(monthOrder) - 1; i >= 0; i-- {
		result = append(result, *monthMap[monthOrder[i]])
	}
	return result, nil
}

func GetMonthlyChargingStats(vin string) (map[string]interface{}, error) {
	now := time.Now()
	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastMonthStart := thisMonthStart.AddDate(0, -1, 0)

	thisItem, err := calcMonthChargingStats(vin, "this_month", thisMonthStart, now)
	if err != nil {
		return nil, err
	}

	lastItem, err := calcMonthChargingStats(vin, "last_month", lastMonthStart, thisMonthStart)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"this_month": thisItem,
		"last_month": lastItem,
	}, nil
}

func calcMonthChargingStats(vin string, monthLabel string, startDate, endDate time.Time) (*MonthlyChargingItem, error) {
	var logs []models.ChargingLog
	err := database.DB.Where("vin = ? AND start_time >= ? AND start_time < ?", vin, startDate, endDate).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}

	var totalKwh, maxPower float64
	chargeCount := len(logs)

	for _, log := range logs {
		totalKwh += log.ChargeKwh
		if log.MaxPower > maxPower {
			maxPower = log.MaxPower
		}
	}

	avgKwhPerCharge := 0.0
	if chargeCount > 0 {
		avgKwhPerCharge = totalKwh / float64(chargeCount)
	}

	return &MonthlyChargingItem{
		Month:        monthLabel,
		ChargeCount:  chargeCount,
		TotalKwh:     totalKwh,
		MaxPower:     maxPower,
		AvgKwhPerCharge: avgKwhPerCharge,
	}, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func GetCurrentCharging(vin string) (*ChargingState, error) {
	key := chargingKeyPrefix + vin
	var state ChargingState
	err := redis.Get(key, &state)
	if err != nil {
		return nil, fmt.Errorf("no active charging session")
	}
	return &state, nil
}
