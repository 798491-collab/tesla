package ai

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"tesla-server/config"
	"tesla-server/internal/database"
	"tesla-server/internal/middleware"
	"tesla-server/internal/redis"
	"tesla-server/internal/ws"
	"tesla-server/models"
	"time"

	"github.com/gin-gonic/gin"
)

func GetTripAnalysis(c *gin.Context) {
	vin := c.Param("vin")
	refID := c.Param("refId")

	var analysis models.AIAnalysis
	if err := database.DB.Where("vin = ? AND type = ? AND ref_id = ?", vin, "trip", refID).Order("created_at DESC").First(&analysis).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": formatAnalysis(&analysis)})
}

func GetChargingAnalysis(c *gin.Context) {
	vin := c.Param("vin")
	refID := c.Param("refId")

	var analysis models.AIAnalysis
	if err := database.DB.Where("vin = ? AND type = ? AND ref_id = ?", vin, "charging", refID).Order("created_at DESC").First(&analysis).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": formatAnalysis(&analysis)})
}

func GetVehicleAnalysis(c *gin.Context) {
	vin := c.Param("vin")
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	refID := "vehicle_daily:" + date

	var analysis models.AIAnalysis
	if err := database.DB.Where("vin = ? AND type = ? AND ref_id = ?", vin, "vehicle", refID).Order("created_at DESC").First(&analysis).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": formatAnalysis(&analysis)})
}

func GetAnalysisHistory(c *gin.Context) {
	vin := c.Param("vin")
	userID := middleware.GetUserID(c)

	analysisType := c.DefaultQuery("type", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	var analyses []models.AIAnalysis
	q := database.DB.Where("vin = ? AND user_id = ?", vin, userID)
	if analysisType != "" {
		q = q.Where("type = ?", analysisType)
	}
	q = q.Order("created_at DESC").Limit(limit)

	if err := q.Find(&analyses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": analyses})
}

func GetAnalysisList(c *gin.Context) {
	vin := c.Param("vin")
	userID := middleware.GetUserID(c)

	analysisType := c.DefaultQuery("type", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	var total int64
	q := database.DB.Model(&models.AIAnalysis{}).Where("vin = ? AND user_id = ?", vin, userID)
	if analysisType != "" {
		q = q.Where("type = ?", analysisType)
	}
	q.Count(&total)

	var analyses []models.AIAnalysis
	q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&analyses)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":      analyses,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

func TriggerTripAnalysis(c *gin.Context) {
	vin := c.Param("vin")
	userID := middleware.GetUserID(c)
	refID := c.Param("refId")

	var existing models.AIAnalysis
	if err := database.DB.Where("vin = ? AND type = ? AND ref_id = ?", vin, "trip", refID).Order("created_at DESC").First(&existing).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": formatAnalysis(&existing)})
		return
	}

	go RunTripAnalysis(vin, userID, refID)

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": nil, "message": "分析已启动"})
}

func TriggerChargingAnalysis(c *gin.Context) {
	vin := c.Param("vin")
	userID := middleware.GetUserID(c)
	refID := c.Param("refId")

	var existing models.AIAnalysis
	if err := database.DB.Where("vin = ? AND type = ? AND ref_id = ?", vin, "charging", refID).Order("created_at DESC").First(&existing).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": formatAnalysis(&existing)})
		return
	}

	go RunChargingAnalysis(vin, userID, refID)

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": nil, "message": "分析已启动"})
}

func TriggerVehicleAnalysis(c *gin.Context) {
	vin := c.Param("vin")
	userID := middleware.GetUserID(c)
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	refID := "vehicle_daily:" + date

	var existing models.AIAnalysis
	if err := database.DB.Where("vin = ? AND type = ? AND ref_id = ?", vin, "vehicle", refID).Order("created_at DESC").First(&existing).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": formatAnalysis(&existing)})
		return
	}

	go RunVehicleAnalysis(vin, userID, date)

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": nil, "message": "分析已启动"})
}

