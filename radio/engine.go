package radio

import "fmt"

// Build the initial prompt for the system
func InitialPrompt(ch Channel) string {
	switch ch {
	case Civilian:
		return "You are a space radio broadcast AI. Generate short civilian transmissions with local flavor."
	case Military:
		return "You are a military radio AI. Generate concise, alert-style transmissions."
	case Anomaly:
		return "You are a cosmic anomaly broadcast AI. Generate eerie, mysterious transmissions."
	default:
		return "You are a space radio AI"
	}
}

// Build the user prompt for the next transmission
func BuildPrompt(ch Channel, state *WorldState) string {
	switch ch {
	case Civilian:
		return fmt.Sprintf(
			"You are a space radio scanner. Generate one short civilian broadcast. Existing ships: %v",
			state.CivilianShips,
		)
	case Military:
		return fmt.Sprintf(
			"Military encrypted channel. Short radio transmission. Units: %v",
			state.MilitaryUnits,
		)
	case Anomaly:
		return fmt.Sprintf(
			"Strange cosmic anomaly broadcast. Keep it eerie. Count=%d",
			state.AnomaliesSeen,
		)
	default:
		return "Space chatter"
	}
}
