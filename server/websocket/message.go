package websocket

import "encoding/json"

type Message struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

func (m *Message) ToJSON() []byte {
	data, _ := json.Marshal(m)
	return data
}