func RunTripAnalysis(vin string, userID uint64, refID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[AI] RunTripAnalysis panic: %v", r)
		}
	}()

	var existing models.AIAnalysis
	if err := database.DB.Where("vin = ? AND type = ? AND ref_id = ?", vin, "trip", refID).Order("created_at DESC").First(&existing).Error; err == nil {
		log.Printf("[AI] Trip analysis already exists for %s/%s", vin, refID)
		return
	}

	systemPrompt := `你是一位专业的Tesla车辆行驶数据分析专家，同时具备行业级数据对比分析能力。请根据用户提供的行驶数据，从以下维度进行专业分析：

1. 行驶风格评估：判断行驶风格（平稳型/激进型/温和型），并与Tesla同车型车主的平均驾驶风格进行对比
2. 驾驶习惯分析：急加速、急刹车、怠速等关键习惯，与行业平均水平对比
3. 出行规律洞察：出行时段、路线偏好等，与同城市Tesla车主的出行特征对比
4. 能耗效率评估：百公里能耗与同型号Tesla的官方标称及车主平均数据进行对比
5. 优化建议：基于以上多维度对比，给出个性化、可操作的优化建议

要求：
- 分析结果使用中文，语言简洁易懂
- 在每个维度中尽可能提供与"同品牌(Tesla)车主、同型号车主、同城市车主、同行业(新能源出行)平均水平"的对比参考
- 对比数据基于你的专业知识库中的行业数据，若无精确数据请给出合理估算范围并标注
- 使用结构化格式输出：风格评估、关键发现、多维度对比、优化建议
- 数据中如有异常值请忽略，仅基于合理数据进行分析`

	var userPrompt string
	var err error

	if len(refID) > 5 && refID[:5] == "trip:" {
		tripIDStr := refID[5:]
		userPrompt, err = buildSingleTripPrompt(vin, tripIDStr)
	} else if len(refID) > 13 && refID[:13] == "trip_monthly:" {
		month := refID[13:]
		userPrompt, err = buildMonthlyTripPrompt(vin, month)
	} else {
		log.Printf("[AI] Unknown trip refID format: %s", refID)
		return
	}

	if err != nil {
		log.Printf("[AI] Build trip prompt failed: %v", err)
		return
	}

	resp, err := Chat(systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[AI] Trip analysis failed: %v", err)
		return
	}

	if len(resp.Choices) == 0 {
		log.Printf("[AI] Trip analysis empty response")
		return
	}

	result := resp.Choices[0].Message.Content
	summary := generateSummary(result)
	cfg := config.Load()

	analysis := models.AIAnalysis{
		UserID:    userID,
		VIN:       vin,
		Type:      "trip",
		RefID:     refID,
		Prompt:    userPrompt,
		Result:    result,
		Summary:   summary,
		Model:     cfg.AI.Model,
		TokensIn:  resp.Usage.PromptTokens,
		TokensOut: resp.Usage.CompletionTokens,
	}
	database.DB.Create(&analysis)
	log.Printf("[AI] Trip analysis completed: vin=%s refID=%s", vin, refID)
	// 通知前端分析完成
	ws.DefaultHub.BroadcastToVIN(vin, "analysis_complete", map[string]interface{}{
		"type":   "trip",
		"ref_id": refID,
		"status": "completed",
	})
}

