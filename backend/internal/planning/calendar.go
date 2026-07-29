package planning

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	googleAuthorizationEndpoint = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenEndpoint         = "https://oauth2.googleapis.com/token"
	googleCalendarBaseURL       = "https://www.googleapis.com/calendar/v3"
)

var calendarScopes = []string{
	"https://www.googleapis.com/auth/calendar.events.readonly",
	"https://www.googleapis.com/auth/calendar.calendarlist.readonly",
}

// TokenCipher protects stored refresh tokens and PKCE verifiers with AES-256-GCM.
type TokenCipher struct{ aead cipher.AEAD }

func NewTokenCipher(encodedKey string) (*TokenCipher, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encodedKey))
	if err != nil {
		key, err = base64.RawStdEncoding.DecodeString(strings.TrimSpace(encodedKey))
	}
	if err != nil {
		return nil, fmt.Errorf("decode Calendar token encryption key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("Calendar token encryption key must decode to 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create Calendar token cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create Calendar token cipher mode: %w", err)
	}
	return &TokenCipher{aead: aead}, nil
}

func (c *TokenCipher) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate encryption nonce: %w", err)
	}
	return c.aead.Seal(nonce, nonce, plaintext, nil), nil
}

func (c *TokenCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < c.aead.NonceSize() {
		return nil, errors.New("invalid encrypted token")
	}
	return c.aead.Open(nil, ciphertext[:c.aead.NonceSize()], ciphertext[c.aead.NonceSize():], nil)
}

// CalendarOAuthConfig contains runtime-only OAuth credentials.
type CalendarOAuthConfig struct{ ClientID, ClientSecret, RedirectURL string }

// GoogleCalendarClient implements OAuth, source discovery, and normalized planning reads.
type GoogleCalendarClient struct {
	store  *Store
	cipher *TokenCipher
	config CalendarOAuthConfig
	client *http.Client
}

func NewGoogleCalendarClient(store *Store, cipher *TokenCipher, config CalendarOAuthConfig) *GoogleCalendarClient {
	return &GoogleCalendarClient{store: store, cipher: cipher, config: config, client: &http.Client{Timeout: 25 * time.Second}}
}

func (c *GoogleCalendarClient) configured() error {
	if c.cipher == nil || strings.TrimSpace(c.config.ClientID) == "" || strings.TrimSpace(c.config.ClientSecret) == "" || strings.TrimSpace(c.config.RedirectURL) == "" {
		return fmt.Errorf("Google Calendar OAuth is not configured")
	}
	return nil
}

// AuthorizationURL creates a single-use, ten-minute OAuth state and PKCE challenge.
func (c *GoogleCalendarClient) AuthorizationURL(ctx context.Context, userID string) (string, error) {
	if err := c.configured(); err != nil {
		return "", err
	}
	state, err := randomURLValue(32)
	if err != nil {
		return "", err
	}
	verifier, err := randomURLValue(48)
	if err != nil {
		return "", err
	}
	verifierCiphertext, err := c.cipher.Encrypt([]byte(verifier))
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(state))
	if err := c.store.CreateGoogleOAuthState(ctx, userID, hash[:], verifierCiphertext, time.Now().UTC().Add(10*time.Minute)); err != nil {
		return "", err
	}
	challengeHash := sha256.Sum256([]byte(verifier))
	query := url.Values{
		"client_id": {c.config.ClientID}, "redirect_uri": {c.config.RedirectURL}, "response_type": {"code"},
		"scope": {strings.Join(calendarScopes, " ")}, "access_type": {"offline"}, "prompt": {"consent"},
		"state": {state}, "code_challenge": {base64.RawURLEncoding.EncodeToString(challengeHash[:])}, "code_challenge_method": {"S256"},
	}
	return googleAuthorizationEndpoint + "?" + query.Encode(), nil
}

