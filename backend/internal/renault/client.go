// Package renault reads a car through the (unofficial) MyRenault API.
package renault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"od/internal/config"
	"od/internal/httpclient"
	"od/internal/parse"
)

// Public keys of the MyRenault app (not account specific). Renault rotates them from time to time:
// current values are kept in https://github.com/hacf-fr/renault-api/blob/main/src/renault_api/const.py
// and can be overridden with RENAULT_GIGYA_API_KEY / RENAULT_KAMEREON_API_KEY.
const gigyaURL = "https://accounts.eu1.gigya.com"
const gigyaAPIKeyDefault = "3_VgdkgtIRH3AdHvJm-cjV2ug2EFE0lxt0IJzMC4MFqZjFpn_GYFXVdNZ19L7wZX0N"
const kamereonURL = "https://api-wired-prod-1-euw1.wrd-aws.com"
const kamereonAPIKeyDefault = "YjkKtHmGfaceeuExUDKGxrLZGGvtVS0J"

const gigyaPendingTwoFactorAuthentication = 403101
const jwtExpirationSeconds = 900

// EndpointVersions lists the Kamereon car-adapter endpoints and the API version each one uses
var EndpointVersions = map[string]int{
	"battery-status":    2,
	"charge-mode":       1,
	"charging-settings": 1,
	"cockpit":           1,
	"hvac-status":       1,
	"location":          1,
	"lock-status":       1,
	"res-state":         1,
}

// The car-adapter answers are kept as parsed JSON (map[string]any). The fields of interest:
//   battery-status: timestamp, batteryLevel (%), batteryAutonomy (km), batteryTemperature, batteryCapacity,
//                   batteryAvailableEnergy (kWh), plugStatus (0 unplugged, 1 plugged), chargingStatus
//                   (1.0 charging, 0.x not charging), chargingRemainingTime (min), chargingInstantaneousPower
//   cockpit:        totalMileage (km), fuelAutonomy (km), fuelQuantity (l)
//   location:       gpsLatitude, gpsLongitude, lastUpdateTime

type Vehicle struct {
	VIN                string `json:"vin"`
	Brand              string `json:"brand,omitempty"`
	Model              string `json:"model,omitempty"`
	RegistrationNumber string `json:"registrationNumber,omitempty"`
}

// GigyaError is returned when Gigya answers HTTP 200 with an errorCode in the body
type GigyaError struct {
	Code    int
	Message string
}

func (gigyaError *GigyaError) Error() string {
	return gigyaError.Message
}

// Client only needs the MyRenault username and password: personID, accountID and VINs are discovered.
//
// Auth chain: Gigya login -> login_token -> personID + JWT (15 min) -> Kamereon calls
type Client struct {
	cfg  config.Renault
	http *httpclient.Client

	mutex        sync.Mutex // the session fields below are refreshed from the polling goroutine
	loginToken   string
	jwt          string
	jwtExpiresAt time.Time
	personID     string
	accountID    string
}

func NewClient(cfg config.Renault, http *httpclient.Client) *Client {
	if cfg.Country == "" {
		cfg.Country = "PT"
	}
	if cfg.GigyaAPIKey == "" {
		cfg.GigyaAPIKey = gigyaAPIKeyDefault
	}
	if cfg.KamereonAPIKey == "" {
		cfg.KamereonAPIKey = kamereonAPIKeyDefault
	}
	return &Client{cfg: cfg, http: http}
}