func RunChargingAnalysis(vin string, userID uint64, refID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[AI] RunChargingAnalysis panic: %v", r)
		}
	}()

	var existing models.AIAnalysis
	if err := database.DB.Where("vin = ? AND type = ? AND ref_id = ?", vin, "charging", refID).Order("created_at DESC").First(&existing).Error; err == nil {
		log.Printf("[AI] Charging analysis already exists for %s/%s", vin, refID)
		return
	}

	systemPrompt := `你是一位专业的Tesla车辆充电数据分析专家，同时具备行业级数据对比分析能力。请根据用户提供的充电数据，从以下维度进行专业分析：

1. 充电效率评估：充电速度等级判断（高速/中速/低速），与同型号Tesla在不同充电桩类型下的平均充电效率对比
2. 充电习惯分析：充电频率、充电时段、SOC区间偏好等，与同城市Tesla车主的充电习惯对比
3. 充电桩表现：充电功率稳定性、功率衰减等，与同品牌车主在同类型充电桩的体验对比
4. 成本与策略：峰谷时段利用率、浅充/深充比例等，与行业最佳实践对比
5. 优化建议：基于多维度对比给出充电策略优化建议

要求：
- 分析结果使用中文，语言简洁易懂
- 在每个维度中尽可能提供与"同品牌(Tesla)车主、同型号车主、同城市车主、同行业(新能源出行)平均水平"的对比参考
- 对比数据基于你的专业知识库中的行业数据，若无精确数据请给出合理估算范围并标注
- 使用结构化格式输出：效率评估、关键发现、多维度对比、优化建议
- 数据中如有异常值请忽略，仅基于合理数据进行分析`

	var userPrompt string
	var err error

	if len(refID) > 9 && refID[:9] == "charging:" {
		chargeIDStr := refID[9:]
		userPrompt, err = buildSingleChargingPrompt(vin, chargeIDStr)
	} else if len(refID) > 17 && refID[:17] == "charging_monthly:" {
		month := refID[17:]
		userPrompt, err = buildMonthlyChargingPrompt(vin, month)
	} else {
		log.Printf("[AI] Unknown charging refID format: %s", refID)
		return
	}

	if err != nil {
		log.Printf("[AI] Build charging prompt failed: %v", err)
		return
	}

	resp, err := Chat(systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[AI] Charging analysis failed: %v", err)
		return
	}

	if len(resp.Choices) == 0 {
		log.Printf("[AI] Charging analysis empty response")
		return
	}

	result := resp.Choices[0].Message.Content
	summary := generateSummary(result)
	cfg := config.Load()

	analysis := models.AIAnalysis{
		UserID:    userID,
		VIN:       vin,
		Type:      "charging",
		RefID:     refID,
		Prompt:    userPrompt,
		Result:    result,
		Summary:   summary,
		Model:     cfg.AI.Model,
		TokensIn:  resp.Usage.PromptTokens,
		TokensOut: resp.Usage.CompletionTokens,
	}
	database.DB.Create(&analysis)
	log.Printf("[AI] Charging analysis completed: vin=%s refID=%s", vin, refID)
	// 通知前端分析完成
	ws.DefaultHub.BroadcastToVIN(vin, "analysis_complete", map[string]interface{}{
		"type":   "charging",
		"ref_id": refID,
		"status": "completed",
	})
}

func RunVehicleAnalysis(vin string, userID uint64, date string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[AI] RunVehicleAnalysis panic: %v", r)
		}
	}()

	refID := "vehicle_daily:" + date

	var existing models.AIAnalysis
	if err := database.DB.Where("vin = ? AND type = ? AND ref_id = ?", vin, "vehicle", refID).Order("created_at DESC").First(&existing).Error; err == nil {
		log.Printf("[AI] Vehicle analysis already exists for %s/%s", vin, refID)
		return
	}

	systemPrompt := `你是一位专业的Tesla车辆状态分析专家，同时具备行业级数据对比分析能力。请根据用户提供的车辆运行数据，从以下维度进行专业分析：

1. 车辆状态评估：判断车辆当日整体运行状态（正常/轻微异常/需关注），与同型号Tesla的常见运行状态对比
2. 电池健康分析：电量消耗速率、充电效率等，与同型号同车龄Tesla的电池衰减数据对比
3. 胎压与安全：胎压数据与Tesla官方推荐值(2.9-3.1bar)及同型号车主平均胎压对比
4. 能耗与续航：当日能耗与同型号Tesla的官方续航标称及车主实际平均能耗对比
5. 维护建议：基于多维度对比给出个性化维护建议

要求：
- 分析结果使用中文，语言简洁易懂
- 在每个维度中尽可能提供与"同品牌(Tesla)车主、同型号车主、同城市车主、同行业(新能源出行)平均水平"的对比参考
- 对比数据基于你的专业知识库中的行业数据，若无精确数据请给出合理估算范围并标注
- 使用结构化格式输出：状态评估、异常分析、多维度对比、维护建议
- 数据中如有异常值请忽略，仅基于合理数据进行分析
- 胎压正常范围2.9-3.1 bar，低于2.8或高于3.3需关注`

	userPrompt, err := buildVehicleDailyPrompt(vin, date)
	if err != nil {
		log.Printf("[AI] Build vehicle prompt failed: %v", err)
		return
	}

	resp, err := Chat(systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[AI] Vehicle analysis failed: %v", err)
		return
	}

	if len(resp.Choices) == 0 {
		log.Printf("[AI] Vehicle analysis empty response")
		return
	}

	result := resp.Choices[0].Message.Content
	summary := generateSummary(result)
	cfg := config.Load()

	analysis := models.AIAnalysis{
		UserID:    userID,
		VIN:       vin,
		Type:      "vehicle",
		RefID:     refID,
		Prompt:    userPrompt,
		Result:    result,
		Summary:   summary,
		Model:     cfg.AI.Model,
		TokensIn:  resp.Usage.PromptTokens,
		TokensOut: resp.Usage.CompletionTokens,
	}
	database.DB.Create(&analysis)
	log.Printf("[AI] Vehicle analysis completed: vin=%s date=%s", vin, date)
	// 通知前端分析完成
	ws.DefaultHub.BroadcastToVIN(vin, "analysis_complete", map[string]interface{}{
		"type":   "vehicle",
		"ref_id": refID,
		"status": "completed",
	})
}

