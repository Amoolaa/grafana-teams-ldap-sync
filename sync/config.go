package sync

import (
	"fmt"

	"github.com/Amoolaa/grafana-teams-ldap-sync/sync/grafana"
)

type LDAPConfig struct {
	Host             string           `koanf:"host"`
	BindDN           string           `koanf:"bind_dn"`
	Password         string           `koanf:"password"`
	BaseDN           string           `koanf:"base_dn"`
	ServerAttributes ServerAttributes `koanf:"server_attributes"`
}

type ServerAttributes struct {
	Email string `koanf:"email"`
}

type SyncConfig struct {
	Enabled  bool   `koanf:"enabled"`
	Schedule string `koanf:"schedule"`
}

type TeamConfig struct {
	Name             string `koanf:"name"`
	AdminUserFilter  string `koanf:"admin_user_filter"`
	MemberUserFilter string `koanf:"member_user_filter"`
}

type MappingConfig struct {
	OrgID int          `koanf:"org_id"`
	Teams []TeamConfig `koanf:"teams"`
}

type Config struct {
	LDAP    LDAPConfig      `koanf:"ldap"`
	Grafana grafana.Config  `koanf:"grafana"`
	Sync    SyncConfig      `koanf:"sync"`
	Mapping []MappingConfig `koanf:"mapping"`
}

func ValidateConfig(c Config) error {
	for _, m := range c.Mapping {
		for _, t := range m.Teams {
			// team config must contain at least an admin or member filter
			if t.AdminUserFilter == "" && t.MemberUserFilter == "" {
				return fmt.Errorf("one of admin_user_filter or member_user_filter must be specified for team %s in orgId %d", t.Name, m.OrgID)
			}
		}
	}
	return nil
}