func (client *Client) Vehicles(ctx context.Context) ([]Vehicle, error) {
	accountID, err := client.getAccountID(ctx)
	if err != nil {
		return nil, err
	}
	data, err := client.kamereonGet(ctx, kamereonURL+"/commerce/v1/accounts/"+accountID+"/vehicles", true)
	if err != nil {
		return nil, err
	}

	vehicleLinks, _ := data["vehicleLinks"].([]any)
	vehicles := []Vehicle{}
	for _, item := range vehicleLinks {
		link, _ := item.(map[string]any)
		brand := parse.String(link["brand"])
		if brand == "" {
			brand = parse.String(path(link, "vehicleDetails", "brand", "label"))
		}
		vehicles = append(vehicles, Vehicle{
			VIN:                parse.String(link["vin"]),
			Brand:              brand,
			Model:              parse.String(path(link, "vehicleDetails", "model", "label")),
			RegistrationNumber: parse.String(path(link, "vehicleDetails", "registrationNumber")),
		})
	}
	return vehicles, nil
}

// CarData returns the data.attributes object of a car-adapter endpoint (see EndpointVersions)
func (client *Client) CarData(ctx context.Context, vin string, endpoint string) (map[string]any, error) {
	version, known := EndpointVersions[endpoint]
	if !known {
		return nil, fmt.Errorf("unknown endpoint %q", endpoint)
	}
	accountID, err := client.getAccountID(ctx)
	if err != nil {
		return nil, err
	}
	requestURL := fmt.Sprintf("%s/commerce/v1/accounts/%s/kamereon/kca/car-adapter/v%d/cars/%s/%s", kamereonURL, accountID, version, vin, endpoint)
	data, err := client.kamereonGet(ctx, requestURL, true)
	if err != nil {
		return nil, err
	}
	attributes, _ := path(data, "data", "attributes").(map[string]any)
	if attributes == nil {
		attributes = map[string]any{}
	}
	return attributes, nil
}

// ---------------- Kamereon ----------------

func (client *Client) kamereonGet(ctx context.Context, requestURL string, retry bool) (map[string]any, error) {
	jwt, err := client.getJWT(ctx)
	if err != nil {
		return nil, err
	}

	separator := "?"
	if strings.Contains(requestURL, "?") {
		separator = "&"
	}
	headers := map[string]string{
		"Content-Type":     "application/vnd.api+json",
		"apikey":           client.cfg.KamereonAPIKey,
		"x-gigya-id_token": jwt,
	}
	body, status, err := client.http.Do(ctx, "GET", requestURL+separator+"country="+url.QueryEscape(client.cfg.Country), "", headers)
	if err != nil {
		// Token rejected: drop the session and try once more with a fresh login
		if retry && (status == 401 || status == 403) {
			slog.Warn("renault: token rejected, re-authenticating", "status", status, "url", requestURL)
			client.mutex.Lock()
			client.loginToken = ""
			client.jwt = ""
			client.mutex.Unlock()
			return client.kamereonGet(ctx, requestURL, false)
		}
		if status != 0 {
			return nil, fmt.Errorf("Kamereon GET %s failed (%d): %s", requestURL, status, string(body))
		}
		return nil, fmt.Errorf("Kamereon GET %s failed: %w", requestURL, err)
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("Kamereon GET %s returned invalid JSON: %w", requestURL, err)
	}
	return data, nil
}

func (client *Client) getAccountID(ctx context.Context) (string, error) {
	client.mutex.Lock()
	accountID := client.accountID
	client.mutex.Unlock()
	if accountID != "" {
		return accountID, nil
	}

	if _, err := client.getJWT(ctx); err != nil { // ensures personID
		return "", err
	}
	person, err := client.kamereonGet(ctx, kamereonURL+"/commerce/v1/persons/"+client.personID, true)
	if err != nil {
		return "", err
	}
	accounts, _ := person["accounts"].([]any)

	// Prefer an active MYRENAULT account, otherwise any MYRENAULT account
	var account map[string]any
	for _, item := range accounts {
		candidate, _ := item.(map[string]any)
		if parse.String(candidate["accountType"]) == "MYRENAULT" && parse.String(candidate["accountStatus"]) != "INACTIVE" {
			account = candidate
			break
		}
	}
	if account == nil {
		for _, item := range accounts {
			candidate, _ := item.(map[string]any)
			if parse.String(candidate["accountType"]) == "MYRENAULT" {
				account = candidate
				break
			}
		}
	}
	if account == nil {
		return "", errors.New("no MYRENAULT account found for this user")
	}

	client.mutex.Lock()
	client.accountID = parse.String(account["accountId"])
	client.mutex.Unlock()
	return client.accountID, nil
}

