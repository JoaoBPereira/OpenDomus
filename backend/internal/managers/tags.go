package managers

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"od/internal/models"
	"od/internal/parse"
	"od/internal/postgres"
)

// TagsManager keeps the enabled tags and their last values in memory and stores the values the
// devices and the API hand it.
type TagsManager struct {
	db    *postgres.DB
	debug bool // keep the values in memory only, write nothing to the tag value tables

	mutex sync.RWMutex // Tag appends to tags on a cache miss while HTTP handlers read the list
	tags  []*models.Tag

	// onValue is told the id of a tag that got a float or boolean value (the automations), when set
	onValue func(tagID int)
}

func NewTagsManager(db *postgres.DB, debug bool) *TagsManager {
	return &TagsManager{db: db, debug: debug, tags: []*models.Tag{}}
}

// Start loads the enabled tags and their last values from the database
func (manager *TagsManager) Start(ctx context.Context) error {
	query := `
      SELECT
        t.id, t.name, t.type, t.value_type, t.value_storage, t.config, t.device_id, t.periodicity_count, t.expires_in, t.enabled, t.display_in,
        f.time AS float_time, f.value AS float_value,
        b.time AS bool_time, b.value AS bool_value,
        l.time AS loc_time, l.latitude, l.longitude, l.accuracy, l.altitude, l.speed
      FROM public.tag t
      LEFT JOIN LATERAL (
        SELECT time, value FROM tagvaluefloat WHERE tag_id = t.id ORDER BY time DESC LIMIT 1
      ) f ON t.value_type = 'tagvaluefloat'
      LEFT JOIN LATERAL (
        SELECT time, value FROM tagvalueboolean WHERE tag_id = t.id ORDER BY time DESC LIMIT 1
      ) b ON t.value_type = 'tagvalueboolean'
      LEFT JOIN LATERAL (
        SELECT time, latitude, longitude, accuracy, altitude, speed FROM tagvaluelocation WHERE tag_id = t.id ORDER BY time DESC LIMIT 1
      ) l ON t.value_type = 'tagvaluelocation'
    `
	rows, err := manager.db.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("loading tags: %w", err)
	}

	tags := []*models.Tag{}
	for _, row := range rows {
		tag := parseTagRow(row)
		if !tag.Enabled {
			slog.Info("tag disabled, skipped", "tag", tag.Name)
			continue
		}
		parseLastValueRow(tag, row)
		tags = append(tags, tag)
	}

	manager.mutex.Lock()
	manager.tags = tags
	manager.mutex.Unlock()
	slog.Info("tags loaded", "count", len(tags))
	return nil
}

func (manager *TagsManager) Stop() {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	manager.tags = []*models.Tag{}
}

// parseTagRow builds a Tag from a row of the tag table. A NULL column is the zero value.
func parseTagRow(row map[string]any) *models.Tag {
	id, _ := row["id"].(int32) // pgx returns a Postgres integer (int4) as int32
	deviceID, _ := row["device_id"].(int32)
	periodicityCount, _ := row["periodicity_count"].(int32)
	expiresIn, _ := row["expires_in"].(int32)
	tag := &models.Tag{
		ID:               int(id),
		Name:             parse.String(row["name"]),
		Type:             parse.String(row["type"]),
		ValueType:        parse.String(row["value_type"]),
		ValueStorage:     parse.String(row["value_storage"]),
		Config:           parse.String(row["config"]),
		DeviceID:         int(deviceID),
		PeriodicityCount: int(periodicityCount),
		ExpiresIn:        int(expiresIn),
		DisplayIn:        []string{},
		// only present in ReadTagsFromDB (LEFT JOIN with the device table)
		DeviceName:        parse.String(row["device_name"]),
		DevicePeriodicity: parse.String(row["device_periodicity"]),
	}
	tag.Enabled, _ = row["enabled"].(bool)

	// pgx returns a Postgres array (character varying[]) as []any
	displayIn, _ := row["display_in"].([]any)
	for _, item := range displayIn {
		if text, isString := item.(string); isString {
			tag.DisplayIn = append(tag.DisplayIn, text)
		}
	}
	return tag
}

