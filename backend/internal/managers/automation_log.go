package managers

import (
	"cmp"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"od/internal/models"
)

// The words of the automation logs: how a node, a tag test and a decision are written

// nodeName describes a node in words for the logs, with the tag and device names:
// "Captur battery", "turn off Captur plug", "wait 300s then Captur battery"
func (manager *AutomationsManager) nodeName(node models.AutomationNode) string {
	switch node.Type {
	case models.NodeTimer:
		return "timer " + node.Cron
	case models.NodeSunrise, models.NodeSunset:
		if node.OffsetMinutes == 0 {
			return node.Type
		}
		return fmt.Sprintf("%s %+d min", node.Type, node.OffsetMinutes)
	case models.NodeTagValueFloat, models.NodeTagValueBool:
		return manager.tagName(node.TagID)
	case models.NodeCondition:
		return "if " + manager.tagName(node.TagID)
	case models.NodeWaitCheck:
		return fmt.Sprintf("wait %ds then %s", node.DelaySeconds, manager.tagName(node.TagID))
	case models.NodeAnd:
		return "AND"
	case models.NodeOr:
		return "OR"
	case models.NodeWhatsApp:
		text := []rune(node.Text)
		if len(text) > 30 {
			text = append(text[:30], '…')
		}
		return "WhatsApp " + strconv.Quote(string(text))
	case models.NodeDevice:
		return pick(node.On, "turn on ", "turn off ") + manager.deviceName(node.DeviceID)
	case models.NodeHTTP:
		return cmp.Or(node.Method, http.MethodGet) + " " + node.URL
	}
	return node.Type + " " + node.ID
}

func (manager *AutomationsManager) tagName(tagID int) string {
	if tag := manager.tags.findTag(tagID); tag != nil && tag.Name != "" {
		return tag.Name
	}
	return "tag " + strconv.Itoa(tagID)
}

func (manager *AutomationsManager) deviceName(deviceID int) string {
	if manager.devices != nil {
		for _, device := range manager.devices.Devices() {
			if fields := device.Fields(); fields.ID == deviceID && fields.Name != "" {
				return fields.Name
			}
		}
	}
	return "device " + strconv.Itoa(deviceID)
}

func pick(condition bool, yes string, no string) string {
	if condition {
		return yes
	}
	return no
}

func mark(ok bool) string {
	return pick(ok, "✓", "✗")
}

// check shows a tag test with the value it read, for the logs: "Captur battery 82 > 80 ✓",
// "Gate open is false, wants true ✗", "Captur battery has no value ✗"
func check(name string, node models.AutomationNode, value models.TagValue, ok bool) string {
	switch value := value.(type) {
	case *models.TagValueFloat:
		if value != nil {
			return name + " " + strconv.FormatFloat(value.Value, 'f', -1, 64) + " " + node.Operator + " " + strconv.FormatFloat(node.Value, 'f', -1, 64) + " " + mark(ok)
		}
	case *models.TagValueBoolean:
		if value != nil {
			return name + " is " + strconv.FormatBool(value.Value) + ", wants " + strconv.FormatBool(node.BoolValue) + " " + mark(ok)
		}
	}
	return name + " has no value " + mark(ok)
}

// step is the decision of one node in words. summary is false for the steps that only add noise to
// the info logs of a run (the triggers that did not fire).
type step struct {
	node    string // id, to find the node in the editor
	text    string
	summary bool
}

// inputTypes names the inputs of a node by their node type, each type once: "timer or and"; with
// onlyTrue only the inputs whose output is true
func inputTypes(inputs []string, types map[string]string, results map[string]nodeResult, onlyTrue bool) string {
	kinds := []string{}
	for _, input := range inputs {
		if (!onlyTrue || results[input].output) && !slices.Contains(kinds, types[input]) {
			kinds = append(kinds, types[input])
		}
	}
	if len(kinds) == 0 {
		return "input"
	}
	return strings.Join(kinds, " or ")
}

// conditionText says in words what a trigger or a gate decided: "sunset fired ✓",
// "Captur battery 82 > 80 ✓", "if Gate open is false, wants true ✗", "AND ✓"
func (manager *AutomationsManager) conditionText(node models.AutomationNode, name string, value models.TagValue, result nodeResult, inputs string) string {
	switch {
	case isEvent(node.Type):
		return name + pick(result.output, " fired ✓", " did not fire ✗")
	case watchesTag(node.Type):
		return check(name, node, value, result.output)
	case node.Type == models.NodeCondition && result.input:
		return "if " + check(manager.tagName(node.TagID), node, value, result.output)
	case node.Type == models.NodeCondition:
		return name + " ✗ (not all of its " + inputs + " inputs are true)"
	}
	return name + " " + mark(result.output) // and, or
}