// ---------------- Gigya ----------------

func (client *Client) getJWT(ctx context.Context) (string, error) {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	// Renew 60s before expiry
	if client.jwt != "" && time.Now().Before(client.jwtExpiresAt.Add(-60*time.Second)) {
		return client.jwt, nil
	}

	if client.loginToken == "" {
		if err := client.login(ctx); err != nil {
			return "", err
		}
	}

	jwt, err := client.requestJWT(ctx)
	if err != nil {
		// login_token may have expired: force a new login next time
		client.loginToken = ""
		client.jwt = ""
		return "", err
	}
	return jwt, nil
}

func (client *Client) requestJWT(ctx context.Context) (string, error) {
	if client.personID == "" {
		info, err := client.gigyaPost(ctx, "accounts.getAccountInfo", map[string]string{"login_token": client.loginToken})
		if err != nil {
			return "", err
		}
		client.personID = parse.String(path(info, "data", "personId"))
		if client.personID == "" {
			return "", errors.New("personId not found in Gigya account info")
		}
	}

	response, err := client.gigyaPost(ctx, "accounts.getJWT", map[string]string{
		"login_token": client.loginToken,
		"fields":      "data.personId,data.gigyaDataCenter",
		"expiration":  strconv.Itoa(jwtExpirationSeconds),
	})
	if err != nil {
		return "", err
	}
	client.jwt = parse.String(response["id_token"])
	client.jwtExpiresAt = time.Now().Add(jwtExpirationSeconds * time.Second)
	return client.jwt, nil
}

func (client *Client) login(ctx context.Context) error {
	if client.cfg.Username == "" || client.cfg.Password == "" {
		return errors.New("username and password are required")
	}

	response, err := client.gigyaPost(ctx, "accounts.login", map[string]string{"loginID": client.cfg.Username, "password": client.cfg.Password})
	if err != nil {
		var gigyaError *GigyaError
		if errors.As(err, &gigyaError) && gigyaError.Code == gigyaPendingTwoFactorAuthentication {
			return errors.New("Renault account requires two-factor authentication (email code), which is not supported yet")
		}
		return err
	}
	client.loginToken = parse.String(path(response, "sessionInfo", "cookieValue"))
	if client.loginToken == "" {
		return errors.New("login_token not found in Gigya login response")
	}
	slog.Info("renault: logged in")
	return nil
}

// gigyaPost calls a Gigya method. Gigya returns HTTP 200 with an errorCode in the body.
func (client *Client) gigyaPost(ctx context.Context, method string, params map[string]string) (map[string]any, error) {
	form := url.Values{}
	form.Set("ApiKey", client.cfg.GigyaAPIKey)
	for name, value := range params {
		form.Set(name, value)
	}
	body, _, err := client.http.Post(ctx, gigyaURL+"/"+method, form.Encode(), map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return nil, err
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("Gigya %s returned invalid JSON: %w", method, err)
	}
	errorCode, _ := parse.Int(data["errorCode"])
	if errorCode != 0 {
		message := strings.TrimSpace(fmt.Sprintf("Gigya %s error %d: %s %s", method, errorCode, parse.String(data["errorMessage"]), parse.String(data["errorDetails"])))
		return nil, &GigyaError{Code: errorCode, Message: message}
	}
	return data, nil
}

// path walks nested JSON objects: path(data, "a", "b") is data.a.b, or nil when a step is missing
func path(data any, keys ...string) any {
	current := data
	for _, key := range keys {
		object, isObject := current.(map[string]any)
		if !isObject {
			return nil
		}
		current = object[key]
	}
	return current
}