// parseLastValueRow sets the last value of the tag from the float_*, bool_* and loc_* columns of the Start query
func parseLastValueRow(tag *models.Tag, row map[string]any) {
	switch tag.ValueType {
	case "tagvaluefloat":
		if at, found := row["float_time"].(time.Time); found {
			value, _ := row["float_value"].(float64)
			tag.SetLastValue(models.NewTagValueFloat(tag.ID, at, value))
		}
	case "tagvalueboolean":
		if at, found := row["bool_time"].(time.Time); found {
			value, _ := row["bool_value"].(bool)
			tag.SetLastValue(models.NewTagValueBoolean(tag.ID, at, value))
		}
	case "tagvaluelocation":
		if at, found := row["loc_time"].(time.Time); found {
			latitude, _ := row["latitude"].(float64)
			longitude, _ := row["longitude"].(float64)
			tag.SetLastValue(&models.TagValueLocation{
				TagID:     tag.ID,
				Latitude:  latitude,
				Longitude: longitude,
				Accuracy:  floatOrNil(row["accuracy"]),
				Altitude:  floatOrNil(row["altitude"]),
				Speed:     floatOrNil(row["speed"]),
				Time:      at,
			})
		}
	}
}

// floatOrNil keeps a NULL double precision column as nil (accuracy, altitude and speed are optional)
func floatOrNil(value any) *float64 {
	number, found := value.(float64)
	if !found {
		return nil
	}
	return &number
}

// OnTagValue sets the function told about every float and boolean value; call it before Start
func (manager *TagsManager) OnTagValue(onValue func(tagID int)) {
	manager.onValue = onValue
}

func (manager *TagsManager) Tags() []*models.Tag {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()
	return manager.tags
}

func (manager *TagsManager) TagsByDeviceID(deviceID int) []*models.Tag {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	result := []*models.Tag{}
	for _, tag := range manager.tags {
		if tag.DeviceID == deviceID {
			result = append(result, tag)
		}
	}
	return result
}

func (manager *TagsManager) findTag(tagID int) *models.Tag {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	for _, tag := range manager.tags {
		if tag.ID == tagID {
			return tag
		}
	}
	return nil
}

