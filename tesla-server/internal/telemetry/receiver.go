package telemetry

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"tesla-server/internal/redis"
	"tesla-server/internal/ws"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/teslamotors/fleet-telemetry/proto/vehicle_data"
)

var (
	mediaMu      sync.RWMutex
	latestMedia  = make(map[string]*MediaState)
	server       *http.Server
	privateKey   *ecdsa.PrivateKey
)

type MediaState struct {
	PlaybackStatus string `json:"media_playback_status"`
	AudioSource    string `json:"media_audio_source"`
	Volume         int    `json:"media_volume"`
	NowPlayingTitle  string `json:"now_playing_title"`
	NowPlayingArtist string `json:"now_playing_artist"`
	NowPlayingAlbum  string `json:"now_playing_album"`
	NowPlayingDuration int `json:"now_playing_duration"`
	NowPlayingElapsed  int `json:"now_playing_elapsed"`
	UpdatedAt      int64  `json:"updated_at"`
}

func InitTelemetryServer(addr string, privKeyPEM []byte) error {
	if addr == "" {
		addr = ":4443"
	}

	if len(privKeyPEM) > 0 {
		key, err := parseECPrivateKey(privKeyPEM)
		if err != nil {
			log.Printf("[Telemetry] Warning: failed to parse private key: %v, running without message verification", err)
		} else {
			privateKey = key
			log.Printf("[Telemetry] Private key loaded for message verification")
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/1/vehicles/", handleTelemetry)

	server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Printf("[Telemetry] Server starting on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Telemetry] Server error: %v", err)
		}
	}()

	return nil
}

func handleTelemetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	vin := pathParts[3]

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[Telemetry] Failed to read body for %s: %v", vin, err)
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var payload vehicle_data.VehicleData
	if err := proto.Unmarshal(body, &payload); err != nil {
		log.Printf("[Telemetry] Failed to unmarshal protobuf for %s: %v (body_len=%d)", vin, err, len(body))
		handleJSONTelemetry(vin, body)
		w.WriteHeader(http.StatusOK)
		return
	}

	processVehicleData(vin, &payload)
	w.WriteHeader(http.StatusOK)
}

func handleJSONTelemetry(vin string, body []byte) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("[Telemetry] Failed to parse JSON fallback for %s: %v", vin, err)
		return
	}

	media := &MediaState{
		UpdatedAt: time.Now().UnixMilli(),
	}

	if v, ok := data["MediaPlaybackStatus"].(string); ok {
		media.PlaybackStatus = v
	}
	if v, ok := data["MediaAudioSource"].(string); ok {
		media.AudioSource = v
	}
	if v, ok := data["MediaVolume"].(float64); ok {
		media.Volume = int(v)
	}
	if v, ok := data["NowPlayingTitle"].(string); ok {
		media.NowPlayingTitle = v
	}
	if v, ok := data["NowPlayingArtist"].(string); ok {
		media.NowPlayingArtist = v
	}
	if v, ok := data["NowPlayingAlbum"].(string); ok {
		media.NowPlayingAlbum = v
	}
	if v, ok := data["NowPlayingDuration"].(float64); ok {
		media.NowPlayingDuration = int(v)
	}
	if v, ok := data["NowPlayingElapsed"].(float64); ok {
		media.NowPlayingElapsed = int(v)
	}

	if media.PlaybackStatus != "" || media.NowPlayingTitle != "" {
		updateMediaState(vin, media)
	}
}

func processVehicleData(vin string, data *vehicle_data.VehicleData) {
	media := &MediaState{
		UpdatedAt: time.Now().UnixMilli(),
	}

	changed := false

	if data.MediaPlaybackStatus != nil {
		media.PlaybackStatus = data.MediaPlaybackStatus.GetValue()
		changed = true
	}
	if data.MediaAudioSource != nil {
		media.AudioSource = data.MediaAudioSource.GetValue()
		changed = true
	}
	if data.MediaVolume != nil {
		media.Volume = int(data.MediaVolume.GetValue())
		changed = true
	}
	if data.NowPlayingTitle != nil {
		media.NowPlayingTitle = data.NowPlayingTitle.GetValue()
		changed = true
	}
	if data.NowPlayingArtist != nil {
		media.NowPlayingArtist = data.NowPlayingArtist.GetValue()
		changed = true
	}
	if data.NowPlayingAlbum != nil {
		media.NowPlayingAlbum = data.NowPlayingAlbum.GetValue()
		changed = true
	}
	if data.NowPlayingDuration != nil {
		media.NowPlayingDuration = int(data.NowPlayingDuration.GetValue())
		changed = true
	}
	if data.NowPlayingElapsed != nil {
		media.NowPlayingElapsed = int(data.NowPlayingElapsed.GetValue())
		changed = true
	}

	if !changed {
		return
	}

	updateMediaState(vin, media)
}

func updateMediaState(vin string, media *MediaState) {
	mediaMu.Lock()
	latestMedia[vin] = media
	mediaMu.Unlock()

	fields := map[string]interface{}{
		"media_playback_status": media.PlaybackStatus,
		"media_audio_source":    media.AudioSource,
		"media_volume":          media.Volume,
		"now_playing_title":     media.NowPlayingTitle,
		"now_playing_artist":    media.NowPlayingArtist,
		"now_playing_album":     media.NowPlayingAlbum,
	}

	if err := redis.UpdateVehicleStateFields(vin, fields); err != nil {
		log.Printf("[Telemetry] Failed to update Redis for %s: %v", vin, err)
	}

	ws.BroadcastToVIN(vin, "media_state", fields)

	log.Printf("[Telemetry] Media update for %s: status=%q source=%q title=%q artist=%q volume=%d",
		vin, media.PlaybackStatus, media.AudioSource, media.NowPlayingTitle, media.NowPlayingArtist, media.Volume)
}

func GetLatestMedia(vin string) *MediaState {
	mediaMu.RLock()
	defer mediaMu.RUnlock()
	m, ok := latestMedia[vin]
	if !ok {
		return nil
	}
	copied := *m
	return &copied
}

func parseECPrivateKey(pemData []byte) (*ecdsa.PrivateKey, error) {
	key, err := x509.ParseECPrivateKey(pemData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse EC private key: %w", err)
	}
	return key, nil
}
