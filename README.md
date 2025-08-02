# grafana-teams-ldap-sync

A tool to synchronize Grafana Teams membership with LDAP/Active Directory groups based on LDAP user filters. It supports multiple organisations, team member and admin roles, and can run either as a one-time sync or as a server with an endpoint that does adhoc syncs.

## Installation

- **Docker:** images are published at `docker pull ghcr.io/amoolaa/grafana-teams-ldap-sync:latest`
- **Binary:** download a precompiled binary from the [Releases](https://github.com/amoolaa/grafana-teams-ldap-sync/releases) page
- or clone and build from source

## Usage
Run `./grafana-teams-ldap-sync sync` for one-time sync run, or `./grafana-teams-ldap-sync server` to run as a server. You can use crontab or kube CronJobs to schedule syncs.

### Configuration
The tool requires two configuration files - the main config file for the syncer (`--config` flag), and another for the mappings from users to teams (`--mapping`). See [config.yaml](https://github.com/Amoolaa/grafana-teams-ldap-sync/blob/main/.dev/config.yaml) and [mapping.yaml](https://github.com/Amoolaa/grafana-teams-ldap-sync/blob/main/.dev/mapping.yaml) for sample configs.

`GRAFANA_PASSWORD`, `GRAFANA_USER` and `LDAP_PASSWORD` should be set as environment variables. The Grafana user credentials must have admin access to the organisations you are mapping against or be a Grafana server admin. The `LDAP_PASSWORD` is used to bind using the `ldap.bind_dn` variable in the main config file.

## How it works
Using the sample `mapping.yaml`:
```yaml
mapping:
  - org_id: 1
    teams:
      - name: foo
        admin_user_filter: "(objectClass=inetOrgPerson)"
        member_user_filter: "(objectClass=inetOrgPerson)"
```
the sync will:
- Create a team with name "foo" in orgId 1 if it doesn't already exist
- Fetch users from LDAP using `admin_user_filter`, `member_user_filter`. If a user is returned in both the `admin_user_filter` and `member_user_filter`, they are made an admin of the team. We use the email attribute specified in `ldap.attributes.email`
- Drop any users who are not users in Grafana (in other words, they must have logged in to Grafana at least once to be eligible for the sync).
- Perform a bulk update to the members of the team the email attributes of users  specified in `ldap.attributes.email`.

If you delete mapping entries it will not remove the created teams, you must manually clean them up.