// Tag returns the tag from memory, or loads it from the database (and keeps it) when it is not
// there yet. A missing or disabled tag is ErrNotFound.
func (manager *TagsManager) Tag(ctx context.Context, tagID int) (*models.Tag, error) {
	if tag := manager.findTag(tagID); tag != nil {
		return tag, nil
	}

	rows, err := manager.db.Query(ctx, "SELECT id, name, type, value_type, value_storage, config, device_id, periodicity_count, expires_in, enabled FROM public.tag WHERE id = $1", tagID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	tag := parseTagRow(rows[0])
	if !tag.Enabled {
		return nil, ErrNotFound
	}

	manager.mutex.Lock()
	manager.tags = append(manager.tags, tag)
	manager.mutex.Unlock()
	slog.Info("tag loaded from the database on a cache miss", "tag", tag.Name, "id", tagID)
	return tag, nil
}

const tagColumns = "name, type, value_type, value_storage, config, device_id, periodicity_count, expires_in, enabled, display_in"

func (manager *TagsManager) CreateTagInDB(ctx context.Context, tag *models.Tag) (*models.Tag, error) {
	query := "INSERT INTO public.tag (" + tagColumns + ") VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING *"
	rows, err := manager.db.Query(ctx, query, tag.Name, tag.Type, tag.ValueType, tag.ValueStorage, tag.Config, deviceIDOrNull(tag), tag.PeriodicityCount, tag.ExpiresIn, tag.Enabled, displayInOrEmpty(tag))
	if err != nil {
		return nil, err
	}
	return parseTagRow(rows[0]), nil
}

func (manager *TagsManager) ReadTagsFromDB(ctx context.Context) ([]*models.Tag, error) {
	rows, err := manager.db.Query(ctx, "SELECT t.*, d.name AS device_name, d.periodicity AS device_periodicity FROM public.tag t LEFT JOIN public.device d ON d.id = t.device_id ORDER BY t.id")
	if err != nil {
		return nil, err
	}
	tags := make([]*models.Tag, 0, len(rows))
	for _, row := range rows {
		tags = append(tags, parseTagRow(row))
	}
	return tags, nil
}

func (manager *TagsManager) UpdateTagInDB(ctx context.Context, id int, tag *models.Tag) (*models.Tag, error) {
	query := `UPDATE public.tag
		SET name = $1, type = $2, value_type = $3, value_storage = $4, config = $5, device_id = $6, periodicity_count = $7, expires_in = $8, enabled = $9, display_in = $10
		WHERE id = $11 RETURNING *`
	rows, err := manager.db.Query(ctx, query, tag.Name, tag.Type, tag.ValueType, tag.ValueStorage, tag.Config, deviceIDOrNull(tag), tag.PeriodicityCount, tag.ExpiresIn, tag.Enabled, displayInOrEmpty(tag), id)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return parseTagRow(rows[0]), nil
}

// deviceIDOrNull stores NULL when no device id was given (the device_id column is a foreign key)
func deviceIDOrNull(tag *models.Tag) any {
	if tag.DeviceID == 0 {
		return nil
	}
	return tag.DeviceID
}

func displayInOrEmpty(tag *models.Tag) []string {
	if tag.DisplayIn == nil {
		return []string{}
	}
	return tag.DisplayIn
}

// SaveTagValueByTagID stores a value received by the API. The timestamp is optional ("" is now).
func (manager *TagsManager) SaveTagValueByTagID(ctx context.Context, tagID int, value any, timestamp string) error {
	tag, err := manager.Tag(ctx, tagID)
	if err != nil {
		return fmt.Errorf("tag %d: %w", tagID, err)
	}

	at := time.Now()
	if timestamp != "" {
		if at, err = parse.Date(timestamp); err != nil {
			return err
		}
	}
	return manager.SaveTagValue(ctx, tag, value, at)
}

// SaveTagValue converts the value to the type of the tag, stores it (unless in debug mode) and
// keeps it as the last value of the tag
func (manager *TagsManager) SaveTagValue(ctx context.Context, tag *models.Tag, value any, at time.Time) error {
	switch tag.ValueType {
	case "tagvaluefloat":
		number, err := parse.Float(value)
		if err != nil {
			return fmt.Errorf("tag %s: %w", tag.Name, err)
		}
		if !manager.debug {
			if err := manager.recordFloat(ctx, tag, number, at); err != nil {
				return err
			}
		}
		tag.SetLastValue(models.NewTagValueFloat(tag.ID, at, number))
		if manager.onValue != nil {
			manager.onValue(tag.ID)
		}
		return nil

	case "tagvalueboolean":
		boolean := parse.Bool(value)
		if !manager.debug {
			if err := manager.recordBoolean(ctx, tag, boolean, at); err != nil {
				return err
			}
		}
		tag.SetLastValue(models.NewTagValueBoolean(tag.ID, at, boolean))
		if manager.onValue != nil {
			manager.onValue(tag.ID)
		}
		return nil

	case "tagvaluelocation":
		location, isLocation := value.(*models.TagValueLocation)
		if !isLocation {
			return fmt.Errorf("tag %s: value is not a location", tag.Name)
		}
		if !manager.debug {
			if err := manager.recordLocation(ctx, tag, location, at); err != nil {
				return err
			}
		}
		last := *location
		last.TagID = tag.ID
		last.Time = at
		tag.SetLastValue(&last)
		return nil
	}
	return fmt.Errorf("tag %s: unknown value type %q", tag.Name, tag.ValueType)
}

// recordFloat writes a tagvaluefloat row: a new one (value_storage insert) or the single row of the tag (update)
func (manager *TagsManager) recordFloat(ctx context.Context, tag *models.Tag, value float64, at time.Time) error {
	switch tag.ValueStorage {
	case "insert":
		inserted, err := manager.db.Exec(ctx, "INSERT INTO tagvaluefloat (tag_id, time, value) VALUES ($1, $2, $3) ON CONFLICT (tag_id, time) DO NOTHING", tag.ID, at, value)
		if err != nil {
			return fmt.Errorf("tag %s: storing float: %w", tag.Name, err)
		}
		if inserted > 0 {
			slog.Info("float value inserted", "tag", tag.Name, "value", value, "time", at)
		}
	case "update":
		_, err := manager.db.Exec(ctx, `WITH upsert AS (
				UPDATE tagvaluefloat SET time = $2, value = $3 WHERE tag_id = $1 RETURNING *
			)
			INSERT INTO tagvaluefloat (tag_id, time, value)
			SELECT $1, $2, $3 WHERE NOT EXISTS (SELECT 1 FROM upsert)`, tag.ID, at, value)
		if err != nil {
			return fmt.Errorf("tag %s: storing float: %w", tag.Name, err)
		}
		slog.Info("float value upserted", "tag", tag.Name, "value", value, "time", at)
	}
	return nil
}

func (manager *TagsManager) recordBoolean(ctx context.Context, tag *models.Tag, value bool, at time.Time) error {
	switch tag.ValueStorage {
	case "insert":
		inserted, err := manager.db.Exec(ctx, "INSERT INTO tagvalueboolean (tag_id, time, value) VALUES ($1, $2, $3) ON CONFLICT (tag_id, time) DO NOTHING", tag.ID, at, value)
		if err != nil {
			return fmt.Errorf("tag %s: storing boolean: %w", tag.Name, err)
		}
		if inserted > 0 {
			slog.Info("boolean value inserted", "tag", tag.Name, "value", value, "time", at)
		}
	case "update":
		_, err := manager.db.Exec(ctx, `WITH upsert AS (
				UPDATE tagvalueboolean SET time = $2, value = $3 WHERE tag_id = $1 RETURNING *
			)
			INSERT INTO tagvalueboolean (tag_id, time, value)
			SELECT $1, $2, $3 WHERE NOT EXISTS (SELECT 1 FROM upsert)`, tag.ID, at, value)
		if err != nil {
			return fmt.Errorf("tag %s: storing boolean: %w", tag.Name, err)
		}
		slog.Info("boolean value upserted", "tag", tag.Name, "value", value, "time", at)
	}
	return nil
}

func (manager *TagsManager) recordLocation(ctx context.Context, tag *models.Tag, location *models.TagValueLocation, at time.Time) error {
	insert := `INSERT INTO public.tagvaluelocation (tag_id, latitude, longitude, accuracy, altitude, speed, time) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	args := []any{tag.ID, location.Latitude, location.Longitude, location.Accuracy, location.Altitude, location.Speed, at}

	switch tag.ValueStorage {
	case "insert":
		// the table has no unique constraint on (tag_id, time), so duplicates are checked here
		existing, err := manager.db.Query(ctx, "SELECT 1 FROM public.tagvaluelocation WHERE tag_id = $1 AND time = $2 LIMIT 1", tag.ID, at)
		if err != nil {
			return fmt.Errorf("tag %s: storing location: %w", tag.Name, err)
		}
		if len(existing) > 0 {
			slog.Info("location already stored, skipped", "tag", tag.Name, "time", at)
			return nil
		}
		if _, err := manager.db.Exec(ctx, insert, args...); err != nil {
			return fmt.Errorf("tag %s: storing location: %w", tag.Name, err)
		}
		slog.Info("location inserted", "tag", tag.Name, "latitude", location.Latitude, "longitude", location.Longitude)
	case "update":
		updated, err := manager.db.Exec(ctx, `UPDATE public.tagvaluelocation SET latitude = $2, longitude = $3, accuracy = $4, altitude = $5, speed = $6, time = $7 WHERE tag_id = $1`, args...)
		if err != nil {
			return fmt.Errorf("tag %s: storing location: %w", tag.Name, err)
		}
		if updated == 0 {
			if _, err := manager.db.Exec(ctx, insert, args...); err != nil {
				return fmt.Errorf("tag %s: storing location: %w", tag.Name, err)
			}
		}
		slog.Info("location upserted", "tag", tag.Name, "latitude", location.Latitude, "longitude", location.Longitude)
	}
	return nil
}

// LatestTagValue is the last value in memory of the tag, or a zero float value dated 1970 when
// the tag never had a value
func (manager *TagsManager) LatestTagValue(ctx context.Context, tagID int) (models.TagValue, error) {
	tag, err := manager.Tag(ctx, tagID)
	if err != nil {
		return nil, err
	}
	if last := tag.LastValue(); last != nil {
		return last, nil
	}
	return models.NewTagValueFloat(tagID, time.Unix(0, 0).UTC(), 0), nil
}

// LatestTagValueLocation is LatestTagValue with a zero location as the fallback
func (manager *TagsManager) LatestTagValueLocation(ctx context.Context, tagID int) (models.TagValue, error) {
	tag, err := manager.Tag(ctx, tagID)
	if err != nil {
		return nil, err
	}
	if last := tag.LastValue(); last != nil {
		return last, nil
	}
	return models.NewTagValueLocation(tagID, 0, 0, 0, 0, 0, time.Unix(0, 0).UTC()), nil
}

func (manager *TagsManager) NearestTagValueFloat(ctx context.Context, tagID int, timestamp string) (*models.TagValueFloat, error) {
	var rows []models.TagValueFloat
	err := manager.db.QueryInto(ctx, &rows, "SELECT tag_id AS tagId, time, value FROM tagvaluefloat WHERE tag_id = $1 ORDER BY abs(extract(epoch from (time - $2::timestamptz))) LIMIT 1", tagID, timestamp)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return &rows[0], nil
}

func (manager *TagsManager) LastTagValueFloat(ctx context.Context, tagID int, limit int) ([]models.TagValueFloat, error) {
	var rows []models.TagValueFloat
	err := manager.db.QueryInto(ctx, &rows, "SELECT tag_id AS tagId, time, value FROM tagvaluefloat WHERE tag_id = $1 ORDER BY time DESC LIMIT $2", tagID, limit)
	return rows, err
}

func (manager *TagsManager) LastTagValueBoolean(ctx context.Context, tagID int, limit int) ([]models.TagValueBoolean, error) {
	var rows []models.TagValueBoolean
	err := manager.db.QueryInto(ctx, &rows, "SELECT tag_id AS tagId, time, value FROM tagvalueboolean WHERE tag_id = $1 ORDER BY time DESC LIMIT $2", tagID, limit)
	return rows, err
}

func (manager *TagsManager) LastTagValueLocation(ctx context.Context, tagID int, limit int) ([]models.TagValueLocation, error) {
	var rows []models.TagValueLocation
	err := manager.db.QueryInto(ctx, &rows, "SELECT tag_id AS tagId, latitude, longitude, accuracy, altitude, speed, time FROM tagvaluelocation WHERE tag_id = $1 ORDER BY time DESC LIMIT $2", tagID, limit)
	return rows, err
}

func (manager *TagsManager) TagValueFloat(ctx context.Context, tagID int, begin string, end string) ([]models.TagValueFloat, error) {
	var rows []models.TagValueFloat
	err := manager.db.QueryInto(ctx, &rows, "SELECT tag_id AS tagId, time, value FROM tagvaluefloat WHERE tag_id = $1 AND time BETWEEN $2 AND $3 ORDER BY time ASC", tagID, begin, end)
	return rows, err
}

func (manager *TagsManager) TagValueFloatTimeSeries(ctx context.Context, tagID int, begin string, end string, intervalMinutes int) ([]models.TagValueFloatTimeSeries, error) {
	query := `SELECT
			time_bucket(make_interval(mins => $4), "time") AS bucket,
			AVG(value) AS avg_value, MAX(value) AS max_value, MIN(value) AS min_value
		FROM public.tagvaluefloat
		WHERE tag_id = $1 AND "time" BETWEEN $2 AND $3
		GROUP BY bucket ORDER BY bucket DESC`
	var rows []models.TagValueFloatTimeSeries
	err := manager.db.QueryInto(ctx, &rows, query, tagID, begin, end, intervalMinutes)
	return rows, err
}

// PowerConsumption sums the energy counted by a total power tag between two timestamps, taking
// the counter resets into account
func (manager *TagsManager) PowerConsumption(ctx context.Context, tagID int, begin string, end string) (float64, error) {
	var deviceTypes []struct {
		Type *string `db:"type"`
	}
	err := manager.db.QueryInto(ctx, &deviceTypes, "SELECT d.type FROM public.tag t JOIN public.device d ON d.id = t.device_id WHERE t.id = $1", tagID)
	if err != nil {
		return 0, err
	}
	deviceType := ""
	if len(deviceTypes) > 0 && deviceTypes[0].Type != nil {
		deviceType = *deviceTypes[0].Type
	}

	rows, err := manager.TagValueFloat(ctx, tagID, begin, end)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	values := make([]float64, len(rows))
	for index, row := range rows {
		values[index] = row.Value
	}
	return powerConsumption(values, deviceType), nil
}

// powerConsumption walks the counter readings: a drop of more than 0.99 is a counter reset
func powerConsumption(values []float64, deviceType string) float64 {
	last := len(values) - 1
	switch deviceType {
	case "NOUS A1T":
		// keep each reading before a reset and the first reading after it, as pairs
		pairs := []float64{values[0]}
		for index := 0; index < last; index++ {
			if values[index] > values[index+1]+0.99 {
				pairs = append(pairs, values[index], values[index+1])
			}
		}
		pairs = append(pairs, values[last])
		sum := 0.0
		for index := 0; index+1 < len(pairs); index += 2 {
			sum += pairs[index+1] - pairs[index]
		}
		return sum
	default:
		// keep the first reading, each reading before a reset, and the last reading
		kept := []float64{values[0]}
		for index := 0; index < last; index++ {
			if values[index] > values[index+1]+0.99 {
				kept = append(kept, values[index])
			}
		}
		kept = append(kept, values[last])
		if len(kept) < 2 {
			return 0
		}
		sum := kept[1] - kept[0]
		for index := 2; index < len(kept); index++ {
			sum += kept[index]
		}
		return sum
	}
}

// PowerConsumptionNearTimestamp returns, for every currentPower tag, its value closest to the
// timestamp within 5 minutes
func (manager *TagsManager) PowerConsumptionNearTimestamp(ctx context.Context, timestamp string) ([]models.PowerConsumptionNearTimestamp, error) {
	query := `WITH candidates AS (
			SELECT tag_id AS tagId, time, value, abs(extract(epoch from (time - $2::timestamp))) AS diff_seconds
			FROM tagvaluefloat
			WHERE tag_id = $1 AND time BETWEEN ($2::timestamp - INTERVAL '5 minutes') AND ($2::timestamp + INTERVAL '5 minutes')
		),
		min_diff AS (SELECT MIN(diff_seconds) AS min_diff FROM candidates)
		SELECT c.* FROM candidates c JOIN min_diff m ON c.diff_seconds = m.min_diff ORDER BY c.time ASC`

	results := []models.PowerConsumptionNearTimestamp{}
	for _, tag := range manager.Tags() {
		if tag.Type != "currentPower" {
			continue
		}
		rows, err := manager.db.Query(ctx, query, tag.ID, timestamp)
		if err != nil {
			return nil, err
		}
		if len(rows) == 1 {
			value, _ := rows[0]["value"].(float64)
			at, _ := rows[0]["time"].(time.Time)
			results = append(results, models.PowerConsumptionNearTimestamp{Tag: tag, Value: value, Time: at})
		}
	}
	return results, nil
}

// DeleteExpiredTagValues deletes from tagvaluefloat and tagvalueboolean the values older than
// the expires_in days of their tag (tags with expires_in = 0 keep everything)
func (manager *TagsManager) DeleteExpiredTagValues(ctx context.Context) error {
	for _, table := range []string{"tagvaluefloat", "tagvalueboolean"} {
		deleted, err := manager.db.Exec(ctx, "DELETE FROM "+table+" v USING (SELECT id, expires_in FROM public.tag WHERE expires_in > 0) t WHERE v.tag_id = t.id AND v.time < NOW() - (t.expires_in * INTERVAL '1 day')")
		if err != nil {
			return fmt.Errorf("deleting expired values of %s: %w", table, err)
		}
		slog.Info("expired tag values deleted", "table", table, "rows", deleted)
	}
	return nil
}

// TagValueFloatPage returns a page of tagvaluefloat rows, newest first, with an optional tag filter.
// count is capped to 10000.
func (manager *TagsManager) TagValueFloatPage(ctx context.Context, offset int, count int, tagID *int) (*models.PaginatedResult, error) {
	count = max(1, min(count, 10000))
	offset = max(0, offset)

	where := ""
	filters := []any{}
	if tagID != nil {
		filters = append(filters, *tagID)
		where = "WHERE tag_id = $1"
	}

	var counts []postgres.CountRow
	if err := manager.db.QueryInto(ctx, &counts, "SELECT COUNT(*) AS total FROM public.tagvaluefloat "+where, filters...); err != nil {
		return nil, err
	}
	total := 0
	if len(counts) > 0 {
		total = counts[0].Total
	}

	var rows []models.TagValueFloat
	query := fmt.Sprintf("SELECT tag_id AS tagId, time, value FROM public.tagvaluefloat %s ORDER BY time DESC LIMIT $%d OFFSET $%d", where, len(filters)+1, len(filters)+2)
	if err := manager.db.QueryInto(ctx, &rows, query, append(filters, count, offset)...); err != nil {
		return nil, err
	}

	return &models.PaginatedResult{
		Rows: rows,
		Meta: models.PaginationMeta{
			Total:  total,
			Offset: offset,
			Count:  count,
			Pages:  max(0, int(math.Ceil(float64(total-offset)/float64(count)))),
		},
	}, nil
}
