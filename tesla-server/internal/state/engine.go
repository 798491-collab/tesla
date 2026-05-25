package state

import (
	"sync"
	"time"
)

const (
	historyWindowSize   = 10
	stateCooldownSec    = 30
	onlineThresholdSec  = 60
	asleepThresholdSec  = 600
	minConfirmCount     = 2
	highConfidenceMin   = 0.8
	mediumConfidenceMin = 0.5
)

type OnlineState string

const (
	OnlineStateOnline  OnlineState = "online"
	OnlineStateAsleep  OnlineState = "asleep"
	OnlineStateOffline OnlineState = "offline"
)

type DriveStateType string

const (
	DriveStateParked    DriveStateType = "parked"
	DriveStateDriving   DriveStateType = "driving"
	DriveStateReversing DriveStateType = "reversing"
)

type ChargeStateType string

const (
	ChargeStateDisconnected  ChargeStateType = "disconnected"
	ChargeStateCharging      ChargeStateType = "charging"
	ChargeStateComplete      ChargeStateType = "complete"
	ChargeStateSupercharging ChargeStateType = "supercharging"
)

type CommandStateType string

const (
	CommandStateIdle     CommandStateType = "idle"
	CommandStateSending  CommandStateType = "sending"
	CommandStateSuccess  CommandStateType = "success"
	CommandStateFailed   CommandStateType = "failed"
	CommandStateTimeout  CommandStateType = "timeout"
	CommandStateRejected CommandStateType = "rejected"
)

type OnlineStateOutput struct {
	OnlineState OnlineState `json:"online_state"`
	Confidence  float64     `json:"confidence"`
	ChangedAt   int64       `json:"changed_at"`
}

type DriveStateOutput struct {
	DriveState DriveStateType `json:"drive_state"`
	Speed      float64        `json:"speed"`
	Gear       string         `json:"gear"`
}

type LockStateOutput struct {
	LockState string `json:"lock_state"`
	DoorsOpen bool   `json:"doors_open"`
}

type ChargeStateOutput struct {
	ChargeState           ChargeStateType `json:"charge_state"`
	BatteryLevel          int             `json:"battery_level"`
	ChargingPower         float64         `json:"charging_power"`
	ChargingTimeRemaining int             `json:"charging_time_remaining"`
}

type CommandStateOutput struct {
	CommandState CommandStateType `json:"command_state"`
	LastCommand  string           `json:"last_command"`
	LatencyMs    int64            `json:"latency_ms"`
}

type MetaOutput struct {
	LastSuccessAt        int64  `json:"last_success_at"`
	LastFailAt           int64  `json:"last_fail_at"`
	StateLockUntil       int64  `json:"state_lock_until"`
	StateTransitionCount int    `json:"state_transition_count"`
	LastStateSource      string `json:"last_state_source"`
}

type VehicleStateOutput struct {
	VIN     string             `json:"vin"`
	State   OnlineStateOutput  `json:"state"`
	Drive   DriveStateOutput   `json:"drive"`
	Lock    LockStateOutput    `json:"lock"`
	Charge  ChargeStateOutput  `json:"charge"`
	Command CommandStateOutput `json:"command"`
	Meta    MetaOutput         `json:"meta"`
}

type VehicleDataInput struct {
	Speed          float64
	Gear           string
	ChargingState  string
	Supercharging  bool
	Soc            int
	ChargePower    float64
	MinutesToFull  int
	Locked         bool
	DoorOpen       bool
}

type stateHistory struct {
	mu                  sync.Mutex
	vin                 string
	onlineState         OnlineState
	stateChangedAt      time.Time
	transitionCount     int
	lastSuccessAt       time.Time
	lastFailAt          time.Time
	stateLockUntil      time.Time
	lastStateSource     string
	successHistory      [historyWindowSize]bool
	historyIndex        int
	historyCount        int
	pendingState        *OnlineState
	pendingSince        time.Time
	pendingConfirmCount int
	commandState        CommandStateType
	lastCommand         string
	commandLatencyMs    int64
	commandStartedAt    time.Time
}

var (
	engines   = make(map[string]*stateHistory)
	enginesMu sync.RWMutex
)

