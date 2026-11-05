// Package managers holds the application services: they own the state loaded from the database
// (tags, devices) and the scheduled jobs, and are the only callers of the postgres package.
package managers

import "errors"

// ErrNotFound is returned when a row does not exist; the REST handlers map it to 404
var ErrNotFound = errors.New("not found")

// ErrInvalidCredentials is returned by UsersManager.Login for an unknown user or a wrong password
var ErrInvalidCredentials = errors.New("invalid credentials")