func formatAnalysis(a *models.AIAnalysis) gin.H {
	return gin.H{
		"id":         a.ID,
		"type":       a.Type,
		"ref_id":     a.RefID,
		"result":     a.Result,
		"summary":    a.Summary,
		"model":      a.Model,
		"tokens_in":  a.TokensIn,
		"tokens_out": a.TokensOut,
		"created_at": a.CreatedAt,
	}
}

func generateSummary(fullResult string) string {
	summaryPrompt := "请用一句话（不超过50个字）概括以下AI分析结果的核心结论，只输出概括文字，不要任何前缀、标点符号以外的格式：\n\n" + fullResult
	resp, err := Chat("你是一个精准的摘要生成器，只输出简洁的一句话概括。", summaryPrompt)
	if err != nil || len(resp.Choices) == 0 {
		if len(fullResult) > 80 {
			return fullResult[:80] + "..."
		}
		return fullResult
	}
	summary := resp.Choices[0].Message.Content
	if len(summary) > 100 {
		summary = summary[:100] + "..."
	}
	return summary
}

func GetLatestAnalysis(c *gin.Context) {
	vin := c.Param("vin")
	analysisType := c.Param("type")

	var analysis models.AIAnalysis
	if err := database.DB.Where("vin = ? AND type = ?", vin, analysisType).Order("created_at DESC").First(&analysis).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": formatAnalysis(&analysis)})
}

func buildSingleTripPrompt(vin, tripIDStr string) (string, error) {
	tripID, err := strconv.ParseUint(tripIDStr, 10, 64)
	if err != nil {
		return "", fmt.Errorf("无效的行程ID")
	}

	var trip models.TripLog
	if err := database.DB.Where("id = ? AND vin = ?", tripID, vin).First(&trip).Error; err != nil {
		return "", fmt.Errorf("行程记录不存在")
	}

	var points []models.TripPoint
	database.DB.Where("trip_id = ?", tripID).Order("recorded_at ASC").Find(&points)

	durationMin := 0.0
	if trip.EndTime != nil {
		durationMin = trip.EndTime.Sub(trip.StartTime).Minutes()
	}

	data := map[string]interface{}{
		"行程ID":       trip.ID,
		"出发时间":      trip.StartTime.Format("2006-01-02 15:04:05"),
		"到达时间":      func() string {
			if trip.EndTime != nil {
				return trip.EndTime.Format("2006-01-02 15:04:05")
			}
			return "进行中"
		}(),
		"行驶时长(分钟)":  fmt.Sprintf("%.1f", durationMin),
		"行驶距离(km)":  fmt.Sprintf("%.1f", trip.Distance),
		"平均速度(km/h)": fmt.Sprintf("%.1f", trip.AvgSpeed),
		"最高速度(km/h)": fmt.Sprintf("%.1f", trip.MaxSpeed),
		"能耗(kWh)":    fmt.Sprintf("%.1f", trip.EnergyUsed),
		"百公里能耗(kWh)": fmt.Sprintf("%.1f", trip.AvgConsumption),
		"出发电量(%)":   trip.StartBatteryLevel,
		"到达电量(%)":   trip.EndBatteryLevel,
		"出发地址":      trip.StartAddress,
		"到达地址":      trip.EndAddress,
		"出发城市":      trip.StartCity,
		"到达城市":      trip.EndCity,
		"行驶时间(秒)":   trip.DriveDuration,
		"怠速时间(秒)":   trip.IdleDuration,
	}

	if len(points) > 0 {
		speeds := make([]float64, 0, len(points))
		for _, p := range points {
			speeds = append(speeds, p.Speed)
		}
		data["轨迹点数量"] = len(points)
		if len(speeds) > 0 {
			maxS := speeds[0]
			accelCount := 0
			decelCount := 0
			for i := 1; i < len(speeds); i++ {
				if speeds[i] > maxS {
					maxS = speeds[i]
				}
				delta := speeds[i] - speeds[i-1]
				if delta > 20 {
					accelCount++
				} else if delta < -20 {
					decelCount++
				}
			}
			data["急加速次数"] = accelCount
			data["急减速次数"] = decelCount
		}
	}

	jsonData, _ := json.MarshalIndent(data, "", "  ")
	return fmt.Sprintf("请分析以下Tesla单次行程数据：\n\n%s", string(jsonData)), nil
}

