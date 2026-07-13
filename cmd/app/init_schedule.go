package main

import (
	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/cron"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service"
)

func initSchedule(cr *cron.Cron, servs *service.Service) {
	/*
		Initial executions
	*/

	servs.ExecutionsService.HousekeepExecutions()
	servs.AuthService.DeleteOldSessions()
	servs.DatabasesService.TestAllDatabases()
	servs.DestinationsService.TestAllDestinations()

	/*
		Schedules
	*/

	err := cr.UpsertJob(uuid.New(), "UTC", "*/10 * * * *", func() {
		servs.ExecutionsService.HousekeepExecutions()
	})
	if err != nil {
		logger.FatalError(
			"error scheduling execution housekeeping",
			logger.KV{"error": err},
		)
	}

	err = cr.UpsertJob(uuid.New(), "UTC", "*/10 * * * *", func() {
		servs.AuthService.DeleteOldSessions()
	})
	if err != nil {
		logger.FatalError(
			"error scheduling deletion of old sessions", logger.KV{"error": err},
		)
	}

	err = cr.UpsertJob(uuid.New(), "UTC", "*/10 * * * *", func() {
		servs.DatabasesService.TestAllDatabases()
	})
	if err != nil {
		logger.FatalError(
			"error scheduling databases tests", logger.KV{"error": err},
		)
	}

	err = cr.UpsertJob(uuid.New(), "UTC", "*/10 * * * *", func() {
		servs.DestinationsService.TestAllDestinations()
	})
	if err != nil {
		logger.FatalError(
			"error scheduling destinations tests", logger.KV{"error": err},
		)
	}

	servs.BackupsService.ScheduleAll()

	// Re-sync cron jobs with the backups table: picks up backups created,
	// edited or toggled via the UI on another pod (web role has no cron).
	err = cr.UpsertJob(uuid.New(), "UTC", "*/5 * * * *", func() {
		servs.BackupsService.ScheduleAll()
	})
	if err != nil {
		logger.FatalError(
			"error scheduling backups re-sync", logger.KV{"error": err},
		)
	}

	err = servs.DiscoveryService.Schedule()
	if err != nil {
		logger.FatalError(
			"error scheduling discovery", logger.KV{"error": err},
		)
	}
}