func getEngine(vin string) *stateHistory {
	enginesMu.Lock()
	defer enginesMu.Unlock()

	if e, ok := engines[vin]; ok {
		return e
	}

	e := &stateHistory{
		vin:             vin,
		onlineState:     OnlineStateOffline,
		stateChangedAt:  time.Now(),
		lastStateSource: "init",
		commandState:    CommandStateIdle,
	}
	engines[vin] = e
	return e
}

func recordSuccess(e *stateHistory) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.successHistory[e.historyIndex] = true
	e.historyIndex = (e.historyIndex + 1) % historyWindowSize
	if e.historyCount < historyWindowSize {
		e.historyCount++
	}
	e.lastSuccessAt = time.Now()
}

func recordFailure(e *stateHistory) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.successHistory[e.historyIndex] = false
	e.historyIndex = (e.historyIndex + 1) % historyWindowSize
	if e.historyCount < historyWindowSize {
		e.historyCount++
	}
	e.lastFailAt = time.Now()
}

func calcConfidence(e *stateHistory) float64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return calcConfidenceUnlocked(e)
}

func deriveOnlineState(lastSuccessAt time.Time, successRate float64) OnlineState {
	elapsed := time.Since(lastSuccessAt)

	if elapsed <= onlineThresholdSec*time.Second && successRate >= 0.6 {
		return OnlineStateOnline
	}
	if elapsed <= asleepThresholdSec*time.Second {
		return OnlineStateAsleep
	}
	return OnlineStateOffline
}

func (e *stateHistory) proposeState(newState OnlineState, source string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if time.Now().Before(e.stateLockUntil) {
		return
	}

	if newState == e.onlineState {
		e.pendingState = nil
		e.pendingConfirmCount = 0
		return
	}

	if e.pendingState != nil && *e.pendingState == newState {
		e.pendingConfirmCount++
		if e.pendingConfirmCount >= minConfirmCount {
			if time.Since(e.stateChangedAt) >= stateCooldownSec*time.Second || e.onlineState == OnlineStateOffline {
				e.onlineState = newState
				e.stateChangedAt = time.Now()
				e.transitionCount++
				e.stateLockUntil = time.Now().Add(stateCooldownSec * time.Second)
				e.pendingState = nil
				e.pendingConfirmCount = 0
				e.lastStateSource = source
			}
		}
	} else {
		pending := newState
		e.pendingState = &pending
		e.pendingSince = time.Now()
		e.pendingConfirmCount = 1
		e.lastStateSource = source
	}
}

func deriveDriveState(input *VehicleDataInput) DriveStateType {
	if input.Speed > 5 {
		return DriveStateDriving
	}
	if input.Gear == "R" {
		return DriveStateReversing
	}
	return DriveStateParked
}

func deriveChargeState(input *VehicleDataInput) ChargeStateType {
	if input.ChargingState != "Charging" {
		switch input.ChargingState {
		case "Complete":
			return ChargeStateComplete
		case "Disconnected", "NoPower", "":
			return ChargeStateDisconnected
		default:
			return ChargeStateDisconnected
		}
	}
	if input.Supercharging {
		return ChargeStateSupercharging
	}
	return ChargeStateCharging
}

func deriveLockState(input *VehicleDataInput) string {
	if input.Locked {
		return "locked"
	}
	return "unlocked"
}

func UpdateFromFullData(vin string, input *VehicleDataInput, source string) *VehicleStateOutput {
	e := getEngine(vin)
	recordSuccess(e)

	onlineState := deriveOnlineState(e.lastSuccessAt, calcConfidence(e))
	e.proposeState(onlineState, source)

	return buildOutput(e, input)
}

func UpdateFromLightweight(vin string, apiState string, online bool, source string) *VehicleStateOutput {
	e := getEngine(vin)

	if online {
		recordSuccess(e)
	} else {
		recordFailure(e)
	}

	onlineState := deriveOnlineState(e.lastSuccessAt, calcConfidence(e))
	e.proposeState(onlineState, source)

	return buildOutputLightweight(e, apiState, online)
}

func RecordCommandStart(vin string, command string) {
	e := getEngine(vin)
	e.mu.Lock()
	defer e.mu.Unlock()
	e.commandState = CommandStateSending
	e.lastCommand = command
	e.commandStartedAt = time.Now()
}