// CompleteAuthorization exchanges a callback code, stores the encrypted refresh token, and discovers calendars.
func (c *GoogleCalendarClient) CompleteAuthorization(ctx context.Context, userID, state, code string) error {
	if err := c.configured(); err != nil {
		return err
	}
	if strings.TrimSpace(code) == "" {
		return fmt.Errorf("Google authorization did not return a code")
	}
	hash := sha256.Sum256([]byte(strings.TrimSpace(state)))
	encryptedVerifier, err := c.store.ConsumeGoogleOAuthState(ctx, userID, hash[:], time.Now().UTC())
	if err != nil {
		return err
	}
	verifier, err := c.cipher.Decrypt(encryptedVerifier)
	if err != nil {
		return fmt.Errorf("decrypt OAuth PKCE verifier: %w", err)
	}
	token, err := c.exchangeCode(ctx, code, string(verifier))
	if err != nil {
		return err
	}
	refreshToken := token.RefreshToken
	if refreshToken == "" {
		_, existing, existingErr := c.store.LoadCalendarConnection(ctx, userID)
		if existingErr != nil {
			return fmt.Errorf("Google did not return a refresh token; reconnect with consent")
		}
		stored, decryptErr := c.cipher.Decrypt(existing)
		if decryptErr != nil {
			return fmt.Errorf("decrypt existing Google refresh token: %w", decryptErr)
		}
		refreshToken = string(stored)
	}
	encryptedToken, err := c.cipher.Encrypt([]byte(refreshToken))
	if err != nil {
		return err
	}
	if _, err := c.store.UpsertCalendarConnection(ctx, userID, encryptedToken, calendarScopes); err != nil {
		return err
	}
	sources, err := c.discoverSources(ctx, token.AccessToken)
	if err != nil {
		return err
	}
	return c.store.ReplaceCalendarSources(ctx, userID, sources)
}

func (c *GoogleCalendarClient) PlanningEvents(ctx context.Context, userID string, planDate time.Time, location *time.Location) ([]CalendarEvent, error) {
	if err := c.configured(); err != nil {
		return nil, err
	}
	_, encrypted, err := c.store.LoadCalendarConnection(ctx, userID)
	if err != nil {
		return nil, err
	}
	refreshToken, err := c.cipher.Decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt Google refresh token: %w", err)
	}
	token, err := c.refreshToken(ctx, string(refreshToken))
	if err != nil {
		return nil, err
	}
	sources, err := c.store.ListCalendarSources(ctx, userID)
	if err != nil {
		return nil, err
	}
	var events []CalendarEvent
	for _, source := range sources {
		if !source.Enabled {
			continue
		}
		items, err := c.listEvents(ctx, token.AccessToken, source, planDate, location)
		if err != nil {
			return nil, err
		}
		events = append(events, items...)
	}
	return events, nil
}

func (c *GoogleCalendarClient) exchangeCode(ctx context.Context, code, verifier string) (googleTokenResponse, error) {
	values := url.Values{"code": {code}, "client_id": {c.config.ClientID}, "client_secret": {c.config.ClientSecret}, "redirect_uri": {c.config.RedirectURL}, "grant_type": {"authorization_code"}, "code_verifier": {verifier}}
	return c.requestToken(ctx, values)
}

func (c *GoogleCalendarClient) refreshToken(ctx context.Context, refreshToken string) (googleTokenResponse, error) {
	values := url.Values{"refresh_token": {refreshToken}, "client_id": {c.config.ClientID}, "client_secret": {c.config.ClientSecret}, "grant_type": {"refresh_token"}}
	return c.requestToken(ctx, values)
}

func (c *GoogleCalendarClient) requestToken(ctx context.Context, values url.Values) (googleTokenResponse, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenEndpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return googleTokenResponse{}, fmt.Errorf("create Google token request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := c.client.Do(request)
	if err != nil {
		return googleTokenResponse{}, fmt.Errorf("send Google token request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return googleTokenResponse{}, fmt.Errorf("Google token request returned %s", response.Status)
	}
	var token googleTokenResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 256<<10)).Decode(&token); err != nil {
		return googleTokenResponse{}, fmt.Errorf("decode Google token response: %w", err)
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return googleTokenResponse{}, fmt.Errorf("Google token response did not contain an access token")
	}
	return token, nil
}

type googleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (c *GoogleCalendarClient) discoverSources(ctx context.Context, accessToken string) ([]CalendarSource, error) {
	var result []CalendarSource
	next := googleCalendarBaseURL + "/users/me/calendarList"
	for next != "" {
		var page struct {
			NextPageToken string `json:"nextPageToken"`
			Items         []struct {
				ID       string `json:"id"`
				Summary  string `json:"summary"`
				TimeZone string `json:"timeZone"`
			} `json:"items"`
		}
		if err := c.calendarGet(ctx, next, accessToken, &page); err != nil {
			return nil, err
		}
		for _, item := range page.Items {
			name, timezone := strings.TrimSpace(item.Summary), strings.TrimSpace(item.TimeZone)
			if name == "" {
				name = item.ID
			}
			if timezone == "" {
				timezone = "UTC"
			}
			result = append(result, CalendarSource{ExternalCalendarID: item.ID, DisplayName: name, Timezone: timezone, Role: CalendarRoleCommitment})
		}
		if page.NextPageToken == "" {
			next = ""
		} else {
			next = googleCalendarBaseURL + "/users/me/calendarList?pageToken=" + url.QueryEscape(page.NextPageToken)
		}
	}
	return result, nil
}

