package grafana

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

type TeamList struct {
	TotalCount int    `json:"totalCount"`
	Teams      []Team `json:"teams"`
}

type AddTeamPayload struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type AddTeamResponse struct {
	Message string `json:"message"`
	TeamID  int    `json:"teamId"`
	UID     string `json:"uid"`
}

type BulkUpdateTeamMembersPayload struct {
	Members []string `json:"members"`
	Admins  []string `json:"admins"`
}

type BulkUpdateTeamMembersResponse struct {
	Message string `json:"message"`
}