func RecordCommandResult(vin string, success bool) {
	e := getEngine(vin)
	e.mu.Lock()
	defer e.mu.Unlock()
	e.commandLatencyMs = time.Since(e.commandStartedAt).Milliseconds()
	if success {
		e.commandState = CommandStateSuccess
	} else {
		e.commandState = CommandStateFailed
	}
}

func GetOutput(vin string, input *VehicleDataInput) *VehicleStateOutput {
	e := getEngine(vin)
	return buildOutput(e, input)
}

func buildOutput(e *stateHistory, input *VehicleDataInput) *VehicleStateOutput {
	e.mu.Lock()
	defer e.mu.Unlock()

	return &VehicleStateOutput{
		VIN: e.vin,
		State: OnlineStateOutput{
			OnlineState: e.onlineState,
			Confidence:  calcConfidenceUnlocked(e),
			ChangedAt:   e.stateChangedAt.Unix(),
		},
		Drive: DriveStateOutput{
			DriveState: deriveDriveState(input),
			Speed:      input.Speed,
			Gear:       input.Gear,
		},
		Lock: LockStateOutput{
			LockState: deriveLockState(input),
			DoorsOpen: input.DoorOpen,
		},
		Charge: ChargeStateOutput{
			ChargeState:           deriveChargeState(input),
			BatteryLevel:          input.Soc,
			ChargingPower:         input.ChargePower,
			ChargingTimeRemaining: input.MinutesToFull,
		},
		Command: CommandStateOutput{
			CommandState: e.commandState,
			LastCommand:  e.lastCommand,
			LatencyMs:    e.commandLatencyMs,
		},
		Meta: MetaOutput{
			LastSuccessAt:        e.lastSuccessAt.Unix(),
			LastFailAt:           e.lastFailAt.Unix(),
			StateLockUntil:       e.stateLockUntil.Unix(),
			StateTransitionCount: e.transitionCount,
			LastStateSource:      e.lastStateSource,
		},
	}
}

func buildOutputLightweight(e *stateHistory, apiState string, online bool) *VehicleStateOutput {
	e.mu.Lock()
	defer e.mu.Unlock()

	return &VehicleStateOutput{
		VIN: e.vin,
		State: OnlineStateOutput{
			OnlineState: e.onlineState,
			Confidence:  calcConfidenceUnlocked(e),
			ChangedAt:   e.stateChangedAt.Unix(),
		},
		Drive: DriveStateOutput{
			DriveState: DriveStateParked,
			Speed:      0,
			Gear:       "P",
		},
		Lock: LockStateOutput{
			LockState: "locked",
			DoorsOpen: false,
		},
		Charge: ChargeStateOutput{
			ChargeState:  ChargeStateDisconnected,
			BatteryLevel: 0,
		},
		Command: CommandStateOutput{
			CommandState: e.commandState,
			LastCommand:  e.lastCommand,
			LatencyMs:    e.commandLatencyMs,
		},
		Meta: MetaOutput{
			LastSuccessAt:        e.lastSuccessAt.Unix(),
			LastFailAt:           e.lastFailAt.Unix(),
			StateLockUntil:       e.stateLockUntil.Unix(),
			StateTransitionCount: e.transitionCount,
			LastStateSource:      e.lastStateSource,
		},
	}
}

func calcConfidenceUnlocked(e *stateHistory) float64 {
	if e.historyCount == 0 {
		return 0.5
	}

	successCount := 0
	for i := 0; i < e.historyCount; i++ {
		if e.successHistory[i] {
			successCount++
		}
	}

	ratio := float64(successCount) / float64(e.historyCount)

	if time.Since(e.stateChangedAt) < 30*time.Second {
		ratio *= 0.9
	}

	if e.transitionCount > 5 {
		ratio *= 0.85
	}

	if ratio > 1.0 {
		ratio = 1.0
	}
	if ratio < 0 {
		ratio = 0
	}
	return ratio
}

func IsOnline(state OnlineState) bool {
	return state != OnlineStateOffline
}

func CanControl(state OnlineState) bool {
	return state == OnlineStateOnline
}

func CleanupEngine(vin string) {
	enginesMu.Lock()
	defer enginesMu.Unlock()
	delete(engines, vin)
}
