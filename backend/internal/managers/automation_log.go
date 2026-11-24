package managers

import (
	"cmp"
	"fmt"
	"net/http"
	"strconv"

	"od/internal/models"
)

// The words of the automation logs: how a node is named

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
