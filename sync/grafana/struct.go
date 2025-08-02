package grafana

// GET /api/teams/search
type TeamList struct {
	TotalCount int    `json:"totalCount"`
	Teams      []Team `json:"teams"`
}

type Team struct {
	ID            int    `json:"id"`
	UID           string `json:"uid"`
	OrgID         int    `json:"orgId"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	ExternalUID   string `json:"externalUID"`
	IsProvisioned bool   `json:"isProvisioned"`
	AvatarURL     string `json:"avatarUrl"`
	MemberCount   int    `json:"memberCount"`
	Permission    int    `jso:"permission"`
}

// POST /api/teams
type AddTeamPayload struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type AddTeamResponse struct {
	Message string `json:"message"`
	TeamID  int    `json:"teamId"`
	UID     string `json:"uid"`
}

// POST /api/teams/<id>/members
type BulkUpdateTeamMembersPayload struct {
	Members []string `json:"members"`
	Admins  []string `json:"admins"`
}

type BulkUpdateTeamMembersResponse struct {
	Message string `json:"message"`
}

// GET /api/orgs/users
type User struct {
	OrgID         int    `json:"orgId"`
	UserID        int    `json:"userId"`
	Email         string `json:"email"`
	AvatarURL     string `json:"avatarUrl"`
	Login         string `json:"login"`
	Role          string `json:"role"`
	LastSeenAt    string `json:"lastSeenAt"`
	LastSeenAtAge string `json:"lastSeenAtAge"`
}
