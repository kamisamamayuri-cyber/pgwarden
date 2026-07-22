package main

import (
	"context"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/cron"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service"
)

func initSchedule(cr *cron.Cron, servs *service.Service, tags []string) {
	/*
		Initial executions
	*/

	servs.ExecutionsService.HousekeepExecutions()
	servs.RestorationsService.HousekeepRestorations()
	servs.DiscoveryService.HousekeepEvents()
	servs.AuditLogsService.HousekeepAuditLogs()
	servs.AuthService.DeleteOldSessions()
	servs.DatabasesService.TestAllDatabases(tags)
	servs.DestinationsService.TestAllDestinations()
	servs.VersionCheckService.Refresh(context.Background())

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
		servs.RestorationsService.HousekeepRestorations()
	})
	if err != nil {
		logger.FatalError(
			"error scheduling restoration housekeeping",
			logger.KV{"error": err},
		)
	}

	err = cr.UpsertJob(uuid.New(), "UTC", "*/10 * * * *", func() {
		servs.DiscoveryService.HousekeepEvents()
	})
	if err != nil {
		logger.FatalError(
			"error scheduling discovery events housekeeping",
			logger.KV{"error": err},
		)
	}

	err = cr.UpsertJob(uuid.New(), "UTC", "*/10 * * * *", func() {
		servs.AuditLogsService.HousekeepAuditLogs()
	})
	if err != nil {
		logger.FatalError(
			"error scheduling audit log housekeeping",
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
		servs.DatabasesService.TestAllDatabases(tags)
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

	err = cr.UpsertJob(uuid.New(), "UTC", "0 */6 * * *", func() {
		servs.VersionCheckService.Refresh(context.Background())
	})
	if err != nil {
		logger.FatalError(
			"error scheduling version check", logger.KV{"error": err},
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
