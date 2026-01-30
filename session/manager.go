package session

import (
	"strings"
	"sync"
)

// Phase represents the current conversation phase
type Phase int

const (
	PhaseSetup Phase = iota
	PhaseSimulation
	PhaseDebrief
	PhasePaused
)

func (p Phase) String() string {
	switch p {
	case PhaseSetup:
		return "setup"
	case PhaseSimulation:
		return "simulation"
	case PhaseDebrief:
		return "debrief"
	case PhasePaused:
		return "paused"
	default:
		return "unknown"
	}
}

// Difficulty levels
type Difficulty string

const (
	DifficultyEasy      Difficulty = "easy"
	DifficultyRealistic Difficulty = "realistic"
	DifficultyHostile   Difficulty = "hostile"
)

// Manager handles session state
type Manager struct {
	mu            sync.RWMutex
	phase         Phase
	previousPhase Phase
	difficulty    Difficulty
	character     string
	scenario      string
}

// NewManager creates a new session manager
func NewManager() *Manager {
	return &Manager{
		phase: PhaseSetup,
	}
}

// GetPhase returns the current phase
func (m *Manager) GetPhase() Phase {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.phase
}

// SetPhase sets the current phase
func (m *Manager) SetPhase(phase Phase) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.previousPhase = m.phase
	m.phase = phase
}

// Pause pauses the simulation
func (m *Manager) Pause() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.phase == PhaseSimulation {
		m.previousPhase = m.phase
		m.phase = PhasePaused
	}
}

// Resume resumes from pause
func (m *Manager) Resume() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.phase == PhasePaused {
		m.phase = m.previousPhase
	}
}

// SetDifficulty sets the difficulty level
func (m *Manager) SetDifficulty(d Difficulty) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.difficulty = d
}

// GetDifficulty returns the difficulty level
func (m *Manager) GetDifficulty() Difficulty {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.difficulty
}

// SetScenario sets the scenario details
func (m *Manager) SetScenario(character, scenario string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.character = character
	m.scenario = scenario
}

// GetState returns the current state as a map
func (m *Manager) GetState() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return map[string]interface{}{
		"phase":      m.phase.String(),
		"difficulty": string(m.difficulty),
		"character":  m.character,
		"scenario":   m.scenario,
	}
}

// ProcessCommand checks for control commands in text
// Returns true if a command was processed
func (m *Manager) ProcessCommand(text string) (command string, handled bool) {
	upperText := strings.ToUpper(strings.TrimSpace(text))

	if strings.Contains(upperText, "PAUSE") {
		m.Pause()
		return "pause", true
	}

	if strings.Contains(upperText, "END SIMULATION") {
		m.SetPhase(PhaseDebrief)
		return "end", true
	}

	return "", false
}

// Reset resets the session to initial state
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.phase = PhaseSetup
	m.previousPhase = PhaseSetup
	m.difficulty = ""
	m.character = ""
	m.scenario = ""
}
