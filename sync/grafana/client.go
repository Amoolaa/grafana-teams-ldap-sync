package grafana

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	searchTeamsPath           = "/api/teams/search"
	teamsPath                 = "/api/teams"
	BulkUpdateTeamMembersPath = "/api/teams/%d/teams"
)

var (
	ErrTeamNotFound = errors.New("team not found")
)

type Client struct {
	HTTPClient *http.Client
	Config     Config
}

func NewClient(cfg Config) *Client {
	return &Client{
		Config: cfg,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func Request[T any](c *Client, method, path string, body any, params map[string]string, orgId int) (*T, error) {
	u, err := url.JoinPath(c.Config.URL, path)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	parsedURL, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	if len(params) > 0 {
		q := parsedURL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		parsedURL.RawQuery = q.Encode()
	}

	var reqBody *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal payload: %w", err)
		}
		reqBody = bytes.NewReader(b)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, parsedURL.String(), reqBody)
	if err != nil {
		return nil, fmt.Errorf("unable to create new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if c.Config.User != "" && c.Config.Password != "" {
		req.SetBasicAuth(c.Config.User, c.Config.Password)
	}

	if orgId != 0 {
		req.Header.Set("X-Grafana-Org-Id", strconv.Itoa(orgId))
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http client do error: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}
	return &result, nil
}

func (c *Client) GetTeam(orgId int, team string) (*Team, error) {
	p := map[string]string{"query": team}
	t, err := Request[TeamList](c, http.MethodGet, searchTeamsPath, nil, p, orgId)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch team list: %w", err)
	}

	if t.TotalCount == 0 {
		return nil, ErrTeamNotFound
	}

	return &t.Teams[0], nil
}

func (c *Client) AddTeam(orgId int, team string) (*AddTeamResponse, error) {
	payload := AddTeamPayload{Name: team}
	resp, err := Request[AddTeamResponse](c, http.MethodPost, teamsPath, payload, nil, orgId)
	if err != nil {
		return nil, fmt.Errorf("failed to add team %s in orgId %d: %w", team, orgId, err)
	}
	return resp, nil
}

func (c *Client) BulkUpdateTeamMembers(orgId, teamId int, memberEmails, adminEmails []string) (*BulkUpdateTeamMembersResponse, error) {
	payload := BulkUpdateTeamMembersPayload{Members: memberEmails, Admins: adminEmails}
	path := fmt.Sprintf(BulkUpdateTeamMembersPath, teamId)
	resp, err := Request[BulkUpdateTeamMembersResponse](c, http.MethodPut, path, payload, nil, orgId)
	if err != nil {
		return nil, fmt.Errorf("failed to update team id %d in orgId %d: %w", teamId, orgId, err)
	}
	return resp, nil
}
