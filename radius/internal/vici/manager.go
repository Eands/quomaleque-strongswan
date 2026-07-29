package vici

import (
	"context"
	"fmt"
	"os"

	"github.com/strongswan/govici/vici"
)

const defaultViciSocket = "/var/run/charon.vici"

type ViciManager struct {
	session *vici.Session
}

func NewViciManager() (*ViciManager, error) {
	socketPath := os.Getenv("VICI_SOCKET")
	if socketPath == "" {
		socketPath = defaultViciSocket
	}

	session, err := vici.NewSession(vici.WithSocketPath(socketPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create VICI session: %w", err)
	}

	return &ViciManager{
		session: session,
	}, nil
}

func (m *ViciManager) Close() {
	if m.session != nil {
		m.session.Close()
	}
}

func (m *ViciManager) GetVersion() (map[string]string, error) {
	msg, err := m.session.Call(context.Background(), "version", nil)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, key := range msg.Keys() {
		result[key] = msg.Get(key).(string)
	}
	return result, nil
}

func (m *ViciManager) GetConnections() ([]string, error) {
	msg, err := m.session.Call(context.Background(), "get-conns", nil)
	if err != nil {
		return nil, err
	}

	conns := msg.Get("conns")
	if conns == nil {
		return []string{}, nil
	}

	if connsSlice, ok := conns.([]string); ok {
		return connsSlice, nil
	}
	if connsInterface, ok := conns.([]interface{}); ok {
		result := make([]string, len(connsInterface))
		for i, v := range connsInterface {
			if s, ok := v.(string); ok {
				result[i] = s
			}
		}
		return result, nil
	}

	return []string{}, nil
}

func (m *ViciManager) GetSAs() ([]map[string]interface{}, error) {
	msg, err := m.session.Call(context.Background(), "list-sas", nil)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, key := range msg.Keys() {
		val := msg.Get(key)
		if m, ok := val.(map[string]interface{}); ok {
			m["name"] = key
			result = append(result, m)
		}
	}
	return result, nil
}

func (m *ViciManager) LoadConnection(config map[string]interface{}) error {
	msg := vici.NewMessage()
	for k, v := range config {
		if err := msg.Set(k, v); err != nil {
			return err
		}
	}

	_, err := m.session.Call(context.Background(), "load-conn", msg)
	return err
}

func (m *ViciManager) UnloadConnection(name string) error {
	msg := vici.NewMessage()
	if err := msg.Set("name", name); err != nil {
		return err
	}

	_, err := m.session.Call(context.Background(), "unload-conn", msg)
	return err
}

func (m *ViciManager) InitiateConnection(name string) error {
	msg := vici.NewMessage()
	if err := msg.Set("name", name); err != nil {
		return err
	}

	_, err := m.session.Call(context.Background(), "initiate", msg)
	return err
}

func (m *ViciManager) TerminateConnection(name string) error {
	msg := vici.NewMessage()
	if err := msg.Set("name", name); err != nil {
		return err
	}

	_, err := m.session.Call(context.Background(), "terminate", msg)
	return err
}

func (m *ViciManager) GetStats() (map[string]interface{}, error) {
	msg, err := m.session.Call(context.Background(), "stats", nil)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for _, key := range msg.Keys() {
		result[key] = msg.Get(key)
	}
	return result, nil
}

func (m *ViciManager) MonitorEvents(handler func(event string, data map[string]interface{})) error {
	events := make(chan vici.Event, 100)

	err := m.session.Subscribe("all")
	if err != nil {
		return err
	}

	m.session.NotifyEvents(events)

	go func() {
		for event := range events {
			data := make(map[string]interface{})
			for _, key := range event.Message.Keys() {
				data[key] = event.Message.Get(key)
			}
			handler(event.Name, data)
		}
	}()

	return nil
}
