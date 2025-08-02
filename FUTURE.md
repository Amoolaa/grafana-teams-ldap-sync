# Future

If time permits, the following are things I'd like to consider:
- Add group to sync aswell as user filter
- Expose an endpoint to sync a user on demand. This could be triggered in auth proxy setups in Grafana to sync on user login.
- Check if teams are already synced before running a bulk update
- Provision folders (best left to the Grafana operator or a separate project imo)
- If performance becomes an issue, use goroutines for concurrency (from my testing performance is not an issue)