func (c *GoogleCalendarClient) listEvents(ctx context.Context, accessToken string, source CalendarSource, planDate time.Time, location *time.Location) ([]CalendarEvent, error) {
	start := time.Date(planDate.Year(), planDate.Month(), planDate.Day(), 0, 0, 0, 0, location)
	end := start.AddDate(0, 0, 31)
	next := googleCalendarBaseURL + "/calendars/" + url.PathEscape(source.ExternalCalendarID) + "/events?singleEvents=true&orderBy=startTime&timeMin=" + url.QueryEscape(start.Format(time.RFC3339)) + "&timeMax=" + url.QueryEscape(end.Format(time.RFC3339))
	var result []CalendarEvent
	for next != "" {
		var page struct {
			NextPageToken string                `json:"nextPageToken"`
			Items         []googleCalendarEvent `json:"items"`
		}
		if err := c.calendarGet(ctx, next, accessToken, &page); err != nil {
			return nil, err
		}
		for _, item := range page.Items {
			event, include, err := normalizeCalendarEvent(item, source, start, location)
			if err != nil {
				return nil, err
			}
			if include {
				result = append(result, event)
			}
		}
		if page.NextPageToken == "" {
			next = ""
		} else {
			next += "&pageToken=" + url.QueryEscape(page.NextPageToken)
		}
	}
	return result, nil
}

func (c *GoogleCalendarClient) calendarGet(ctx context.Context, endpoint, accessToken string, out any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create Google Calendar request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("send Google Calendar request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Google Calendar request returned %s", response.Status)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(out); err != nil {
		return fmt.Errorf("decode Google Calendar response: %w", err)
	}
	return nil
}

type googleCalendarEvent struct {
	ID          string             `json:"id"`
	Summary     string             `json:"summary"`
	Description string             `json:"description"`
	Location    string             `json:"location"`
	HTMLLink    string             `json:"htmlLink"`
	Start       googleCalendarDate `json:"start"`
	End         googleCalendarDate `json:"end"`
}
type googleCalendarDate struct {
	DateTime string `json:"dateTime"`
	Date     string `json:"date"`
}

func normalizeCalendarEvent(raw googleCalendarEvent, source CalendarSource, today time.Time, location *time.Location) (CalendarEvent, bool, error) {
	event := CalendarEvent{CalendarSourceID: source.ID, ExternalEventID: raw.ID, CalendarName: source.DisplayName, Role: source.Role, Instructions: source.PlanningInstructions, Title: strings.TrimSpace(raw.Summary), HTMLURL: raw.HTMLLink}
	if event.Title == "" {
		event.Title = "Untitled event"
	}
	if raw.Start.Date != "" {
		start, err := time.ParseInLocation(time.DateOnly, raw.Start.Date, location)
		if err != nil {
			return CalendarEvent{}, false, fmt.Errorf("parse Calendar all-day start: %w", err)
		}
		end, err := time.ParseInLocation(time.DateOnly, raw.End.Date, location)
		if err != nil {
			return CalendarEvent{}, false, fmt.Errorf("parse Calendar all-day end: %w", err)
		}
		event.AllDay, event.StartDate, event.EndDate = true, &start, &end
		return event, true, nil
	}
	start, err := time.Parse(time.RFC3339, raw.Start.DateTime)
	if err != nil {
		return CalendarEvent{}, false, fmt.Errorf("parse Calendar event start: %w", err)
	}
	end, err := time.Parse(time.RFC3339, raw.End.DateTime)
	if err != nil {
		return CalendarEvent{}, false, fmt.Errorf("parse Calendar event end: %w", err)
	}
	event.StartsAt, event.EndsAt = &start, &end
	dayOffset := int(start.In(location).Sub(today).Hours() / 24)
	if dayOffset == 0 {
		event.Description, event.Location = raw.Description, raw.Location
	}
	return event, true, nil
}

func randomURLValue(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate OAuth random value: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
