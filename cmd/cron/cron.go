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
 *
 * The scheduler runs asynchronously in the background.
 *
 * @return void
 */
func SetupCron() {
	s := gocron.NewScheduler(time.UTC)

	// Run non-capacity scraper immediately on startup
	go scraper.ScrapeNonCapacityRoutes()

	// Run cleanup immediately on startup
	go scraper.CleanupOldSailings()

	// Schedule non-capacity routes every 1 hour
	s.Every(1).Hour().Do(func() {
		scraper.ScrapeNonCapacityRoutes()
	})

	// Schedule database cleanup every 6 hours to remove old sailing data
	s.Every(6).Hours().Do(func() {
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
