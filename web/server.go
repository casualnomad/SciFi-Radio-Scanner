package web

import (
	"encoding/json"
	"net/http"

	"Radio-Scanner/llm"
	"Radio-Scanner/radio"
)

type Server struct {
	Llm   *llm.Client
	State *radio.WorldState
}

func New(llmClient *llm.Client, state *radio.WorldState) *Server {
	return &Server{
		Llm:   llmClient,
		State: state,
	}
}

func (s *Server) HandleNext(w http.ResponseWriter, r *http.Request) {
	ch := radio.Channel(r.URL.Query().Get("channel"))

	// Init history if empty
	if len(s.State.History[ch]) == 0 {
		systemMsg := radio.ChatMessage{
			Role:    "system",
			Content: radio.InitialPrompt(ch),
		}
		s.State.History[ch] = append(s.State.History[ch], systemMsg)
	}

	// Add user prompt to history
	userMsg := radio.ChatMessage{
		Role:    "user",
		Content: radio.BuildPrompt(ch, s.State),
	}
	s.State.History[ch] = append(s.State.History[ch], userMsg)

	// Call LLM with full history
	text, err := s.Llm.Chat(toLLMMessages(s.State.History[ch]), 200)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	// Append assistant responce to history
	s.State.History[ch] = append(s.State.History[ch], radio.ChatMessage{
		Role:    "assistant",
		Content: text,
	})

	// send response
	json.NewEncoder(w).Encode(map[string]string{
		"channel": string(ch),
		"text":    text,
	})
}

func toLLMMessages(msgs []radio.ChatMessage) []llm.ChatMessage {
	out := make([]llm.ChatMessage, len(msgs))
	for i, m := range msgs {
		out[i] = llm.ChatMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}
	return out
}

func (s *Server) Routes() {
	// Serve the web UI
	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/", fs)

	//API Endpoint for next transmission
	http.HandleFunc("/next", s.HandleNext)
}
