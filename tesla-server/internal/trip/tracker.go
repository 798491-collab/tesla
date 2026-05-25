package trip

import (
	"fmt"
	"log"
	"math"
	"tesla-server/internal/ai"
	"tesla-server/internal/database"
	"tesla-server/internal/fleet"
	"tesla-server/internal/geo"
	"tesla-server/internal/redis"
	"tesla-server/models"
	"time"
)

const tripKeyPrefix = "tesla:trip:"
const tripLastPointKeyPrefix = "tesla:trip:last_point:"
const tripPendingKeyPrefix = "tesla:trip:pending:"
const tripStopKeyPrefix = "tesla:trip:stop:"

const (
	TripStartConfirmDuration = 10 * time.Second
	TripEndConfirmDuration   = 300 * time.Second
)

type TripPhase string

const (
	PhaseIdle       TripPhase = "idle"
	PhasePending    TripPhase = "pending_start"
	PhaseDriving    TripPhase = "driving"
	PhasePendingEnd TripPhase = "pending_end"
)

type TripState struct {
	VIN           string    `json:"vin"`
	TripID        uint64    `json:"trip_id"`
	Phase         TripPhase `json:"phase"`
	StartTime     time.Time `json:"start_time"`
	StartOdometer float64   `json:"start_odometer"`
	StartLat      float64   `json:"start_lat"`
	StartLng      float64   `json:"start_lng"`
	StartSOC      int       `json:"start_soc"`
	StartAddress  string    `json:"start_address"`
	StartCity     string    `json:"start_city"`
	IsDriving     bool      `json:"is_driving"`
	MaxSpeed      float64   `json:"max_speed"`
	DriveDuration int       `json:"drive_duration"`
	IdleDuration  int       `json:"idle_duration"`
	StoppedSince  time.Time `json:"stopped_since"`
	DrivingSince  time.Time `json:"driving_since"`
}

type LastTripPoint struct {
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Speed      float64   `json:"speed"`
	Heading    int       `json:"heading"`
	RecordedAt time.Time `json:"recorded_at"`
}

func ProcessDrivingState(vin string, data *fleet.SimpleVehicleData) {
	var state TripState
	key := tripKeyPrefix + vin

	isDriving := data.Gear == "D" || data.Gear == "R" || data.Gear == "N"
	isStopped := data.Gear == "P" || (data.Gear == "" && data.Speed == 0)
	isAsleep := data.State == "asleep" || data.State == "offline"

	err := redis.Get(key, &state)

	if err != nil {
		if isDriving {
			state = TripState{
				VIN:          vin,
				Phase:        PhasePending,
				IsDriving:    false,
				DrivingSince: time.Now(),
			}
			redis.Set(key, state, 24*time.Hour)
			log.Printf("[TripEngine] %s: pending_start (shift=%s)", vin, data.Gear)
		}
		return
	}

	switch state.Phase {
	case PhasePending:
		if isDriving {
			elapsed := time.Since(state.DrivingSince)
			if elapsed >= TripStartConfirmDuration {
				state.Phase = PhaseDriving
				confirmStartTrip(vin, data, &state)
				log.Printf("[TripEngine] %s: trip confirmed after %v", vin, elapsed)
			}
		} else {
			state.Phase = PhaseIdle
			state.IsDriving = false
			redis.Set(key, state, 24*time.Hour)
			log.Printf("[TripEngine] %s: pending_start cancelled (not driving)", vin)
		}

	case PhaseDriving:
		if isDriving {
			state.IsDriving = true
			updateTripStats(vin, data, &state)
			saveTripPointSmart(state.TripID, vin, data)
			state.StoppedSince = time.Time{}
			redis.Set(key, state, 24*time.Hour)
		} else if isStopped {
			if state.StoppedSince.IsZero() {
				state.StoppedSince = time.Now()
				state.IsDriving = true
				redis.Set(key, state, 24*time.Hour)
				log.Printf("[TripEngine] %s: pending_end (stopped)", vin)
			} else {
				elapsed := time.Since(state.StoppedSince)
				if elapsed >= TripEndConfirmDuration {
					endTrip(vin, data)
					log.Printf("[TripEngine] %s: trip ended after %v stopped", vin, elapsed)
					return
				}
			}
		}
		if isAsleep {
			endTrip(vin, data)
			log.Printf("[TripEngine] %s: trip ended (vehicle asleep)", vin)
			return
		}

	case PhasePendingEnd:
		if isDriving {
			state.Phase = PhaseDriving
			state.StoppedSince = time.Time{}
			state.IsDriving = true
			redis.Set(key, state, 24*time.Hour)
			log.Printf("[TripEngine] %s: trip resumed", vin)
		}

	case PhaseIdle:
		if isDriving {
			state.Phase = PhasePending
			state.DrivingSince = time.Now()
			redis.Set(key, state, 24*time.Hour)
			log.Printf("[TripEngine] %s: pending_start", vin)
		}
	}
}