func buildMonthlyTripPrompt(vin, month string) (string, error) {
	var trips []models.TripLog
	startDate, _ := time.Parse("2006-01", month)
	if startDate.IsZero() {
		return "", fmt.Errorf("无效的月份格式")
	}
	endDate := startDate.AddDate(0, 1, 0)

	if err := database.DB.Where("vin = ? AND start_time >= ? AND start_time < ?", vin, startDate, endDate).
		Find(&trips).Error; err != nil {
		return "", fmt.Errorf("查询行程数据失败")
	}

	if len(trips) == 0 {
		return "", fmt.Errorf("该月暂无行程数据")
	}

	var totalDistance, totalEnergy, avgSpeedSum float64
	var maxSpeed, totalDriveDuration, totalIdleDuration float64
	var tripCount int
	cityMap := make(map[string]int)
	hourMap := make(map[int]int)

	for _, trip := range trips {
		totalDistance += trip.Distance
		totalEnergy += trip.EnergyUsed
		avgSpeedSum += trip.AvgSpeed
		if trip.MaxSpeed > maxSpeed {
			maxSpeed = trip.MaxSpeed
		}
		totalDriveDuration += float64(trip.DriveDuration)
		totalIdleDuration += float64(trip.IdleDuration)
		tripCount++

		if trip.StartCity != "" {
			cityMap[trip.StartCity]++
		}
		hourMap[trip.StartTime.Hour()]++
	}

	avgSpeed := 0.0
	if tripCount > 0 {
		avgSpeed = avgSpeedSum / float64(tripCount)
	}

	avgConsumption := 0.0
	if totalDistance > 0 {
		avgConsumption = (totalEnergy / totalDistance) * 100.0
	}

	data := map[string]interface{}{
		"月份":            month,
		"行程总数":          tripCount,
		"总行驶里程(km)":     fmt.Sprintf("%.1f", totalDistance),
		"总能耗(kWh)":      fmt.Sprintf("%.1f", totalEnergy),
		"平均速度(km/h)":    fmt.Sprintf("%.1f", avgSpeed),
		"最高速度(km/h)":    fmt.Sprintf("%.1f", maxSpeed),
		"百公里平均能耗(kWh)":  fmt.Sprintf("%.1f", avgConsumption),
		"总行驶时间(秒)":      fmt.Sprintf("%.0f", totalDriveDuration),
		"总怠速时间(秒)":      fmt.Sprintf("%.0f", totalIdleDuration),
		"怠速占比(%)":       fmt.Sprintf("%.1f", totalIdleDuration/(totalDriveDuration+totalIdleDuration+1)*100),
		"高频出发城市":        cityMap,
		"出发时段分布(小时)":    hourMap,
	}

	jsonData, _ := json.MarshalIndent(data, "", "  ")
	return fmt.Sprintf("请分析以下Tesla月度行程汇总数据：\n\n%s", string(jsonData)), nil
}

