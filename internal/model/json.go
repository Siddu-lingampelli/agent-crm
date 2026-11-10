package model

import "encoding/json"

// Expose JSON values as structured data rather than encoded JSON strings.
func (w Webhook) MarshalJSON() ([]byte, error) {
	type alias Webhook
	events := []string{}
	if err := json.Unmarshal([]byte(w.Events), &events); err != nil {
		events = []string{}
	}
	return json.Marshal(struct {
		alias
		Events []string `json:"events"`
	}{alias(w), events})
}
func (i Integration) MarshalJSON() ([]byte, error) {
	type alias Integration
	config := map[string]any{}
	if err := json.Unmarshal([]byte(i.Config), &config); err != nil {
		config = map[string]any{}
	}
	return json.Marshal(struct {
		alias
		Config map[string]any `json:"config"`
	}{alias(i), config})
}