func confirmStartTrip(vin string, data *fleet.SimpleVehicleData, state *TripState) {
	startGeo := geo.ReverseGeocodeDetail(data.Latitude, data.Longitude)

	trip := models.TripLog{
		VIN:               vin,
		StartTime:         time.Now(),
		StartOdometer:     data.OdometerKm,
		StartLat:          data.Latitude,
		StartLng:          data.Longitude,
		StartAddress:      startGeo.Address,
		StartCity:         startGeo.City,
		StartBatteryLevel: data.Soc,
	}
	database.DB.Create(&trip)

	state.TripID = trip.ID
	state.StartTime = time.Now()
	state.StartOdometer = data.OdometerKm
	state.StartLat = data.Latitude
	state.StartLng = data.Longitude
	state.StartSOC = data.Soc
	state.StartAddress = startGeo.Address
	state.StartCity = startGeo.City
	state.IsDriving = true
	state.MaxSpeed = data.Speed
	state.DriveDuration = 0
	state.IdleDuration = 0

	key := tripKeyPrefix + vin
	redis.Set(key, state, 24*time.Hour)

	saveTripPoint(trip.ID, vin, data)
}

func endTrip(vin string, data *fleet.SimpleVehicleData) {
	key := tripKeyPrefix + vin

	var state TripState
	if err := redis.Get(key, &state); err != nil {
		return
	}

	if state.TripID == 0 {
		redis.Delete(key)
		redis.Delete(tripLastPointKeyPrefix + vin)
		return
	}

	now := time.Now()
	distance := data.OdometerKm - state.StartOdometer
	duration := now.Sub(state.StartTime).Seconds()

	if distance < 0.1 {
		database.DB.Delete(&models.TripLog{}, state.TripID)
		database.DB.Where("trip_id = ?", state.TripID).Delete(&models.TripPoint{})
		redis.Delete(key)
		redis.Delete(tripLastPointKeyPrefix + vin)
		return
	}

	avgSpeed := 0.0
	if duration > 0 {
		avgSpeed = distance / (duration / 3600.0)
	}

	energyUsed := estimateEnergyUsed(vin, state.StartSOC, data.Soc)

	endGeo := geo.ReverseGeocodeDetail(data.Latitude, data.Longitude)

	avgConsumption := 0.0
	if distance > 0 {
		avgConsumption = (energyUsed / distance) * 100.0
	}

	realDistance := geo.CalculateDistance(state.StartLat, state.StartLng, data.Latitude, data.Longitude)
	if realDistance > distance {
		distance = realDistance
	}

	database.DB.Model(&models.TripLog{}).Where("id = ?", state.TripID).Updates(map[string]interface{}{
		"end_time":          &now,
		"end_odometer":      data.OdometerKm,
		"distance":          distance,
		"avg_speed":         avgSpeed,
		"max_speed":         state.MaxSpeed,
		"energy_used":       energyUsed,
		"end_lat":           data.Latitude,
		"end_lng":           data.Longitude,
		"start_address":     state.StartAddress,
		"end_address":       endGeo.Address,
		"start_city":        state.StartCity,
		"end_city":          endGeo.City,
		"avg_consumption":   avgConsumption,
		"drive_duration":    state.DriveDuration,
		"idle_duration":     state.IdleDuration,
		"end_battery_level": data.Soc,
	})

	redis.Delete(key)
	redis.Delete(tripLastPointKeyPrefix + vin)

	go func() {
		var v models.TeslaVehicle
		if err := database.DB.Where("vin = ? AND bind_status = 1", vin).First(&v).Error; err == nil {
			refID := fmt.Sprintf("trip:%d", state.TripID)
			go ai.RunTripAnalysis(vin, v.UserID, refID)
		}
	}()
}

