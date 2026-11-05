package devices

import (
	"encoding/json"
	"log/slog"
	"net"
	"time"

	"od/internal/models"
)

// YeeDevice turns a Yeelight bulb on or off: the bulbs listen on TCP port 55443 for JSON commands
// ended by \r\n.
type YeeDevice struct {
	models.DeviceFields
}

func NewYeeDevice(host string) *YeeDevice {
	return &YeeDevice{DeviceFields: models.DeviceFields{Type: "YeeDevice", URL: host, Enabled: true}}
}

func (device *YeeDevice) Stop() {}

func (device *YeeDevice) On() {
	device.setPower("on")
}

func (device *YeeDevice) Off() {
	device.setPower("off")
}

// setPower sends the command in the background, holding the connection open for 10 seconds
// like the yeelight-platform package did
func (device *YeeDevice) setPower(state string) {
	go func() {
		connection, err := net.DialTimeout("tcp", net.JoinHostPort(device.URL, "55443"), 10*time.Second)
		if err != nil {
			slog.Error("yeelight: connection failed", "host", device.URL, "state", state, "err", err)
			return
		}
		defer connection.Close()

		command, _ := json.Marshal(map[string]any{
			"id":     -1,
			"method": "set_power",
			"params": []any{state, "smooth", 300},
		})
		if _, err := connection.Write(append(command, '\r', '\n')); err != nil {
			slog.Error("yeelight: sending the command failed", "host", device.URL, "state", state, "err", err)
			return
		}
		slog.Info("yeelight: power set", "host", device.URL, "state", state)
		time.Sleep(10 * time.Second)
	}()
}
