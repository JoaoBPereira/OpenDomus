package models

import "encoding/json"

// Automation is a row of the automation table: a flow of trigger nodes, logic gates and actions,
// stored as a graph so a node editor (n8n style) can draw and save it as is
type Automation struct {
	ID      int             `json:"id"`
	Name    string          `json:"name"`
	Enabled bool            `json:"enabled"`
	Graph   AutomationGraph `json:"graph"`
}

type AutomationGraph struct {
	Nodes []AutomationNode `json:"nodes"`
	Edges []AutomationEdge `json:"edges"`
}

// Node types
const (
	NodeTimer         = "timer"           // trigger: fires on Cron
	NodeSunrise       = "sunrise"         // trigger: fires every day at sunrise plus OffsetMinutes
	NodeSunset        = "sunset"          // trigger: fires every day at sunset plus OffsetMinutes
	NodeTagValueFloat = "tagValueFloat"   // trigger: true while the last value of TagID <Operator> Value
	NodeTagValueBool  = "tagValueBoolean" // trigger: true while the last value of TagID is BoolValue
	NodeAnd           = "and"             // gate: true when every input is true
	NodeOr            = "or"              // gate: true when one input is true
	NodeCondition     = "condition"       // gate: true when every input is true and the last value of TagID matches
	NodeWaitCheck     = "waitCheck"       // gate: DelaySeconds after its input becomes true, true once if the last value of TagID matches
	NodeWhatsApp      = "whatsapp"        // action: sends Text
	NodeDevice        = "device"          // action: turns DeviceID on or off (On)
	NodeHTTP          = "http"            // action: sends Method URL with Body and Headers
)

// AutomationNode holds the fields of every node type; each type reads only its own fields
type AutomationNode struct {
	ID   string `json:"id"`
	Type string `json:"type"`

	Cron string `json:"cron,omitempty"`

	OffsetMinutes int `json:"offsetMinutes,omitempty"` // sunrise/sunset: minutes after (negative: before)

	TagID    int     `json:"tagId,omitempty"`    // tagValueFloat and condition
	Operator string  `json:"operator,omitempty"` // >, <, =, >=, <=
	Value    float64 `json:"value"`

	// boolean tags (tagValueBoolean, and condition/waitCheck on a boolean tag) match BoolValue
	BoolValue bool `json:"boolValue"`

	DelaySeconds int `json:"delaySeconds,omitempty"` // waitCheck

	Text string `json:"text,omitempty"`

	DeviceID int  `json:"deviceId,omitempty"`
	On       bool `json:"on"`

	URL     string            `json:"url,omitempty"`
	Method  string            `json:"method,omitempty"`
	Body    string            `json:"body,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`

	// UI is kept untouched for the editor (position, labels...)
	UI json.RawMessage `json:"ui,omitempty"`
}

// AutomationEdge connects the output of From to an input of To
type AutomationEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}
