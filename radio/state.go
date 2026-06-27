package radio

import "sync"

type Channel string

const (
	Civilian Channel = "civilian"
	Military Channel = "military"
	Anomaly  Channel = "anomaly"
)

// Valid reports whether ch is a known channel.
func (ch Channel) Valid() bool {
	switch ch {
	case Civilian, Military, Anomaly:
		return true
	default:
		return false
	}
}

type WorldState struct {
	// Mu guards all mutable fields below; take it for any read or write
	// of the world state from a request handler.
	Mu sync.Mutex

	CivilianShips []string
	MilitaryUnits []string
	AnomaliesSeen int

	//keep per-channel chat history
	History map[Channel][]ChatMessage
}

type ChatMessage struct {
	Role    string
	Content string
}

func NewWorld() *WorldState {
	return &WorldState{
		CivilianShips: []string{"Silver Needle", "Wren Courier-9"},
		MilitaryUnits: []string{"New World Patrol"},
		AnomaliesSeen: 0,
		History: map[Channel][]ChatMessage{
			Civilian: {},
			Military: {},
			Anomaly:  {},
		},
	}
}
