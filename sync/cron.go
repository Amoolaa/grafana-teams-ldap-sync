package sync

import (
	"fmt"

	"github.com/go-co-op/gocron/v2"
)

func (s *Syncer) InitCron() (gocron.Scheduler, error) {
	cron, err := gocron.NewScheduler()
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}

	_, err = cron.NewJob(
		gocron.CronJob(s.Config.Sync.Schedule, false),
		gocron.NewTask(s.SyncCron),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to schedule cron job: %w", err)
	}
	cron.Start()
	s.Logger.Info("cron job started", "schedule", s.Config.Sync.Schedule)
	return cron, nil
}

func (s *Syncer) SyncCron() {
	if err := s.Sync(); err != nil {
		s.Logger.Error("cron sync error", "error", err)
	} else {
		s.Logger.Info("cron sync completed")
	}
}
