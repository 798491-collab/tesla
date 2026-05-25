package tesla

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"tesla-server/config"
	"tesla-server/internal/database"
	"tesla-server/internal/fleet"
	"tesla-server/internal/middleware"
	"tesla-server/internal/redis"
	"tesla-server/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token"` // JWT 格式的 id_token，包含 sub
}

// TeslaJWTClaims Tesla access_token 的 JWT Claims
// Tesla 中国区的 token 端点不返回 scope 字段，需要从 JWT 中解析
type TeslaJWTClaims struct {
	Sub       string   `json:"sub"`        // Tesla 用户唯一 ID
	SCP       []string `json:"scp"`        // 授权 scope 列表
	AccountID string   `json:"account_id"` // Tesla 账户 ID
	Exp       int64    `json:"exp"`        // 过期时间
}

// ParseTeslaJWT 解析 Tesla JWT token (access_token)
// 直接从 access_token 中获取 sub 和 scope，不需要调用 userinfo
func ParseTeslaJWT(token string) (*TeslaJWTClaims, error) {
	if token == "" {
		return nil, fmt.Errorf("token is empty")
	}

	// JWT 格式: header.payload.signature
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid jwt token")
	}

	payload := parts[1]

	// JWT padding
	if m := len(payload) % 4; m != 0 {
		payload += strings.Repeat("=", 4-m)
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %v", err)
	}

	var claims TeslaJWTClaims
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse JWT claims: %v", err)
	}

	return &claims, nil
}

// generateAuthID 生成唯一的 auth_id
func generateAuthID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// GetAuthData 通过 auth_id 获取 OAuth 数据
func GetAuthData(c *gin.Context) {
	authID := c.Query("auth_id")
	if authID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "auth_id is required"})
		return
	}

	redisKey := fmt.Sprintf("tesla:oauth:%s", authID)
	var authDataJSON []byte

	if err := redis.Get(redisKey, &authDataJSON); err != nil {
		log.Printf("[Tesla OAuth] Failed to get auth data from redis: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Auth data not found or expired"})
		return
	}

	var authData map[string]interface{}
	if err := json.Unmarshal(authDataJSON, &authData); err != nil {
		log.Printf("[Tesla OAuth] Failed to unmarshal auth data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to decode auth data"})
		return
	}

	// 删除 Redis 中的数据（一次性使用）
	redis.Delete(redisKey)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": authData,
	})
}

type VehicleResponse struct {
	Response []struct {
		ID            int64    `json:"id"`
		VehicleID     int64    `json:"vehicle_id"`
		VIN           string   `json:"vin"`
		DisplayName   string   `json:"display_name"`
		OptionCodes   string   `json:"option_codes"`
		Color         string   `json:"color"`
		AccessType    string   `json:"access_type"`
		GranularAccess struct {
			HidePrivate bool `json:"hide_private"`
		} `json:"granular_access"`
		Tokens                 []string `json:"tokens"`
		State                  string   `json:"state"`
		InService              bool     `json:"in_service"`
		IDS                    string   `json:"id_s"`
		CalendarEnabled        bool     `json:"calendar_enabled"`
		ApiVersion             int      `json:"api_version"`
		BackseatToken          string   `json:"backseat_token"`
		BackseatTokenUpdatedAt int64    `json:"backseat_token_updated_at"`
		VehicleConfig          struct {
			CarType       string `json:"car_type"`
			ExteriorColor string `json:"exterior_color"`
			WheelType     string `json:"wheel_type"`
		} `json:"vehicle_config"`
	} `json:"response"`
}

func GetAuthURL(c *gin.Context) {
	cfg := config.Load()

	platform := c.Query("platform")
	state := generateRandomState()

	if platform == "app" {
		state = "app:" + state
	}

	params := url.Values{}
	params.Add("client_id", cfg.Tesla.ClientID)
	params.Add("redirect_uri", cfg.Tesla.RedirectURI)
	params.Add("response_type", "code")
	params.Add("scope", "openid offline_access vehicle_device_data vehicle_cmds vehicle_charging_cmds vehicle_location")
	params.Add("audience", cfg.Tesla.Audience)
	params.Add("state", state)
	params.Add("prompt", "consent")

	authURL := cfg.Tesla.AuthURL + "?" + params.Encode()

	redis.Set("tesla:oauth:state:"+state, true, 10*time.Minute)

	log.Printf("[Tesla OAuth] auth URL generated, state saved, platform: %s", platform)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"auth_url": authURL,
			"state":    state,
		},
	})
}

func generateRandomState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func Callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")
	errorDescription := c.Query("error_description")
	issuer := c.Query("issuer")

	isApp := strings.HasPrefix(state, "app:")

	cfg := config.Load()

	log.Printf("[Tesla OAuth] Callback called, isApp: %v", isApp)

	if state != "" {
		stateKey := "tesla:oauth:state:" + state
		exists, err := redis.Exists(stateKey)
		if err != nil || !exists {
			log.Printf("[Tesla OAuth] invalid state parameter, possible CSRF attack")
			errorParams := url.Values{}
			errorParams.Add("error", "invalid_state")
			errorParams.Add("error_description", "Invalid state parameter")
			if isApp {
				c.Redirect(http.StatusFound, "teslaapp://callback?"+errorParams.Encode())
			} else {
				frontendCallbackURL := cfg.Tesla.FrontendCallbackURL
				if frontendCallbackURL == "" {
					frontendCallbackURL = "http://localhost:3000/#/pages/callback/callback"
				}
				c.Redirect(http.StatusFound, buildFrontendURL(frontendCallbackURL, errorParams))
			}
			return
		}
		redis.Delete(stateKey)
	}
	log.Printf("[Tesla OAuth] issuer: %s", issuer)

	frontendCallbackURL := cfg.Tesla.FrontendCallbackURL
	if frontendCallbackURL == "" {
		frontendCallbackURL = "http://localhost:3000/#/pages/callback/callback"
	}

	if errorParam != "" {
		log.Printf("[Tesla OAuth] Tesla returned error: error=%s, description=%s", errorParam, errorDescription)
		errorParams := url.Values{}
		errorParams.Add("error", errorParam)
		errorParams.Add("error_description", errorDescription)
		if isApp {
			c.Redirect(http.StatusFound, "teslaapp://callback?"+errorParams.Encode())
		} else {
			c.Redirect(http.StatusFound, buildFrontendURL(frontendCallbackURL, errorParams))
		}
		return
	}

	if code == "" {
		log.Printf("[Tesla OAuth] authorization failed: no code parameter")
		errorParams := url.Values{}
		errorParams.Add("error", "access_denied")
		errorParams.Add("error_description", "Authorization code is required")
		c.Redirect(http.StatusFound, buildFrontendURL(frontendCallbackURL, errorParams))
		return
	}

	log.Printf("[Tesla OAuth] fetching token...")
	log.Printf("[Tesla OAuth] TokenURL: %s", cfg.Tesla.TokenURL)
	log.Printf("[Tesla OAuth] RedirectURI: %s", cfg.Tesla.RedirectURI)

	resp, err := newTeslaClient().
		SetFormData(map[string]string{
			"grant_type":    "authorization_code",
			"client_id":     cfg.Tesla.ClientID,
			"client_secret": cfg.Tesla.ClientSecret,
			"code":          code,
			"redirect_uri":  cfg.Tesla.RedirectURI,
		}).
		Post(cfg.Tesla.TokenURL)

	if err != nil {
		log.Printf("[Tesla OAuth] token request failed: %v", err)
		errParams := url.Values{}
		errParams.Add("error", "token_error")
		errParams.Add("error_description", "Failed to get token")
		c.Redirect(http.StatusFound, buildFrontendURL(frontendCallbackURL, errParams))
		return
	}

	log.Printf("[Tesla OAuth] Token response status: %s", resp.Status())
	log.Printf("[Tesla OAuth] Token response body: %s", string(resp.Body()))

	if resp.StatusCode() != http.StatusOK {
		log.Printf("[Tesla OAuth] Token request failed, HTTP status: %d", resp.StatusCode())
		errParams := url.Values{}
		errParams.Add("error", "token_error")
		errParams.Add("error_description", fmt.Sprintf("Token endpoint returned %d", resp.StatusCode()))
		c.Redirect(http.StatusFound, buildFrontendURL(frontendCallbackURL, errParams))
		return
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(resp.Body(), &tokenResp); err != nil {
		log.Printf("[Tesla OAuth] failed to parse token response: %v", err)
		errParams := url.Values{}
		errParams.Add("error", "parse_error")
		errParams.Add("error_description", "Failed to parse token response")
		c.Redirect(http.StatusFound, buildFrontendURL(frontendCallbackURL, errParams))
		return
	}

	var errorCheck map[string]interface{}
	json.Unmarshal(resp.Body(), &errorCheck)
	if errMsg, ok := errorCheck["error"].(string); ok && errMsg != "" {
		log.Printf("[Tesla OAuth] Token endpoint returned error: %s", errMsg)
		errorDesc, _ := errorCheck["error_description"].(string)
		errParams := url.Values{}
		errParams.Add("error", errMsg)
		errParams.Add("error_description", errorDesc)
		c.Redirect(http.StatusFound, buildFrontendURL(frontendCallbackURL, errParams))
		return
	}

	if tokenResp.AccessToken == "" {
		log.Printf("[Tesla OAuth] error: AccessToken is empty")
		errParams := url.Values{}
		errParams.Add("error", "token_error")
		errParams.Add("error_description", "Empty access token")
		c.Redirect(http.StatusFound, buildFrontendURL(frontendCallbackURL, errParams))
		return
	}

	log.Printf("[Tesla OAuth] token obtained, scope: %s", tokenResp.Scope)
	if !strings.Contains(tokenResp.Scope, "vehicle_location") {
		log.Printf("[Tesla OAuth] WARNING: vehicle_location scope NOT granted! Granted scopes: %s", tokenResp.Scope)
	}
	log.Printf("[Tesla OAuth] token obtained, fetching vehicle list...")

	vehicles, err := fetchVehicles(tokenResp.AccessToken)
	if err != nil {
		log.Printf("[Tesla OAuth] failed to fetch vehicles: %v", err)
		errParams := url.Values{}
		errParams.Add("error", "vehicle_error")
		errParams.Add("error_description", "Failed to fetch vehicles")
		c.Redirect(http.StatusFound, buildFrontendURL(frontendCallbackURL, errParams))
		return
	}

	if vehicles == nil {
		vehicles = []gin.H{}
	}

	log.Printf("[Tesla OAuth] vehicle list obtained, count: %d", len(vehicles))

	authData := gin.H{
		"access_token":  tokenResp.AccessToken,
		"refresh_token": tokenResp.RefreshToken,
		"expires_in":    tokenResp.ExpiresIn,
		"scope":         tokenResp.Scope,
		"id_token":      tokenResp.IDToken, // 用于获取 sub
		"vehicles":      vehicles,
		"state":         state,
	}

	// 使用 Redis 存储 auth_data，避免 URL 过长导致数据丢失
	authDataJSON, err := json.Marshal(authData)
	if err != nil {
		log.Printf("[Tesla OAuth] JSON marshal failed: %v", err)
		errParams := url.Values{}
		errParams.Add("error", "marshal_error")
		errParams.Add("error_description", "Failed to encode auth data")
		if isApp {
			c.Redirect(http.StatusFound, "teslaapp://callback?"+errParams.Encode())
		} else {
			c.Redirect(http.StatusFound, buildFrontendURL(frontendCallbackURL, errParams))
		}
		return
	}

	authID := generateAuthID()
	redisKey := fmt.Sprintf("tesla:oauth:%s", authID)

	if err := redis.Set(redisKey, authDataJSON, 10*time.Minute); err != nil {
		log.Printf("[Tesla OAuth] Failed to store auth data in redis: %v", err)
		authDataEncoded := base64.URLEncoding.EncodeToString(authDataJSON)
		successParams := url.Values{}
		successParams.Add("auth_data", authDataEncoded)
		if isApp {
			c.Redirect(http.StatusFound, "teslaapp://callback?"+successParams.Encode())
		} else {
			c.Redirect(http.StatusFound, buildFrontendURL(frontendCallbackURL, successParams))
		}
		return
	}

	log.Printf("[Tesla OAuth] auth_data stored in redis with key: %s", authID)

	successParams := url.Values{}
	successParams.Add("auth_id", authID)

	if isApp {
		c.Redirect(http.StatusFound, "teslaapp://callback?"+successParams.Encode())
	} else {
		c.Redirect(http.StatusFound, buildFrontendURL(frontendCallbackURL, successParams))
	}
}

// BindVehicle 绑定车辆（生产级流程）
// Step 1: 保存 TeslaOAuthAccount（账户级 token）
// Step 2: 保存 TeslaVehicle（车辆信息，包含 vehicle_tag）
// Step 3: 验证关键权限（fleet_status, vehicle_data）
// Step 4: 缓存到 Redis
func BindVehicle(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		AccessToken  string `json:"access_token" binding:"required"`
		RefreshToken string `json:"refresh_token" binding:"required"`
		ExpiresIn    int    `json:"expires_in"`
		VIN          string `json:"vin" binding:"required"`
		// scope 和 sub 从 access_token JWT 中解析，不需要前端传入
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// Step 1: 获取车辆列表，找到对应的车辆信息
	vehicleDetail, err := fetchVehicleDetail(req.AccessToken, req.VIN)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Failed to fetch vehicle info"})
		return
	}

	log.Printf("[BindVehicle] Vehicle detail fetched - VIN: %s, VehicleTag(id_s): %s", req.VIN, vehicleDetail.IDS)

	// Step 2: 解析 access_token JWT 获取 sub 和 scope
	// Tesla 中国区的 token 端点不返回 scope，需要从 JWT 中解析
	claims, err := ParseTeslaJWT(req.AccessToken)
	if err != nil {
		log.Printf("[BindVehicle] Failed to parse Tesla token: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Failed to parse Tesla token"})
		return
	}

	teslaUID := claims.Sub
	if teslaUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Tesla sub is empty"})
		return
	}
	log.Printf("[BindVehicle] Tesla UID(sub): %s", teslaUID)

	// 从 JWT 中获取 scope（Tesla CN 不返回 scope 字段）
	grantedScope := strings.Join(claims.SCP, " ")
	log.Printf("[BindVehicle] Granted scopes from JWT: %s", grantedScope)

	// Step 3: 保存/更新 TeslaOAuthAccount（账户级 token）
	var oauthAccount models.TeslaOAuthAccount
	result := database.DB.Where("user_id = ? AND tesla_uid = ?", userID, teslaUID).First(&oauthAccount)

	expiresAt := time.Now().UTC().Add(time.Duration(req.ExpiresIn) * time.Second).Unix()

	if result.Error != nil {
		// 新建账户
		oauthAccount = models.TeslaOAuthAccount{
			UserID:        userID,
			TeslaUID:      teslaUID,
			AccessToken:   req.AccessToken,
			RefreshToken:  req.RefreshToken,
			ExpiresAt:     expiresAt,
			GrantedScopes: grantedScope, // 从 JWT 中获取
			TokenInvalid:  false,
		}
		if err := database.DB.Create(&oauthAccount).Error; err != nil {
			log.Printf("[BindVehicle] Failed to create TeslaOAuthAccount: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to save account: " + err.Error()})
			return
		}
		log.Printf("[BindVehicle] Created new TeslaOAuthAccount for user %d", userID)
	} else {
		// 更新账户 token
		oauthAccount.AccessToken = req.AccessToken
		oauthAccount.RefreshToken = req.RefreshToken
		oauthAccount.ExpiresAt = expiresAt
		oauthAccount.GrantedScopes = grantedScope // 从 JWT 中获取
		oauthAccount.TokenInvalid = false
		if err := database.DB.Save(&oauthAccount).Error; err != nil {
			log.Printf("[BindVehicle] Failed to update TeslaOAuthAccount: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to update account: " + err.Error()})
			return
		}
		log.Printf("[BindVehicle] Updated TeslaOAuthAccount for user %d", userID)
	}

	// Step 4: 保存/更新 TeslaVehicle（车辆信息）
	var vehicle models.TeslaVehicle
	vehicleResult := database.DB.Where("user_id = ? AND vin = ?", userID, req.VIN).First(&vehicle)

	log.Printf("[BindVehicle] Vehicle query result - Error: %v, Error type: %T, Vehicle found: %v", vehicleResult.Error, vehicleResult.Error, vehicleResult.RowsAffected)

	if vehicleResult.Error != nil {
		// 新建车辆
		vehicle = models.TeslaVehicle{
			UserID:       userID,
			TeslaUID:     teslaUID,
			VIN:          req.VIN,
			VehicleTag:   vehicleDetail.IDS,
			DisplayName:  vehicleDetail.DisplayName,
			AccessType:   vehicleDetail.AccessType,
			BindStatus:   1,
			OnlineState:  vehicleDetail.State,
			ApiVersion:   vehicleDetail.ApiVersion,
			OptionCodes:  vehicleDetail.OptionCodes,
			VehicleImage: getVehicleImage(vehicleDetail.DisplayName, vehicleDetail.OptionCodes),
		}
		log.Printf("[BindVehicle] Creating new TeslaVehicle - UserID: %d, VIN: %s, VehicleTag: %s", userID, req.VIN, vehicleDetail.IDS)
		if err := database.DB.Create(&vehicle).Error; err != nil {
			log.Printf("[BindVehicle] Failed to create TeslaVehicle: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to save vehicle: " + err.Error()})
			return
		}
		log.Printf("[BindVehicle] Created new TeslaVehicle - VIN: %s, VehicleTag: %s, ID: %d", req.VIN, vehicleDetail.IDS, vehicle.ID)
	} else {
		// 更新车辆信息（关键：也要更新 tesla_uid，从旧版的 VIN 更新为 sub）
		log.Printf("[BindVehicle] Updating existing TeslaVehicle - ID: %d, VIN: %s, old_tesla_uid: %s, new_tesla_uid: %s", vehicle.ID, vehicle.VIN, vehicle.TeslaUID, teslaUID)
		vehicle.TeslaUID = teslaUID
		vehicle.VehicleTag = vehicleDetail.IDS
		vehicle.DisplayName = vehicleDetail.DisplayName
		vehicle.AccessType = vehicleDetail.AccessType
		vehicle.OnlineState = vehicleDetail.State
		vehicle.ApiVersion = vehicleDetail.ApiVersion
		vehicle.OptionCodes = vehicleDetail.OptionCodes
		vehicle.VehicleImage = getVehicleImage(vehicleDetail.DisplayName, vehicleDetail.OptionCodes)
		vehicle.BindStatus = 1
		if err := database.DB.Save(&vehicle).Error; err != nil {
			log.Printf("[BindVehicle] Failed to update TeslaVehicle: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to update vehicle: " + err.Error()})
			return
		}
		log.Printf("[BindVehicle] Updated TeslaVehicle - VIN: %s, VehicleTag: %s", req.VIN, vehicleDetail.IDS)
	}

	// Step 5: 验证关键权限（异步）
	go func() {
		// 5.1 验证 vehicle_device_data（拉取 vehicle_data）
		_, err := fleet.GetVehicleState(req.AccessToken, vehicleDetail.IDS)
		if err != nil {
			log.Printf("[BindVehicle] vehicle_data verification failed for %s: %v", req.VIN, err)
		} else {
			log.Printf("[BindVehicle] vehicle_data verification passed for %s", req.VIN)
		}

		// 5.2 验证 fleet_status（virtual key）
		status, err := fleet.VerifyVirtualKey(req.AccessToken, []string{req.VIN})
		if err != nil {
			log.Printf("[BindVehicle] fleet_status verification failed for %s: %v", req.VIN, err)
		} else {
			log.Printf("[BindVehicle] fleet_status verification passed for %s, key_paired: %v", req.VIN, status.KeyPaired)
			// 更新车辆 virtual key 状态
			now := time.Now().UTC()
			updates := map[string]interface{}{
				"virtual_key_status":      boolToInt(status.KeyPaired),
				"virtual_key_last_check":  now,
				"fleet_telemetry_version": status.FleetTelemetry,
				"discounted_device_data":  status.DiscountedData,
			}
			if status.KeyPaired {
				updates["virtual_key_paired_at"] = now
			}
			database.DB.Model(&models.TeslaVehicle{}).Where("vin = ?", req.VIN).Updates(updates)
		}
	}()

	// Step 6: 缓存到 Redis
	go func() {
		mapping := &redis.VehicleMapping{
			VIN:         req.VIN,
			VehicleTag:  vehicleDetail.IDS,
			AccessToken: req.AccessToken,
			UserID:      userID,
		}
		if err := redis.SetVehicleMapping(mapping); err != nil {
			log.Printf("[BindVehicle] Failed to cache vehicle mapping: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Vehicle bound successfully",
		"data": gin.H{
			"vin":          req.VIN,
			"vehicle_tag":  vehicleDetail.IDS,
			"display_name": vehicleDetail.DisplayName,
		},
	})
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func GetUserVehicles(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var vehicles []models.TeslaVehicle
	if err := database.DB.Where("user_id = ? AND bind_status = 1", userID).Find(&vehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to fetch vehicles"})
		return
	}

	var result []gin.H
	for _, v := range vehicles {
		result = append(result, gin.H{
			"id":                 v.ID,
			"vin":                v.VIN,
			"vehicle_tag":        v.VehicleTag,
			"display_name":       v.DisplayName,
			"access_type":        v.AccessType,
			"online_state":       v.OnlineState,
			"virtual_key_status": v.VirtualKeyStatus,
			"bind_status":        v.BindStatus,
			"option_codes":       v.OptionCodes,
			"vehicle_image":      v.VehicleImage,
			"created_at":         v.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": result,
	})
}

func GetVehicleDetail(c *gin.Context) {
	userID := middleware.GetUserID(c)
	vin := c.Param("vin")

	var vehicle models.TeslaVehicle
	if err := database.DB.Where("user_id = ? AND vin = ? AND bind_status = 1", userID, vin).First(&vehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Vehicle not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"id":                      vehicle.ID,
			"vin":                     vehicle.VIN,
			"vehicle_tag":             vehicle.VehicleTag,
			"display_name":            vehicle.DisplayName,
			"access_type":             vehicle.AccessType,
			"online_state":            vehicle.OnlineState,
			"virtual_key_status":      vehicle.VirtualKeyStatus,
			"virtual_key_paired_at":   vehicle.VirtualKeyPairedAt,
			"virtual_key_last_check":  vehicle.VirtualKeyLastCheck,
			"location_authorized":     vehicle.LocationAuthorized,
			"fleet_telemetry_version": vehicle.FleetTelemetryVersion,
			"discounted_device_data":  vehicle.DiscountedDeviceData,
			"api_version":             vehicle.ApiVersion,
			"option_codes":            vehicle.OptionCodes,
			"vehicle_image":           vehicle.VehicleImage,
			"bind_status":             vehicle.BindStatus,
			"created_at":              vehicle.CreatedAt,
		},
	})
}

func GetFleetStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)
	vin := c.Param("vin")

	// 获取车辆信息
	var vehicle models.TeslaVehicle
	if err := database.DB.Where("user_id = ? AND vin = ? AND bind_status = 1", userID, vin).First(&vehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Vehicle not found"})
		return
	}

	// 获取账户 token
	var oauthAccount models.TeslaOAuthAccount
	if err := database.DB.Where("user_id = ? AND tesla_uid = ?", userID, vehicle.TeslaUID).First(&oauthAccount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "OAuth account not found"})
		return
	}

	status, err := fleet.VerifyVirtualKey(oauthAccount.AccessToken, []string{vin})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"data": gin.H{
				"key_paired":                false,
				"key_count":                 0,
				"command_protocol_required": true,
				"signed_command_available":  true,
				"fleet_telemetry_version":   "",
				"discounted_device_data":    false,
			},
		})
		return
	}

	virtualKeyStatus := 0
	if status.KeyPaired {
		virtualKeyStatus = 1
	}

	now := time.Now().UTC()
	updates := map[string]interface{}{
		"virtual_key_status":       virtualKeyStatus,
		"virtual_key_last_check":   now,
		"fleet_telemetry_version":  status.FleetTelemetry,
		"discounted_device_data":   status.DiscountedData,
	}
	if status.KeyPaired {
		updates["virtual_key_paired_at"] = now
	}

	database.DB.Model(&models.TeslaVehicle{}).
		Where("vin = ?", vin).
		Updates(updates)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"key_paired":                status.KeyPaired,
			"key_count":                 status.KeyCount,
			"command_protocol_required": status.CommandRequired,
			"signed_command_available":  status.SignedCommand,
			"fleet_telemetry_version":   status.FleetTelemetry,
			"discounted_device_data":    status.DiscountedData,
		},
	})
}

func UnbindVehicle(c *gin.Context) {
	userID := middleware.GetUserID(c)
	vin := c.Param("vin")

	// 删除车辆绑定
	if err := database.DB.Where("user_id = ? AND vin = ?", userID, vin).Delete(&models.TeslaVehicle{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to unbind vehicle"})
		return
	}

	// 清除 Redis 缓存
	redis.DeleteVehicleMapping(vin)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Vehicle unbound successfully",
	})
}

func fetchVehicles(accessToken string) ([]gin.H, error) {
	cfg := config.Load()
	client := resty.New().
		SetRedirectPolicy(resty.FlexibleRedirectPolicy(0)).
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36").
		SetHeader("Accept", "application/json")

	apiURL := cfg.Tesla.FleetAPIURL + "/api/1/vehicles"
	log.Printf("[Tesla API] fetching vehicles: %s", apiURL)

	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+accessToken).
		SetHeader("Content-Type", "application/json").
		Get(apiURL)

	if err != nil {
		log.Printf("[Tesla API] vehicle list request failed: %v", err)
		return nil, err
	}

	log.Printf("[Tesla API] vehicle list response status: %d", resp.StatusCode())

	if resp.StatusCode() != http.StatusOK {
		var errorResp map[string]interface{}
		if err := json.Unmarshal(resp.Body(), &errorResp); err == nil {
			if errMsg, ok := errorResp["error"].(string); ok {
				return nil, fmt.Errorf("vehicles API error: %s", errMsg)
			}
		}
		return nil, fmt.Errorf("vehicles API returned status %d", resp.StatusCode())
	}

	var vehicleResp VehicleResponse
	if err := json.Unmarshal(resp.Body(), &vehicleResp); err != nil {
		log.Printf("[Tesla API] failed to parse vehicle list: %v", err)
		return nil, err
	}

	var vehicles []gin.H
	for _, v := range vehicleResp.Response {
		vehicleConfigJSON, _ := json.Marshal(v.VehicleConfig)
		vehicles = append(vehicles, gin.H{
			"id":               v.ID,
			"vin":              v.VIN,
			"display_name":     v.DisplayName,
			"state":            v.State,
			"option_codes":     v.OptionCodes,
			"color":            v.Color,
			"access_type":      v.AccessType,
			"in_service":       v.InService,
			"calendar_enabled": v.CalendarEnabled,
			"api_version":      v.ApiVersion,
			"id_s":             v.IDS, // 关键：返回 id_s 给前端
			"vehicle_config":   json.RawMessage(vehicleConfigJSON),
		})
	}

	return vehicles, nil
}

type VehicleDetail struct {
	ID                     int64
	VehicleID              int64
	VIN                    string
	DisplayName            string
	OptionCodes            string
	Color                  string
	AccessType             string
	GranularAccess         interface{}
	Tokens                 []string
	State                  string
	InService              bool
	IDS                    string
	CalendarEnabled        bool
	ApiVersion             int
	BackseatToken          string
	BackseatTokenUpdatedAt int64
	VehicleConfig          interface{}
}

func fetchVehicleDetail(accessToken, vin string) (*VehicleDetail, error) {
	cfg := config.Load()
	client := resty.New().
		SetRedirectPolicy(resty.FlexibleRedirectPolicy(0)).
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36").
		SetHeader("Accept", "application/json")

	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+accessToken).
		SetHeader("Content-Type", "application/json").
		Get(cfg.Tesla.FleetAPIURL + "/api/1/vehicles")

	if err != nil {
		return nil, err
	}

	log.Printf("[Tesla API] /api/1/vehicles response: %s", string(resp.Body()))

	var vehicleResp VehicleResponse
	if err := json.Unmarshal(resp.Body(), &vehicleResp); err != nil {
		return nil, err
	}

	for _, v := range vehicleResp.Response {
		log.Printf("[Tesla API] Vehicle - ID: %d, VehicleID: %d, VIN: %s, id_s: %s", v.ID, v.VehicleID, v.VIN, v.IDS)
		if v.VIN == vin {
			log.Printf("[Tesla API] Found vehicle %s, id_s: %s", vin, v.IDS)
			return &VehicleDetail{
				ID:                     v.ID,
				VehicleID:              v.VehicleID,
				VIN:                    v.VIN,
				DisplayName:            v.DisplayName,
				OptionCodes:            v.OptionCodes,
				Color:                  v.Color,
				AccessType:             v.AccessType,
				GranularAccess:         v.GranularAccess,
				Tokens:                 v.Tokens,
				State:                  v.State,
				InService:              v.InService,
				IDS:                    v.IDS,
				CalendarEnabled:        v.CalendarEnabled,
				ApiVersion:             v.ApiVersion,
				BackseatToken:          v.BackseatToken,
				BackseatTokenUpdatedAt: v.BackseatTokenUpdatedAt,
				VehicleConfig:          v.VehicleConfig,
			}, nil
		}
	}

	return nil, fmt.Errorf("vehicle %s not found in Tesla account", vin)
}

// RefreshToken 刷新 token
// 注意：刷新后必须同时更新数据库中的 access_token, refresh_token, expires_at
// 因为 Tesla 有时会在刷新时同时更新 refresh_token
// 注意：refresh 时不要传 scope，避免 invalid_scope 错误
func RefreshToken(refreshToken string) (*TokenResponse, error) {
	cfg := config.Load()
	client := resty.New().
		SetRedirectPolicy(resty.FlexibleRedirectPolicy(0)).
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36").
		SetHeader("Accept", "application/json").
		SetHeader("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	log.Printf("[RefreshToken] TokenURL: %s", cfg.Tesla.TokenURL)

	if cfg.Tesla.TokenURL[:5] != "https" {
		log.Printf("[RefreshToken] WARNING: TokenURL is not HTTPS! URL=%s, forcing HTTPS", cfg.Tesla.TokenURL)
	}

	formData := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"client_id":     cfg.Tesla.ClientID,
	}

	resp, err := client.R().
		SetFormData(formData).
		Post(cfg.Tesla.TokenURL)

	if err != nil {
		log.Printf("[RefreshToken] Request error: %v", err)
		return nil, err
	}

	log.Printf("[RefreshToken] Response status: %d, URL used: %s", resp.StatusCode(), cfg.Tesla.TokenURL)

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("token refresh failed: HTTP %d - %s", resp.StatusCode(), string(resp.Body()))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(resp.Body(), &tokenResp); err != nil {
		return nil, err
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("token refresh returned empty access_token")
	}

	log.Printf("[RefreshToken] Success, expires_in=%d, has_new_refresh=%v",
		tokenResp.ExpiresIn, tokenResp.RefreshToken != "")

	return &tokenResp, nil
}

// RefreshTokenForVehicle 为指定车辆刷新 token
// 通过 VIN 找到对应的 TeslaOAuthAccount，刷新 token，更新数据库和 Redis
func RefreshTokenForVehicle(vin string) (*TokenResponse, error) {
	var vehicle models.TeslaVehicle
	if err := database.DB.Where("vin = ? AND bind_status = 1", vin).First(&vehicle).Error; err != nil {
		return nil, fmt.Errorf("vehicle not found: %s", vin)
	}

	var oauthAccount models.TeslaOAuthAccount
	if err := database.DB.Where("user_id = ? AND tesla_uid = ?", vehicle.UserID, vehicle.TeslaUID).First(&oauthAccount).Error; err != nil {
		return nil, fmt.Errorf("oauth account not found for vehicle: %s", vin)
	}

	if oauthAccount.TokenInvalid {
		return nil, fmt.Errorf("token has been marked as invalid, please re-authorize")
	}

	tokenResp, err := RefreshToken(oauthAccount.RefreshToken)
	if err != nil {
		now := time.Now()
		errMsg := err.Error()
		isPermanent := strings.Contains(errMsg, "invalid_grant") ||
			strings.Contains(errMsg, "invalid_client") ||
			strings.Contains(errMsg, "login_required")

		updates := map[string]interface{}{
			"last_token_refresh_error": errMsg,
			"last_token_refresh_at":    now,
		}

		if isPermanent {
			updates["token_invalid"] = true
			log.Printf("[RefreshTokenForVehicle] %s token permanently invalid, user must re-authorize: %s", vin, errMsg)
		} else {
			log.Printf("[RefreshTokenForVehicle] %s token refresh failed (transient): %s", vin, errMsg)
		}

		database.DB.Model(&oauthAccount).Updates(updates)
		return nil, fmt.Errorf("token refresh failed: %v", err)
	}

	newRefreshToken := oauthAccount.RefreshToken
	if tokenResp.RefreshToken != "" {
		newRefreshToken = tokenResp.RefreshToken
		log.Printf("[RefreshTokenForVehicle] %s got new refresh_token, saving to DB", vin)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Unix()

	database.DB.Model(&oauthAccount).Updates(map[string]interface{}{
		"access_token":            tokenResp.AccessToken,
		"refresh_token":           newRefreshToken,
		"expires_at":              expiresAt,
		"last_token_refresh_at":   now,
		"token_invalid":           false,
		"last_token_refresh_error": "",
	})

	go func() {
		mapping := &redis.VehicleMapping{
			VIN:         vin,
			VehicleTag:  vehicle.VehicleTag,
			AccessToken: tokenResp.AccessToken,
			UserID:      vehicle.UserID,
			ExpiresAt:   expiresAt,
		}
		if err := redis.SetVehicleMapping(mapping); err != nil {
			log.Printf("[RefreshTokenForVehicle] Failed to update Redis cache: %v", err)
		}
	}()

	return tokenResp, nil
}

// GetValidAccessToken 获取有效的 access_token（自动刷新）
// 优先从 Redis 获取，如果即将过期则自动刷新
func GetValidAccessToken(vin string) (string, error) {
	// 1. 先查 Redis
	mapping, err := redis.GetVehicleMapping(vin)
	if err == nil && mapping != nil && mapping.AccessToken != "" {
		now := time.Now().UTC().Unix()
		if mapping.ExpiresAt > now+1800 {
			return mapping.AccessToken, nil
		}
		log.Printf("[GetValidAccessToken] Redis token for %s expiring soon (expires_at=%d, now=%d), need refresh", vin, mapping.ExpiresAt, now)
	}

	// 2. Redis 未命中或即将过期，查数据库
	var vehicle models.TeslaVehicle
	if err := database.DB.Where("vin = ? AND bind_status = 1", vin).First(&vehicle).Error; err != nil {
		return "", fmt.Errorf("vehicle not found: %s", vin)
	}

	// 获取账户信息（必须加上 user_id，防止一个 Tesla 账户被多个用户绑定时的串号问题）
	log.Printf("[GetValidAccessToken] Looking for oauth account: user_id=%d, tesla_uid=%s", vehicle.UserID, vehicle.TeslaUID)
	var oauthAccount models.TeslaOAuthAccount
	if err := database.DB.Where("user_id = ? AND tesla_uid = ?", vehicle.UserID, vehicle.TeslaUID).First(&oauthAccount).Error; err != nil {
		// 兼容性处理：如果查不到，可能是旧数据使用 VIN 作为 tesla_uid
		// 尝试用 VIN 查询
		log.Printf("[GetValidAccessToken] Trying fallback with VIN: user_id=%d, vin=%s", vehicle.UserID, vin)
		if err := database.DB.Where("user_id = ? AND tesla_uid = ?", vehicle.UserID, vin).First(&oauthAccount).Error; err != nil {
			log.Printf("[GetValidAccessToken] OAuth account not found: user_id=%d, tesla_uid=%s, error=%v", vehicle.UserID, vehicle.TeslaUID, err)
			return "", fmt.Errorf("oauth account not found")
		}
		// 找到旧数据，更新 tesla_uid 为正确的 sub
		log.Printf("[GetValidAccessToken] Found old data with VIN, updating tesla_uid from %s to %s", vin, vehicle.TeslaUID)
		oauthAccount.TeslaUID = vehicle.TeslaUID
		database.DB.Save(&oauthAccount)
	}

	now := time.Now().UTC().Unix()
	if oauthAccount.ExpiresAt < now+1800 {
		// 需要刷新，加分布式锁防止并发刷新
		locked, err := redis.AcquireTokenRefreshLock(vin)
		if err != nil {
			log.Printf("[GetValidAccessToken] Failed to acquire lock for %s: %v", vin, err)
			// 锁失败，继续尝试刷新（可能失败，但不会阻塞）
		}

		if !locked {
			// 其他协程正在刷新，等待后重试（最多3次）
			log.Printf("[GetValidAccessToken] Another goroutine is refreshing token for %s, waiting...", vin)
			for i := 0; i < 3; i++ {
				time.Sleep(time.Second)
				mapping, err := redis.GetVehicleMapping(vin)
				if err == nil && mapping != nil && mapping.AccessToken != "" &&
					mapping.ExpiresAt > time.Now().UTC().Unix()+1800 {
					return mapping.AccessToken, nil
				}
			}
			return "", fmt.Errorf("token refresh timeout")
		}

		// 获取锁成功，刷新 token
		log.Printf("[GetValidAccessToken] Token for %s expiring soon, refreshing...", vin)
		tokenResp, err := RefreshTokenForVehicle(vin)
		redis.ReleaseTokenRefreshLock(vin) // 释放锁

		if err != nil {
			return "", err
		}
		return tokenResp.AccessToken, nil
	}

	// 4. 缓存到 Redis（带上过期时间）- 同步执行，不要 goroutine
	newMapping := &redis.VehicleMapping{
		VIN:         vin,
		VehicleTag:  vehicle.VehicleTag,
		AccessToken: oauthAccount.AccessToken,
		UserID:      vehicle.UserID,
		ExpiresAt:   oauthAccount.ExpiresAt,
	}
	if err := redis.SetVehicleMapping(newMapping); err != nil {
		log.Printf("[GetValidAccessToken] Failed to cache mapping: %v", err)
	}

	return oauthAccount.AccessToken, nil
}

func buildFrontendURL(baseURL string, params url.Values) string {
	hashIndex := strings.Index(baseURL, "#")
	if hashIndex == -1 {
		return baseURL + "?" + params.Encode()
	}
	base := baseURL[:hashIndex]
	hash := baseURL[hashIndex:]
	if strings.Contains(base, "?") {
		return base + "&" + params.Encode() + hash
	}
	return base + "?" + params.Encode() + hash
}

func newTeslaClient() *resty.Request {
	return resty.New().
		SetRedirectPolicy(resty.FlexibleRedirectPolicy(0)).
		R().
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36").
		SetHeader("Accept", "application/json").
		SetHeader("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
}

func getPartnerToken() (string, error) {
	cfg := config.Load()

	resp, err := newTeslaClient().
		SetFormData(map[string]string{
			"grant_type":    "client_credentials",
			"client_id":     cfg.Tesla.ClientID,
			"client_secret": cfg.Tesla.ClientSecret,
			"scope":         "openid vehicle_device_data vehicle_cmds vehicle_charging_cmds vehicle_location",
			"audience":      cfg.Tesla.Audience,
		}).
		Post(cfg.Tesla.TokenURL)

	if err != nil {
		return "", fmt.Errorf("partner token request failed: %v", err)
	}

	log.Printf("[Tesla Partner] Token response status: %s", resp.Status())
	log.Printf("[Tesla Partner] Token response body: %s", string(resp.Body()))

	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("partner token request returned status %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(resp.Body(), &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse partner token response: %v", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("partner access token is empty")
	}

	return tokenResp.AccessToken, nil
}

func RegisterPartnerAccount(c *gin.Context) {
	cfg := config.Load()

	partnerToken, err := getPartnerToken()
	if err != nil {
		log.Printf("[Tesla Partner] Failed to get partner token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("Failed to get partner token: %v", err),
		})
		return
	}

	log.Printf("[Tesla Partner] Got partner token: %s...", partnerToken[:20])

	domain := cfg.Tesla.PartnerDomain
	if domain == "" {
		domain = "your-domain.com"
	}
	if reqDomain := c.Query("domain"); reqDomain != "" {
		domain = reqDomain
	}

	resp, err := newTeslaClient().
		SetHeader("Authorization", "Bearer "+partnerToken).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]string{
			"domain": domain,
		}).
		Post(cfg.Tesla.FleetAPIURL + "/api/1/partner_accounts")

	if err != nil {
		log.Printf("[Tesla Partner] Register request failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("Register request failed: %v", err),
		})
		return
	}

	log.Printf("[Tesla Partner] Register response status: %s", resp.Status())
	log.Printf("[Tesla Partner] Register response body: %s", string(resp.Body()))

	if resp.StatusCode() != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("Register failed with status %d: %s", resp.StatusCode(), string(resp.Body())),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Partner account registered successfully",
		"data":    string(resp.Body()),
	})
}

func GetVirtualKeyPairingURL(c *gin.Context) {
	userID := middleware.GetUserID(c)
	vin := c.Query("vin")

	if vin == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "vin is required"})
		return
	}

	var vehicle models.TeslaVehicle
	if err := database.DB.Where("user_id = ? AND vin = ? AND bind_status = 1", userID, vin).First(&vehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Vehicle not found"})
		return
	}

	cfg := config.Load()
	domain := cfg.Tesla.PartnerDomain
	if domain == "" {
		domain = c.Query("domain")
	}
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Partner domain not configured. Set TESLA_PARTNER_DOMAIN env variable."})
		return
	}

	pairingURL := fmt.Sprintf("https://tesla.cn/_ak/%s", domain)
	if vin != "" {
		pairingURL += fmt.Sprintf("?vin=%s", vin)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"pairing_url":    pairingURL,
			"domain":         domain,
			"vin":            vin,
			"current_status": vehicle.VirtualKeyStatus,
			"instructions": []string{
				"1. 点击 pairing_url 在浏览器中打开",
				"2. 使用 Tesla App 扫码或直接在 Tesla App 中打开该链接",
				"3. 在 Tesla App 中确认添加虚拟钥匙",
				"4. 等待车辆确认配对（车辆需在线）",
				"5. 配对完成后，调用 /api/tesla/vehicle/{vin}/fleet-status 检查状态",
			},
		},
	})
}

func CheckPublicKeyHosting(c *gin.Context) {
	cfg := config.Load()
	domain := cfg.Tesla.PartnerDomain
	if domain == "" {
		domain = c.Query("domain")
	}
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "domain is required"})
		return
	}

	publicKeyURL := fmt.Sprintf("https://%s/.well-known/appspecific/com.tesla.3p.public-key.pem", domain)

	resp, err := resty.New().SetTimeout(10 * time.Second).R().
		SetHeader("Range", "bytes=0-200").
		Get(publicKeyURL)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"data": gin.H{
				"accessible":  false,
				"url":         publicKeyURL,
				"domain":      domain,
				"error":       fmt.Sprintf("Failed to access public key URL: %v", err),
				"suggestion":  "请确保公钥文件已部署到 https://{domain}/.well-known/appspecific/com.tesla.3p.public-key.pem",
			},
		})
		return
	}

	accessible := resp.StatusCode() == 200 || resp.StatusCode() == 206
	body := string(resp.Body())

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"accessible":    accessible,
			"url":           publicKeyURL,
			"domain":        domain,
			"status_code":   resp.StatusCode(),
			"key_preview":   body[:min(len(body), 100)],
			"suggestion":    "",
		},
	})
}

func CheckPartnerPublicKey(c *gin.Context) {
	cfg := config.Load()

	partnerToken, err := getPartnerToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("Failed to get partner token: %v", err)})
		return
	}

	domain := cfg.Tesla.PartnerDomain
	if domain == "" {
		domain = c.Query("domain")
	}
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "domain is required"})
		return
	}

	resp, err := newTeslaClient().
		SetHeader("Authorization", "Bearer "+partnerToken).
		Get(cfg.Tesla.FleetAPIURL + "/api/1/partner_accounts/public_key?domain=" + domain)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("Failed to check partner public key: %v", err)})
		return
	}

	log.Printf("[Tesla Partner] Public key check response: status=%d, body=%s", resp.StatusCode(), string(resp.Body()))

	if resp.StatusCode() != http.StatusOK {
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"data": gin.H{
				"registered":  false,
				"domain":      domain,
				"status_code": resp.StatusCode(),
				"raw_response": string(resp.Body()),
				"suggestion":  "公钥未在 Tesla 注册，请先调用 /api/tesla/partner/register 注册",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"registered":   true,
			"domain":       domain,
			"raw_response": string(resp.Body()),
		},
	})
}

func RefreshVehicleInfo(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var vehicles []models.TeslaVehicle
	if err := database.DB.Where("user_id = ? AND bind_status = 1", userID).Find(&vehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to fetch vehicles"})
		return
	}

	if len(vehicles) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "No vehicles to refresh", "data": gin.H{"updated": 0}})
		return
	}

	updated := 0
	for _, v := range vehicles {
		accessToken, err := GetValidAccessToken(v.VIN)
		if err != nil {
			log.Printf("[RefreshVehicleInfo] Failed to get token for %s: %v", v.VIN, err)
			continue
		}

		detail, err := fetchVehicleDetail(accessToken, v.VIN)
		if err != nil {
			log.Printf("[RefreshVehicleInfo] Failed to fetch detail for %s: %v", v.VIN, err)
			continue
		}

		v.OptionCodes = detail.OptionCodes
		v.VehicleImage = getVehicleImage(detail.DisplayName, detail.OptionCodes)
		if detail.DisplayName != "" {
			v.DisplayName = detail.DisplayName
		}
		if detail.AccessType != "" {
			v.AccessType = detail.AccessType
		}
		if detail.State != "" {
			v.OnlineState = detail.State
		}
		v.ApiVersion = detail.ApiVersion

		if err := database.DB.Save(&v).Error; err != nil {
			log.Printf("[RefreshVehicleInfo] Failed to save %s: %v", v.VIN, err)
			continue
		}
		updated++
		log.Printf("[RefreshVehicleInfo] Updated %s, option_codes: %s", v.VIN, v.OptionCodes)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": fmt.Sprintf("Refreshed %d/%d vehicles", updated, len(vehicles)),
		"data": gin.H{
			"updated": updated,
			"total":   len(vehicles),
		},
	})
}

func getVehicleImage(displayName, optionCodes string) string {
	model := "my"
	name := strings.ToLower(displayName)
	if strings.Contains(name, "model y") || strings.Contains(name, "modely") {
		model = "my"
	} else if strings.Contains(name, "model 3") || strings.Contains(name, "model3") {
		model = "m3"
	} else if strings.Contains(name, "model s") || strings.Contains(name, "models") {
		model = "ms"
	} else if strings.Contains(name, "model x") || strings.Contains(name, "modelx") {
		model = "mx"
	} else {
		codes := strings.ToUpper(optionCodes)
		if strings.Contains(codes, "MDLY") || strings.Contains(codes, "MTY03") || strings.Contains(codes, "MTY02") {
			model = "my"
		} else if strings.Contains(codes, "MDL3") || strings.Contains(codes, "MT3") {
			model = "m3"
		} else if strings.Contains(codes, "MDLS") || strings.Contains(codes, "MTS") {
			model = "ms"
		} else if strings.Contains(codes, "MDLX") || strings.Contains(codes, "MTX") {
			model = "mx"
		}
	}

	params := []string{"view=STUD_3QTR", "model=" + model}
	if optionCodes != "" {
		params = append(params, "options="+url.QueryEscape(optionCodes))
	}
	return "https://static-assets.tesla.com/v1/compositor/?" + strings.Join(params, "&")
}
