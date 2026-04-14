package websocket

import "encoding/json"

type Message struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

func (m *Message) ToJSON() []byte {
	data, _ := json.Marshal(m)
	return data
}
