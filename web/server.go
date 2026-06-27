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
	if !ch.Valid() {
		http.Error(w, "unknown channel: "+string(ch), http.StatusBadRequest)
		return
	}

	// Build the message list to send, mutating shared state under the lock.
	s.State.Mu.Lock()

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

	// Snapshot the history so the (potentially slow) LLM call happens
	// without holding the lock.
	messages := toLLMMessages(s.State.History[ch])
	s.State.Mu.Unlock()

	// Call LLM with full history
	text, err := s.Llm.Chat(messages, 200)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Append assistant response to history
	s.State.Mu.Lock()
	s.State.History[ch] = append(s.State.History[ch], radio.ChatMessage{
		Role:    "assistant",
		Content: text,
	})
	s.State.Mu.Unlock()

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