func buildSingleChargingPrompt(vin, chargeIDStr string) (string, error) {
	chargeID, err := strconv.ParseUint(chargeIDStr, 10, 64)
	if err != nil {
		return "", fmt.Errorf("无效的充电记录ID")
	}

	var charge models.ChargingLog
	if err := database.DB.Where("id = ? AND vin = ?", chargeID, vin).First(&charge).Error; err != nil {
		return "", fmt.Errorf("充电记录不存在")
	}

	durationMin := 0.0
	if charge.EndTime != nil {
		durationMin = charge.EndTime.Sub(charge.StartTime).Minutes()
	}

	chargingSpeed := "低速"
	if charge.MaxPower > 100 {
		chargingSpeed = "高速(超充)"
	} else if charge.MaxPower > 30 {
		chargingSpeed = "中速(快充)"
	}

	data := map[string]interface{}{
		"充电ID":    charge.ID,
		"充电开始时间": charge.StartTime.Format("2006-01-02 15:04:05"),
		"充电结束时间": func() string {
			if charge.EndTime != nil {
				return charge.EndTime.Format("2006-01-02 15:04:05")
			}
			return "进行中"
		}(),
		"充电时长(分钟)":  fmt.Sprintf("%.1f", durationMin),
		"开始SOC(%)":  charge.SocStart,
		"结束SOC(%)":  charge.SocEnd,
		"SOC增量(%)":  charge.SocEnd - charge.SocStart,
		"充电电量(kWh)": fmt.Sprintf("%.2f", charge.ChargeKwh),
		"最大功率(kW)":  fmt.Sprintf("%.1f", charge.MaxPower),
		"平均功率(kW)":  fmt.Sprintf("%.1f", charge.AveragePowerKw),
		"充电类型": func() string {
			if charge.IsDcFastCharge {
				return "DC快充"
			}
			if charge.ChargeType == "DC" {
				return "DC快充"
			}
			return "AC慢充"
		}(),
		"充电速度等级": chargingSpeed,
		"充电地点":    charge.Address,
		"充电城市":    charge.City,
		"POI名称":   charge.PoiName,
		"车外温度(℃)": charge.OutsideTemp,
		"电池温度(℃)": charge.BatteryTemp,
	}

	jsonData, _ := json.MarshalIndent(data, "", "  ")
	return fmt.Sprintf("请分析以下Tesla单次充电数据：\n\n%s", string(jsonData)), nil
}

func buildMonthlyChargingPrompt(vin, month string) (string, error) {
	var logs []models.ChargingLog
	startDate, _ := time.Parse("2006-01", month)
	if startDate.IsZero() {
		return "", fmt.Errorf("无效的月份格式")
	}
	endDate := startDate.AddDate(0, 1, 0)

	if err := database.DB.Where("vin = ? AND start_time >= ? AND start_time < ?", vin, startDate, endDate).
		Find(&logs).Error; err != nil {
		return "", fmt.Errorf("查询充电数据失败")
	}

	if len(logs) == 0 {
		return "", fmt.Errorf("该月暂无充电数据")
	}

	var totalKwh, maxPower, avgPowerSum float64
	var acCount, dcCount int
	cityMap := make(map[string]int)
	hourMap := make(map[int]int)
	poiMap := make(map[string]int)

	for _, l := range logs {
		totalKwh += l.ChargeKwh
		if l.MaxPower > maxPower {
			maxPower = l.MaxPower
		}
		avgPowerSum += l.AveragePowerKw
		if l.ChargeType == "DC" || l.IsDcFastCharge {
			dcCount++
		} else {
			acCount++
		}
		if l.City != "" {
			cityMap[l.City]++
		}
		hourMap[l.StartTime.Hour()]++
		if l.PoiName != "" {
			poiMap[l.PoiName]++
		}
	}

	chargeCount := len(logs)
	avgPower := 0.0
	if chargeCount > 0 {
		avgPower = avgPowerSum / float64(chargeCount)
	}

	data := map[string]interface{}{
		"月份":          month,
		"充电总次数":       chargeCount,
		"DC快充次数":      dcCount,
		"AC慢充次数":      acCount,
		"总充电电量(kWh)":  fmt.Sprintf("%.1f", totalKwh),
		"次均充电电量(kWh)": fmt.Sprintf("%.1f", totalKwh/float64(chargeCount)),
		"最大功率(kW)":    fmt.Sprintf("%.1f", maxPower),
		"平均功率(kW)":    fmt.Sprintf("%.1f", avgPower),
		"高频充电城市":      cityMap,
		"高频充电时段(小时)":  hourMap,
		"常用充电桩(POI)":  poiMap,
	}

	jsonData, _ := json.MarshalIndent(data, "", "  ")
	return fmt.Sprintf("请分析以下Tesla月度充电汇总数据：\n\n%s", string(jsonData)), nil
}