func updateTripStats(vin string, data *fleet.SimpleVehicleData, state *TripState) {
	key := tripKeyPrefix + vin

	if data.Speed > state.MaxSpeed {
		state.MaxSpeed = data.Speed
	}

	if data.Speed > 5 {
		state.DriveDuration += 5
	} else {
		state.IdleDuration += 5
	}

	redis.Set(key, state, 24*time.Hour)
}

func saveTripPointSmart(tripID uint64, vin string, data *fleet.SimpleVehicleData) {
	if data.Latitude == 0 && data.Longitude == 0 {
		return
	}

	lastKey := tripLastPointKeyPrefix + vin
	var last LastTripPoint

	shouldSave := false

	if err := redis.Get(lastKey, &last); err != nil {
		shouldSave = true
	} else {
		speedDelta := math.Abs(data.Speed - last.Speed)
		if speedDelta > 10 {
			shouldSave = true
		}

		headingDelta := absHeadingDiff(data.Heading, last.Heading)
		if headingDelta > 15 {
			shouldSave = true
		}

		distance := haversineDistance(data.Latitude, data.Longitude, last.Latitude, last.Longitude)
		if distance > 30 {
			shouldSave = true
		}

		if time.Since(last.RecordedAt) > 60*time.Second {
			shouldSave = true
		}
	}

	if shouldSave {
		saveTripPoint(tripID, vin, data)

		last := LastTripPoint{
			Latitude:   data.Latitude,
			Longitude:  data.Longitude,
			Speed:      data.Speed,
			Heading:    data.Heading,
			RecordedAt: time.Now(),
		}
		redis.Set(lastKey, last, 24*time.Hour)
	}
}

func saveTripPoint(tripID uint64, vin string, data *fleet.SimpleVehicleData) {
	if data.Latitude == 0 && data.Longitude == 0 {
		return
	}

	point := models.TripPoint{
		TripID:       tripID,
		VIN:          vin,
		Latitude:     data.Latitude,
		Longitude:    data.Longitude,
		Speed:        data.Speed,
		Heading:      data.Heading,
		BatteryLevel: data.Soc,
		RecordedAt:   time.Now(),
	}
	database.DB.Create(&point)
}

