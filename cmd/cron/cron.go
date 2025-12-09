package cron

import (
	"time"

	"github.com/go-co-op/gocron"
	"github.com/jeffcstock/bc-ferries-api/cmd/scraper"
)

/*
 * SetupCron
 *
 * Initializes and starts scheduled background scraping tasks using gocron.
 *
 * - Scrapes non-capacity route data immediately on startup, then every 1 hour.
 * - Cleans up sailing records older than 48 hours every 6 hours.
 * - Capacity route scraping is disabled (not needed for Southern Gulf Islands focus).
 * - MaxConcurrentJobs(1) prevents memory exhaustion from overlapping scraper runs.
 *
 * The scheduler runs asynchronously in the background.
 */
func SetupCron() {
	s := gocron.NewScheduler(time.UTC)

	// Prevent concurrent scraper runs - critical for memory-constrained environments
	s.SetMaxConcurrentJobs(1, gocron.WaitMode)

	// Schedule non-capacity routes every 1 hour, run immediately on startup
	s.Every(1).Hour().StartImmediately().Do(func() {
		scraper.ScrapeNonCapacityRoutes()
	})

	// Schedule database cleanup every 6 hours, run immediately on startup
	s.Every(6).Hours().StartImmediately().Do(func() {
		scraper.CleanupOldSailings()
	})

	// Capacity scraping disabled - not needed for Southern Gulf Islands
	// Uncomment below if you need capacity routes in the future:
	// go scraper.ScrapeCapacityRoutes()
	// s.Every(1).Minute().Do(func() {
	//     scraper.ScrapeCapacityRoutes()
	// })

	s.StartAsync()
}
