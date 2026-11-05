package managers

import (
	"fmt"
	"net/url"

	"od/internal/models"
	"od/internal/schedule"
)

// isEvent is true for the triggers that fire at a moment (true only when they are the one firing)
func isEvent(nodeType string) bool {
	return nodeType == models.NodeTimer || nodeType == models.NodeSunrise || nodeType == models.NodeSunset
}

func isTrigger(nodeType string) bool {
	return isEvent(nodeType) || nodeType == models.NodeTagValueFloat || nodeType == models.NodeTagValueBool
}

// matches compares the last value of a tag with the node: a float with Operator and Value, a
// boolean with BoolValue; no value does not match
func matches(node models.AutomationNode, value models.TagValue) bool {
	switch value := value.(type) {
	case *models.TagValueFloat:
		if value == nil {
			return false
		}
		result, _ := compare(node.Operator, value.Value, node.Value)
		return result
	case *models.TagValueBoolean:
		return value != nil && value.Value == node.BoolValue
	}
	return false
}

func isAction(nodeType string) bool {
	return nodeType == models.NodeWhatsApp || nodeType == models.NodeDevice || nodeType == models.NodeHTTP
}

// validateGraph rejects what the engine cannot run: unknown types, missing fields, dangling or
// backwards edges (into a trigger, out of an action), unconnected gates/actions and cycles
func validateGraph(graph models.AutomationGraph) error {
	nodes := map[string]models.AutomationNode{}
	triggers := 0
	for _, node := range graph.Nodes {
		if node.ID == "" {
			return fmt.Errorf("a node has no id")
		}
		if _, duplicate := nodes[node.ID]; duplicate {
			return fmt.Errorf("node %q: duplicate id", node.ID)
		}
		nodes[node.ID] = node
		if err := validateNode(node); err != nil {
			return fmt.Errorf("node %q: %w", node.ID, err)
		}
		if isTrigger(node.Type) {
			triggers++
		}
	}
	if triggers == 0 {
		return fmt.Errorf("the automation needs at least one trigger")
	}

	inputs := inputsByNode(graph)
	for _, edge := range graph.Edges {
		from, fromFound := nodes[edge.From]
		to, toFound := nodes[edge.To]
		if !fromFound || !toFound {
			return fmt.Errorf("edge %s -> %s: unknown node", edge.From, edge.To)
		}
		if isAction(from.Type) {
			return fmt.Errorf("edge %s -> %s: an action has no output", edge.From, edge.To)
		}
		if isTrigger(to.Type) {
			return fmt.Errorf("edge %s -> %s: a trigger has no input", edge.From, edge.To)
		}
	}
	for _, node := range graph.Nodes {
		if !isTrigger(node.Type) && len(inputs[node.ID]) == 0 {
			return fmt.Errorf("node %q: has no input", node.ID)
		}
	}

	// cycle check: depth first, a node met again while still on the path closes a cycle
	const visiting, done = 1, 2
	state := map[string]int{}
	var visit func(id string) error
	visit = func(id string) error {
		switch state[id] {
		case visiting:
			return fmt.Errorf("node %q: the edges form a cycle", id)
		case done:
			return nil
		}
		state[id] = visiting
		for _, input := range inputs[id] {
			if err := visit(input); err != nil {
				return err
			}
		}
		state[id] = done
		return nil
	}
	for _, node := range graph.Nodes {
		if err := visit(node.ID); err != nil {
			return err
		}
	}
	return nil
}

func validateNode(node models.AutomationNode) error {
	switch node.Type {
	case models.NodeTimer:
		return schedule.Validate(node.Cron)
	case models.NodeSunrise, models.NodeSunset:
		if node.OffsetMinutes < -720 || node.OffsetMinutes > 720 {
			return fmt.Errorf("offsetMinutes must be between -720 and 720")
		}
	case models.NodeTagValueBool:
		if node.TagID <= 0 {
			return fmt.Errorf("tagId is required")
		}
	case models.NodeWaitCheck:
		if node.DelaySeconds < 1 || node.DelaySeconds > 86400 {
			return fmt.Errorf("delaySeconds must be between 1 and 86400")
		}
		fallthrough
	case models.NodeTagValueFloat, models.NodeCondition:
		if node.TagID <= 0 {
			return fmt.Errorf("tagId is required")
		}
		if _, err := compare(node.Operator, 0, 0); err != nil {
			return err
		}
	case models.NodeAnd, models.NodeOr:
	case models.NodeWhatsApp:
		if node.Text == "" {
			return fmt.Errorf("text is required")
		}
	case models.NodeDevice:
		if node.DeviceID <= 0 {
			return fmt.Errorf("deviceId is required")
		}
	case models.NodeHTTP:
		parsed, err := url.Parse(node.URL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return fmt.Errorf("url must be an http(s) url")
		}
	default:
		return fmt.Errorf("unknown type %q", node.Type)
	}
	return nil
}

func compare(operator string, value float64, threshold float64) (bool, error) {
	switch operator {
	case ">":
		return value > threshold, nil
	case "<":
		return value < threshold, nil
	case "=":
		return value == threshold, nil
	case ">=":
		return value >= threshold, nil
	case "<=":
		return value <= threshold, nil
	}
	return false, fmt.Errorf("operator must be one of > < = >= <=, not %q", operator)
}

func inputsByNode(graph models.AutomationGraph) map[string][]string {
	inputs := map[string][]string{}
	for _, edge := range graph.Edges {
		inputs[edge.To] = append(inputs[edge.To], edge.From)
	}
	return inputs
}

// evaluateGraph returns the output of every node of a validated graph. A timer, sunrise or sunset
// is true only when it is the one firing (firedTimer); a tagValueFloat/tagValueBoolean matches the last
// value of its tag (false when the tag has no value); an action is true when one of its inputs is.
func evaluateGraph(graph models.AutomationGraph, firedTimer string, lastValue func(tagID int) models.TagValue) map[string]bool {
	nodes := map[string]models.AutomationNode{}
	for _, node := range graph.Nodes {
		nodes[node.ID] = node
	}
	inputs := inputsByNode(graph)
	outputs := map[string]bool{}

	var output func(id string) bool
	output = func(id string) bool {
		if result, found := outputs[id]; found {
			return result
		}
		node := nodes[id]
		result := false
		switch {
		case isEvent(node.Type):
			result = id == firedTimer
		case node.Type == models.NodeTagValueFloat || node.Type == models.NodeTagValueBool:
			result = matches(node, lastValue(node.TagID))
		case node.Type == models.NodeWaitCheck:
			// true only on its own delayed check (firedTimer), see AutomationsManager.evaluate
			result = id == firedTimer && matches(node, lastValue(node.TagID))
		case node.Type == models.NodeAnd || node.Type == models.NodeCondition:
			result = true
			for _, input := range inputs[id] {
				result = output(input) && result
			}
			// a condition only reads the tag: unlike tagValueFloat, new values do not evaluate the graph
			if node.Type == models.NodeCondition && result {
				result = matches(node, lastValue(node.TagID))
			}
		default: // or, and the actions
			for _, input := range inputs[id] {
				result = output(input) || result
			}
		}
		outputs[id] = result
		return result
	}
	for id := range nodes {
		output(id)
	}
	return outputs
}
