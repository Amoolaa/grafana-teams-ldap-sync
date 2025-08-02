package sync

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/Amoolaa/grafana-teams-ldap-sync/sync/grafana"
	"github.com/go-ldap/ldap/v3"
)

type Syncer struct {
	Logger        *slog.Logger
	Config        Config
	GrafanaClient *grafana.Client
}

func NewSyncer(logger *slog.Logger) *Syncer {
	return &Syncer{
		Logger: logger,
	}
}

func (s *Syncer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /sync", s.SyncHandler)
	s.Logger.Info("serving traffic on :8080")
	return http.ListenAndServe(":8080", mux)
}

func (s *Syncer) SyncHandler(w http.ResponseWriter, r *http.Request) {
	if err := s.Sync(); err != nil {
		s.Logger.Error("sync error", "error", err)
		http.Error(w, fmt.Sprintf("sync error: %v", err), http.StatusInternalServerError)
	}
}

func getLdapConn(cfg LDAPConfig) (*ldap.Conn, error) {
	address := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	timeout := 10 * time.Second
	var conn *ldap.Conn
	if cfg.UseSSL {
		tlsCfg := &tls.Config{
			InsecureSkipVerify: cfg.InsecureSkipVerify,
			ServerName:         cfg.Host,
		}
		c, err := tls.DialWithDialer(&net.Dialer{Timeout: timeout}, "tcp", address, tlsCfg)
		if err != nil {
			return nil, err
		}
		conn = ldap.NewConn(c, true)
	} else {
		c, err := net.DialTimeout("tcp", address, timeout)
		if err != nil {
			return nil, err
		}
		conn = ldap.NewConn(c, false)
	}
	conn.Start()
	return conn, nil
}

func (s *Syncer) Sync() error {
	ldapConn, err := getLdapConn(s.Config.LDAP)
	if err != nil {
		return fmt.Errorf("LDAP connect error: %w", err)
	}
	defer ldapConn.Close()

	err = ldapConn.Bind(s.Config.LDAP.BindDN, s.Config.LDAP.Password)
	if err != nil {
		return fmt.Errorf("LDAP bind error: %w", err)
	}

	var errs error

	for _, m := range s.Config.Mapping {
		for _, t := range m.Teams {
			var memberEmails, adminEmails []string
			if t.MemberUserFilter != "" {
				memberEmails, err = s.GetEmails(ldapConn, t, t.MemberUserFilter)
				if err != nil {
					s.Logger.Error("failed to get users for filter", "filter", t.MemberUserFilter, "error", err)
					errs = errors.Join(errs, err)
				}
			}
			if t.AdminUserFilter != "" {
				adminEmails, err = s.GetEmails(ldapConn, t, t.AdminUserFilter)
				if err != nil {
					s.Logger.Error("failed to get users for filter", "filter", t.AdminUserFilter, "error", err)
					errs = errors.Join(errs, err)
				}
			}

			adminSet := make(map[string]struct{}, len(adminEmails))
			for _, email := range adminEmails {
				adminSet[email] = struct{}{}
			}

			var filteredMembers []string
			for _, email := range memberEmails {
				if _, isAdmin := adminSet[email]; !isAdmin {
					filteredMembers = append(filteredMembers, email)
				}
			}

			s.Logger.Info("successfully fetched user emails", "adminEmails", adminEmails, "memberEmails", filteredMembers)
			if err := s.TeamSync(m.OrgID, t.Name, filteredMembers, adminEmails); err != nil {
				errs = errors.Join(errs, err)
			}
		}
	}
	return errs
}

func (s *Syncer) GetEmails(ldapConn *ldap.Conn, t TeamConfig, filter string) ([]string, error) {
	searchRequest := ldap.NewSearchRequest(
		s.Config.LDAP.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter, []string{s.Config.LDAP.Attributes.Email},
		nil,
	)

	sr, err := ldapConn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}
	var emails []string
	for _, entry := range sr.Entries {
		emails = append(emails, entry.GetAttributeValues(s.Config.LDAP.Attributes.Email)...)
	}
	return emails, nil
}

func (s *Syncer) TeamSync(orgId int, team string, memberEmails, adminEmails []string) error {
	t, err := s.GrafanaClient.GetTeam(orgId, team)
	var teamId int
	if err == nil {
		s.Logger.Info("team found", "team", team, "org", orgId, "teamId", t.ID)
		teamId = t.ID
	} else if errors.Is(err, grafana.ErrTeamNotFound) {
		s.Logger.Info("team doesn't exist, so adding team", "team", team, "org", orgId)
		resp, err := s.GrafanaClient.AddTeam(orgId, team)
		if err != nil {
			return fmt.Errorf("error adding team: %w", err)
		}
		teamId = resp.TeamID
		s.Logger.Info("team successfully created", "resp", resp)
	} else {
		return fmt.Errorf("error fetching team: %w", err)
	}

	resp, err := s.GrafanaClient.BulkUpdateTeamMembers(orgId, teamId, memberEmails, adminEmails)
	if err != nil {
		return fmt.Errorf("error bulk updating team members: %w", err)
	}
	s.Logger.Info("successfully bulk updated team members", "resp", resp)
	return nil
}