func estimateEnergyUsed(vin string, startSOC, endSOC int) float64 {
	if startSOC <= endSOC {
		return 0
	}
	socDelta := float64(startSOC - endSOC)
	batteryCapacity := getBatteryCapacity(vin)
	return batteryCapacity * socDelta / 100.0
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

func GetTripLogs(vin string, limit int, startDate, endDate time.Time) ([]models.TripLog, error) {
	var trips []models.TripLog
	q := database.DB.Where("vin = ?", vin)
	if !startDate.IsZero() && !endDate.IsZero() {
		q = q.Where("start_time >= ? AND start_time < ?", startDate, endDate)
	}
	err := q.Order("start_time DESC").Limit(limit).Find(&trips).Error
	return trips, err
}

func GetTripPoints(tripID uint64) ([]models.TripPoint, error) {
	var points []models.TripPoint
	err := database.DB.Where("trip_id = ?", tripID).
		Order("recorded_at ASC").
		Find(&points).Error
	return points, err
}

func GetTripStats(vin string, startDate, endDate time.Time) (map[string]interface{}, error) {
	var trips []models.TripLog
	err := database.DB.Where("vin = ? AND start_time >= ? AND start_time <= ?", vin, startDate, endDate).
		Find(&trips).Error
	if err != nil {
		return nil, err
	}

	var totalDistance, totalEnergy float64
	var tripCount int
	var totalDuration time.Duration

	for _, trip := range trips {
		totalDistance += trip.Distance
		totalEnergy += trip.EnergyUsed
		tripCount++
		if trip.EndTime != nil {
			totalDuration += trip.EndTime.Sub(trip.StartTime)
		}
	}

	avgSpeed := 0.0
	if totalDuration.Hours() > 0 {
		avgSpeed = totalDistance / totalDuration.Hours()
	}

	avgConsumption := 0.0
	if totalDistance > 0 {
		avgConsumption = (totalEnergy / totalDistance) * 100.0
	}

	return map[string]interface{}{
		"total_distance":  totalDistance,
		"total_energy":    totalEnergy,
		"trip_count":      tripCount,
		"total_duration":  totalDuration.Hours(),
		"avg_speed":       avgSpeed,
		"avg_consumption": avgConsumption,
	}, nil
}

type MonthlyStatsItem struct {
	Month          string  `json:"month"`
	TripCount      int     `json:"trip_count"`
	TotalDistance   float64 `json:"total_distance"`
	TotalEnergy    float64 `json:"total_energy"`
	AvgConsumption float64 `json:"avg_consumption"`
}

func GetMonthlyTripList(vin string) ([]MonthlyStatsItem, error) {
	var trips []models.TripLog
	err := database.DB.Where("vin = ?", vin).
		Select("vin, start_time, distance, energy_used").
		Find(&trips).Error
	if err != nil {
		return nil, err
	}

	monthMap := make(map[string]*MonthlyStatsItem)
	var monthOrder []string

	for _, trip := range trips {
		monthKey := trip.StartTime.Format("2006-01")
		if _, ok := monthMap[monthKey]; !ok {
			monthMap[monthKey] = &MonthlyStatsItem{Month: monthKey}
			monthOrder = append(monthOrder, monthKey)
		}
		m := monthMap[monthKey]
		m.TripCount++
		m.TotalDistance += trip.Distance
		m.TotalEnergy += trip.EnergyUsed
	}

	for _, m := range monthMap {
		if m.TotalDistance > 0 {
			m.AvgConsumption = (m.TotalEnergy / m.TotalDistance) * 100.0
		}
	}

	result := make([]MonthlyStatsItem, 0, len(monthOrder))
	for i := len(monthOrder) - 1; i >= 0; i-- {
		result = append(result, *monthMap[monthOrder[i]])
	}
	return result, nil
}

func GetMonthlyStats(vin string) (map[string]interface{}, error) {
	now := time.Now()

	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastMonthStart := thisMonthStart.AddDate(0, -1, 0)

	thisMonthItem, err := calcMonthStats(vin, "this_month", thisMonthStart, now)
	if err != nil {
		return nil, err
	}

	lastMonthItem, err := calcMonthStats(vin, "last_month", lastMonthStart, thisMonthStart)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"this_month": thisMonthItem,
		"last_month": lastMonthItem,
	}, nil
}

func calcMonthStats(vin string, monthLabel string, startDate, endDate time.Time) (*MonthlyStatsItem, error) {
	var trips []models.TripLog
	err := database.DB.Where("vin = ? AND start_time >= ? AND start_time < ?", vin, startDate, endDate).
		Find(&trips).Error
	if err != nil {
		return nil, err
	}

	var totalDistance, totalEnergy float64
	tripCount := len(trips)

	for _, trip := range trips {
		totalDistance += trip.Distance
		totalEnergy += trip.EnergyUsed
	}

	avgConsumption := 0.0
	if totalDistance > 0 {
		avgConsumption = (totalEnergy / totalDistance) * 100.0
	}

	return &MonthlyStatsItem{
		Month:          monthLabel,
		TripCount:      tripCount,
		TotalDistance:   totalDistance,
		TotalEnergy:    totalEnergy,
		AvgConsumption: avgConsumption,
	}, nil
}

func GetCurrentTrip(vin string) (*TripState, error) {
	key := tripKeyPrefix + vin
	var state TripState
	err := redis.Get(key, &state)
	if err != nil {
		return nil, fmt.Errorf("no active trip")
	}
	return &state, nil
}

func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371000
	phi1 := lat1 * math.Pi / 180
	phi2 := lat2 * math.Pi / 180
	deltaPhi := (lat2 - lat1) * math.Pi / 180
	deltaLambda := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(deltaPhi/2)*math.Sin(deltaPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*
			math.Sin(deltaLambda/2)*math.Sin(deltaLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func absHeadingDiff(a, b int) int {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	if diff > 180 {
		diff = 360 - diff
	}
	return diff
}
