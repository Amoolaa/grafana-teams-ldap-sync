package sync

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
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
		var orgErrs error
		// need to filter through and check if any users do not exist (i.e. have never logged in), otherwise batch update will fail
		currentUsers, err := s.GrafanaClient.GetAllUsersInOrg(m.OrgID)
		if err != nil {
			orgErrs = errors.Join(orgErrs, err)
			s.Logger.Error("errors while syncing org", "orgId", m.OrgID, "errors", orgErrs)
			continue
		}

		s.Logger.Info("fetched users from org", "orgId", m.OrgID, "num_users", len(currentUsers))

		for _, t := range m.Teams {
			var memberEmails, adminEmails []string
			if t.MemberUserFilter != "" {
				memberEmails, err = s.GetEmails(ldapConn, t, t.MemberUserFilter)
				if err != nil {
					s.Logger.Error("failed to get users for filter", "filter", t.MemberUserFilter, "error", err)
					orgErrs = errors.Join(orgErrs, err)
					continue
				}
			}
			if t.AdminUserFilter != "" {
				adminEmails, err = s.GetEmails(ldapConn, t, t.AdminUserFilter)
				if err != nil {
					s.Logger.Error("failed to get users for filter", "filter", t.AdminUserFilter, "error", err)
					orgErrs = errors.Join(orgErrs, err)
					continue
				}
			}

			// convert to set (for quick lookup)
			activeEmailSet := make(map[string]bool, len(currentUsers))
			for _, user := range currentUsers {
				activeEmailSet[user.Email] = true
			}

			// any emails that appear in both members and admins will only be part of the admin list
			adminSet := make(map[string]bool, len(adminEmails))
			for _, email := range adminEmails {
				adminSet[email] = true
			}

			var filteredMembers []string
			for _, email := range memberEmails {
				if !adminSet[email] {
					if activeEmailSet[email] {
						filteredMembers = append(filteredMembers, email)
					} else {
						s.Logger.Warn("user missing in grafana dropped from member list", "email", email)
					}
				}
			}

			var filteredAdmins []string
			for _, email := range adminEmails {
				if adminSet[email] {
					if activeEmailSet[email] {
						filteredAdmins = append(filteredAdmins, email)
					} else {
						s.Logger.Warn("user missing in grafana dropped from admin list", "email", email)
					}
				}
			}

			s.Logger.Info("successfully fetched user emails", "adminEmails", filteredAdmins, "memberEmails", filteredMembers)
			if err := s.TeamSync(m.OrgID, t.Name, filteredMembers, filteredAdmins); err != nil {
				orgErrs = errors.Join(orgErrs, err)
				continue
			}
		}
		if orgErrs != nil {
			s.Logger.Error("errors while syncing org", "orgId", m.OrgID, "errors", orgErrs)
			errs = errors.Join(errs, orgErrs)
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
