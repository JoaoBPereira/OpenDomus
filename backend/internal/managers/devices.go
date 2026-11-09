package managers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"od/internal/config"
	"od/internal/devices"
	"od/internal/httpclient"
	"od/internal/models"
	"od/internal/postgres"
	"od/internal/renault"
)

// DevicesManager builds the devices of the device table and keeps them running
type DevicesManager struct {
	cfg  *config.Config
	db   *postgres.DB
	deps devices.Deps

	mutex   sync.RWMutex // devices is replaced by Start/Stop while handlers and jobs read it
	devices []models.Device
}

func NewDevicesManager(cfg *config.Config, db *postgres.DB, tags *TagsManager, http *httpclient.Client) *DevicesManager {
	return &DevicesManager{cfg: cfg, db: db, deps: devices.Deps{Tags: tags, HTTP: http}, devices: []models.Device{}}
}

// Start builds one device per row of the device table and starts their polling. A row whose
// device cannot be built (unknown type, invalid periodicity) is logged and skipped.
func (manager *DevicesManager) Start(ctx context.Context) error {
	rows, err := manager.ReadAllDevicesFromDB(ctx)
	if err != nil {
		return fmt.Errorf("loading devices: %w", err)
	}

	list := []models.Device{}
	for _, fields := range rows {
		// checked before building: constructors start polling right away
		if !fields.Enabled {
			slog.Warn("device skipped because it is disabled", "id", fields.ID, "name", fields.Name, "type", fields.Type)
			continue
		}

		device, err := manager.newDevice(fields)
		if err != nil {
			slog.Warn("device skipped", "id", fields.ID, "name", fields.Name, "type", fields.Type, "err", err)
			continue
		}
		list = append(list, device)
	}

	manager.mutex.Lock()
	manager.devices = list
	manager.mutex.Unlock()
	slog.Info("devices started", "count", len(list), "rows", len(rows))
	return nil
}

func (manager *DevicesManager) Stop() {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	for _, device := range manager.devices {
		device.Stop()
	}
	manager.devices = []models.Device{}
}

func (manager *DevicesManager) Devices() []models.Device {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()
	return manager.devices
}

// PowerDevice returns the device with the given id when it is a power device, or nil
func (manager *DevicesManager) PowerDevice(deviceID int) models.PowerDevice {
	for _, device := range manager.Devices() {
		if device.Fields().ID == deviceID {
			powerDevice, _ := device.(models.PowerDevice)
			return powerDevice
		}
	}
	return nil
}

// build turns the typed result of a constructor into the Device interface, keeping a nil error in step with a nil device
func build[D models.Device](device D, err error) (models.Device, error) {
	if err != nil {
		return nil, err
	}
	return device, nil
}

func (manager *DevicesManager) newDevice(fields models.DeviceFields) (models.Device, error) {
	deps := manager.deps
	switch fields.Type {
		case "Shelly 1":
			return build(devices.NewShelly1PM(deps, fields))
		case "Shelly EM":
			return build(devices.NewShellyEM(deps, fields))
		case "Shelly Plug S":
			return build(devices.NewShellyPlugS(deps, fields))
		case "Shelly Plus Plug S", "Shelly Gen3 Plug S":
			return build(devices.NewShellyPlusGen3PlugS(deps, fields))
		case "Shelly Plug M Gen3":
			return build(devices.NewShellyPlugMGen3(deps, fields))
		case "NOUS A1T":
			return build(devices.NewNousA1T(deps, fields))
		case "EcoForest":
			return build(devices.NewEcoForest(deps, fields, manager.cfg.EcoForestAuth))
		case "Rohan Air":
			return build(devices.NewRohanAir(deps, fields, manager.cfg.RohanAirTuyaID, manager.cfg.RohanAirTuyaKey))
		case "SNZB-02D":
			return devices.NewSNZB02D(deps, fields), nil
		case "Renault":
			return build(devices.NewRenault(deps, fields, renault.NewClient(manager.cfg.Renault, deps.HTTP)))
		case "":
			return nil, errors.New("device has no type")
	}
	return nil, fmt.Errorf("unknown device type %q", fields.Type)
}

// deviceColumns reads the nullable columns as empty values, so the rows scan into DeviceFields
const deviceColumns = "id, COALESCE(type, '') AS type, COALESCE(url, '') AS url, COALESCE(name, '') AS name, COALESCE(periodicity, '') AS periodicity, COALESCE(enabled, false) AS enabled, COALESCE(config, '') AS config"

func (manager *DevicesManager) ReadAllDevicesFromDB(ctx context.Context) ([]models.DeviceFields, error) {
	var rows []models.DeviceFields
	err := manager.db.QueryInto(ctx, &rows, "SELECT "+deviceColumns+" FROM public.device ORDER BY id")
	return rows, err
}

func (manager *DevicesManager) CreateDeviceInDB(ctx context.Context, fields *models.DeviceFields) (*models.DeviceFields, error) {
	var rows []models.DeviceFields
	err := manager.db.QueryInto(ctx, &rows, "INSERT INTO public.device (type, url, name, periodicity, enabled, config) VALUES ($1, $2, $3, $4, $5, $6) RETURNING "+deviceColumns,
		fields.Type, fields.URL, fields.Name, fields.Periodicity, fields.Enabled, fields.Config)
	if err != nil {
		return nil, err
	}
	return &rows[0], nil
}

func (manager *DevicesManager) UpdateDeviceInDB(ctx context.Context, id int, fields *models.DeviceFields) (*models.DeviceFields, error) {
	var rows []models.DeviceFields
	err := manager.db.QueryInto(ctx, &rows, "UPDATE public.device SET type = $1, url = $2, name = $3, periodicity = $4, enabled = $5, config = $6 WHERE id = $7 RETURNING "+deviceColumns,
		fields.Type, fields.URL, fields.Name, fields.Periodicity, fields.Enabled, fields.Config, id)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return &rows[0], nil
}
