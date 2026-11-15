// Package oauthscopes holds the single source of truth for the CRM scope
// vocabulary so both the auth package and the OAuth metadata can share it
// without an import cycle.
package oauthscopes

import "strings"

// all is the complete scope vocabulary for the single-owner release.
const all = "search:read companies:read companies:write contacts:read contacts:write leads:read leads:write deals:read deals:write tasks:read tasks:write projects:read projects:write emails:read emails:write emails:send calendar:read calendar:write workflows:read workflows:write workflows:execute integrations:read integrations:write webhooks:read webhooks:write audit:read audit:write agents:read agents:write dashboard:read settings:read settings:write admin:read admin:write"

// All returns the full scope vocabulary as a slice.
func All() []string { return strings.Fields(all) }