func buildVehicleDailyPrompt(vin, date string) (string, error) {
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", fmt.Errorf("无效的日期格式")
	}
	nextDay := parsedDate.AddDate(0, 0, 1)

	type VehicleStateCache struct {
		BatteryLevel   float64 `json:"battery_level"`
		ChargingState  string  `json:"charging_state"`
		ChargerPower   float64 `json:"charger_power"`
		Speed          float64 `json:"speed"`
		InsideTemp     float64 `json:"inside_temp"`
		OutsideTemp    float64 `json:"outside_temp"`
		TirePressureFL float64 `json:"tire_pressure_fl"`
		TirePressureFR float64 `json:"tire_pressure_fr"`
		TirePressureRL float64 `json:"tire_pressure_rl"`
		TirePressureRR float64 `json:"tire_pressure_rr"`
		OdometerKm     float64 `json:"odometer_km"`
		Locked         bool    `json:"locked"`
		SentryMode     bool    `json:"sentry_mode"`
	}

	var currentState VehicleStateCache
	stateKey := "tesla:vehicle:" + vin + ":state"
	hasState := false
	if err := redis.Get(stateKey, &currentState); err == nil {
		hasState = true
	}

	var telemetries []models.VehicleTelemetry
	database.DB.Where("vin = ? AND recorded_at >= ? AND recorded_at < ?", vin, parsedDate, nextDay).
		Order("recorded_at ASC").
		Find(&telemetries)

	var trips []models.TripLog
	database.DB.Where("vin = ? AND start_time >= ? AND start_time < ?", vin, parsedDate, nextDay).
		Find(&trips)

	var charges []models.ChargingLog
	database.DB.Where("vin = ? AND start_time >= ? AND start_time < ?", vin, parsedDate, nextDay).
		Find(&charges)

	data := map[string]interface{}{
		"日期":     date,
		"VIN":    vin,
		"当日行程数":  len(trips),
		"当日充电次数": len(charges),
	}

	if hasState {
		data["当前电量(%)"] = currentState.BatteryLevel
		data["充电状态"] = currentState.ChargingState
		data["充电功率(kW)"] = currentState.ChargerPower
		data["车内温度(℃)"] = currentState.InsideTemp
		data["车外温度(℃)"] = currentState.OutsideTemp
		data["左前胎压(bar)"] = currentState.TirePressureFL
		data["右前胎压(bar)"] = currentState.TirePressureFR
		data["左后胎压(bar)"] = currentState.TirePressureRL
		data["右后胎压(bar)"] = currentState.TirePressureRR
		data["总里程(km)"] = currentState.OdometerKm
		data["车锁状态"] = func() string {
			if currentState.Locked {
				return "已锁"
			}
			return "未锁"
		}()
		data["哨兵模式"] = func() string {
			if currentState.SentryMode {
				return "开启"
			}
			return "关闭"
		}()
	}

	if len(trips) > 0 {
		var totalDist, totalEnergy float64
		for _, t := range trips {
			totalDist += t.Distance
			totalEnergy += t.EnergyUsed
		}
		data["当日总行驶里程(km)"] = fmt.Sprintf("%.1f", totalDist)
		data["当日总能耗(kWh)"] = fmt.Sprintf("%.1f", totalEnergy)
	}

	if len(charges) > 0 {
		var totalKwh float64
		for _, c := range charges {
			totalKwh += c.ChargeKwh
		}
		data["当日总充电量(kWh)"] = fmt.Sprintf("%.1f", totalKwh)
	}

	if len(telemetries) > 0 {
		data["遥测记录数"] = len(telemetries)
		maxSpeed := 0.0
		minBattery := 100
		maxBattery := 0
		for _, t := range telemetries {
			if t.Speed > maxSpeed {
				maxSpeed = t.Speed
			}
			if t.BatteryLevel < minBattery {
				minBattery = t.BatteryLevel
			}
			if t.BatteryLevel > maxBattery {
				maxBattery = t.BatteryLevel
			}
		}
		data["当日最高速度(km/h)"] = fmt.Sprintf("%.1f", maxSpeed)
		data["当日最低电量(%)"] = minBattery
		data["当日最高电量(%)"] = maxBattery
	}

	jsonData, _ := json.MarshalIndent(data, "", "  ")
	return fmt.Sprintf("请分析以下Tesla车辆当日运行数据：\n\n%s", string(jsonData)), nil
}
