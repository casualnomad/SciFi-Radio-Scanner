package radio

type Channel string

const (
	Civilian Channel = "civilian"
	Military Channel = "military"
	Anomaly  Channel = "anomaly"
)

type WorldState struct {
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
