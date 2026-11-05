package hub

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"time"
)

type Reading struct {
	Timestamp time.Time      `json:"timestamp"`
	Source    string         `json:"source"`
	Tags      map[string]any `json:"tags"`
}

type Automation struct {
	ID       string `json:"id"`
	Tag      string `json:"tag"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
	Message  string `json:"message"`
}

type Event struct {
	AutomationID string    `json:"automationId"`
	Message      string    `json:"message"`
	Timestamp    time.Time `json:"timestamp"`
}

type TagValue struct {
	Name      string    `json:"name"`
	Value     any       `json:"value"`
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
}

type persistedState struct {
	Readings    []Reading    `json:"readings"`
	Automations []Automation `json:"automations"`
	Events      []Event      `json:"events"`
}

type Store struct {
	mu          sync.RWMutex
	path        string
	readings    []Reading
	automations []Automation
	events      []Event
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path}
	if path == "" {
		return s, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
	}
	var state persistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decode state: %w", err)
	}
	s.readings = state.Readings
	s.automations = state.Automations
	s.events = state.Events
	return s, nil
}

func (s *Store) AddReading(reading Reading) ([]Event, error) {
	if reading.Source == "" || len(reading.Tags) == 0 || reading.Timestamp.IsZero() {
		return nil, errors.New("source, timestamp, and at least one tag are required")
	}
	for name := range reading.Tags {
		if name == "" {
			return nil, errors.New("tag names cannot be empty")
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.readings = append(s.readings, reading)
	var triggered []Event
	for _, automation := range s.automations {
		value, ok := reading.Tags[automation.Tag]
		if ok && matches(value, automation.Operator, automation.Value) {
			event := Event{
				AutomationID: automation.ID,
				Message:      automation.Message,
				Timestamp:    reading.Timestamp,
			}
			triggered = append(triggered, event)
			s.events = append(s.events, event)
		}
	}
	if err := s.persistLocked(); err != nil {
		s.readings = s.readings[:len(s.readings)-1]
		if len(triggered) > 0 {
			s.events = s.events[:len(s.events)-len(triggered)]
		}
		return nil, err
	}
	return triggered, nil
}

func (s *Store) AddAutomation(automation Automation) error {
	if automation.ID == "" || automation.Tag == "" || automation.Message == "" {
		return errors.New("id, tag, and message are required")
	}
	switch automation.Operator {
	case "eq", "gt", "lt", "gte", "lte":
	default:
		return errors.New("operator must be eq, gt, lt, gte, or lte")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.automations {
		if existing.ID == automation.ID {
			return errors.New("automation id already exists")
		}
	}
	s.automations = append(s.automations, automation)
	if err := s.persistLocked(); err != nil {
		s.automations = s.automations[:len(s.automations)-1]
		return err
	}
	return nil
}

func (s *Store) Readings(tag string, limit int) []Reading {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []Reading
	for _, reading := range s.readings {
		if tag == "" {
			result = append(result, reading)
		} else if _, ok := reading.Tags[tag]; ok {
			result = append(result, reading)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})
	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result
}

func (s *Store) Tags() []TagValue {
	s.mu.RLock()
	defer s.mu.RUnlock()
	latest := make(map[string]TagValue)
	for _, reading := range s.readings {
		for name, value := range reading.Tags {
			current, exists := latest[name]
			if !exists || reading.Timestamp.After(current.Timestamp) {
				latest[name] = TagValue{
					Name: name, Value: value, Source: reading.Source, Timestamp: reading.Timestamp,
				}
			}
		}
	}
	result := make([]TagValue, 0, len(latest))
	for _, tag := range latest {
		result = append(result, tag)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (s *Store) Automations() []Automation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Automation(nil), s.automations...)
}

func (s *Store) Events() []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Event(nil), s.events...)
}

func (s *Store) persistLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	data, err := json.Marshal(persistedState{
		Readings: s.readings, Automations: s.automations, Events: s.events,
	})
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".opendomus-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary state: %w", err)
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("set state permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write state: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close state: %w", err)
	}
	if err := os.Rename(tmp.Name(), s.path); err != nil {
		return fmt.Errorf("replace state: %w", err)
	}
	return nil
}

func matches(actual any, operator string, expected any) bool {
	if operator == "eq" {
		return reflect.DeepEqual(actual, expected)
	}
	left, leftOK := number(actual)
	right, rightOK := number(expected)
	if !leftOK || !rightOK {
		return false
	}
	switch operator {
	case "gt":
		return left > right
	case "lt":
		return left < right
	case "gte":
		return left >= right
	case "lte":
		return left <= right
	default:
		return false
	}
}

func number(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		n, err := typed.Float64()
		return n, err == nil
	case string:
		n, err := strconv.ParseFloat(typed, 64)
		return n, err == nil
	default:
		return 0, false
	}
}
