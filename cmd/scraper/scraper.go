package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"log"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"

	"github.com/jeffcstock/bc-ferries-api/cmd/db"
	"github.com/jeffcstock/bc-ferries-api/cmd/models"
	"github.com/jeffcstock/bc-ferries-api/cmd/staticdata"
)

// Shared HTTP client to prevent memory leaks from creating new clients
// HTTP clients maintain connection pools, so reusing one is more efficient
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

/*
 * CleanupOldSailings
 *
 * Deletes sailing records older than 48 hours from both capacity and non-capacity tables.
 * This prevents the database from growing indefinitely and consuming memory.
 *
 * @return void
 */
func CleanupOldSailings() {
	// Calculate the cutoff date (48 hours ago)
	cutoffDate := time.Now().Add(-48 * time.Hour).Format("2006-01-02")

	// Delete old capacity routes
	sqlCapacity := `DELETE FROM capacity_routes WHERE date < $1`
	result, err := db.Conn.Exec(sqlCapacity, cutoffDate)
	if err != nil {
		log.Printf("CleanupOldSailings: failed to delete old capacity routes: %v", err)
	} else {
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			log.Printf("CleanupOldSailings: deleted %d old capacity route(s)", rowsAffected)
		}
	}

	// Delete old non-capacity routes
	sqlNonCapacity := `DELETE FROM non_capacity_routes WHERE date < $1`
	result, err = db.Conn.Exec(sqlNonCapacity, cutoffDate)
	if err != nil {
		log.Printf("CleanupOldSailings: failed to delete old non-capacity routes: %v", err)
	} else {
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			log.Printf("CleanupOldSailings: deleted %d old non-capacity route(s)", rowsAffected)
		}
	}
}

/*
 * MakeCurrentConditionsLink
 *
 * Makes a link to the current conditions page for a given departure and destination
 *
 * @param string departure
 * @param string destination
 *
 * @return string
 */
func MakeCurrentConditionsLink(departure, destination string) string {
	return "https://www.bcferries.com/current-conditions/" + departure + "-" + destination
}

/*
 * MakeScheduleLink
 *
 * Builds a link to the SEASONAL schedule page for a given departure and destination.
 * Seasonal pages contain the weekly tables used by the non-capacity scraper.
 *
 * @param string departure
 * @param string destination
 *
 * @return string
 */
func MakeScheduleLink(departure, destination string) string {
    return "https://www.bcferries.com/routes-fares/schedules/seasonal/" + departure + "-" + destination
}

/*
 * ScrapeCapacityRoutes
 *
 * Scrapes capacity routes
 *
 * @return void
 */
func ScrapeCapacityRoutes() {
	departureTerminals := staticdata.GetCapacityDepartureTerminals()
	destinationTerminals := staticdata.GetCapacityDestinationTerminals()

	for i := 0; i < len(departureTerminals); i++ {
		for j := 0; j < len(destinationTerminals[i]); j++ {
			link := MakeCurrentConditionsLink(departureTerminals[i], destinationTerminals[i][j])

			// Make HTTP GET request using shared client
			req, err := http.NewRequest("GET", link, nil)
			if err != nil {
				log.Printf("ScrapeCapacityRoutes: failed to create request for %s: %v", link, err)
				continue
			}

			req.Header.Add("User-Agent", "Mozilla")
			response, err := httpClient.Do(req)
			if err != nil {
				log.Printf("ScrapeCapacityRoutes: failed to fetch %s: %v", link, err)
				continue
			}

			defer response.Body.Close()

			document, err := goquery.NewDocumentFromReader(response.Body)
			if err != nil {
				log.Printf("ScrapeCapacityRoutes: failed to parse response from %s: %v", link, err)
				continue
			}

			ScrapeCapacityRoute(document, departureTerminals[i], destinationTerminals[i][j])
		}
	}
}

/*
 * ScrapeCapacityRoute
 *
 * Scrapes capacity data for a given route
 *
 * @param *goquery.Document document
 * @param string fromTerminalCode
 * @param string toTerminalCode
 *
 * @return void
 */
func ScrapeCapacityRoute(document *goquery.Document, fromTerminalCode string, toTerminalCode string) {
	// Get current date in Pacific Time (BC Ferries operates in PT)
	loc, err := time.LoadLocation("America/Vancouver")
	if err != nil {
		log.Printf("ScrapeCapacityRoute: failed to load PT location: %v", err)
		loc = time.UTC
	}
	currentDate := time.Now().In(loc).Format("2006-01-02")

	route := models.CapacityRoute{
		Date:             currentDate,
		RouteCode:        fromTerminalCode + toTerminalCode,
		ToTerminalCode:   toTerminalCode,
		FromTerminalCode: fromTerminalCode,
		Sailings:         []models.CapacitySailing{},
	}

	document.Find("table.detail-departure-table").Each(func(i int, table *goquery.Selection) {
		table.Find("tbody").Each(func(j int, tbody *goquery.Selection) {
                tbody.Find("tr.mobile-friendly-row").Each(func(k int, row *goquery.Selection) {
                    // Init sailing
                    sailing := models.CapacitySailing{}

                    row.Find("td").Each(func(l int, td *goquery.Selection) {
                        rowTextLower := strings.ToLower(row.Text())

                        // Handle explicitly cancelled rows
                        if strings.Contains(rowTextLower, "cancelled") {
                            sailing.SailingStatus = "cancelled"

                            if l == 0 {
                                // Scheduled time and vessel
                                timeString := strings.Join(strings.Fields(strings.TrimSpace(td.Text())), " ")
                                re := regexp.MustCompile(`(?P<Time>\d{1,2}:\d{2} [ap]m)(?: \(Tomorrow\))? (?P<VesselName>.+)`)
                                matches := re.FindStringSubmatch(strings.Join(strings.Fields(timeString), " "))
                                if len(matches) >= 3 {
                                    sailing.DepartureTime = matches[1]
                                    sailing.VesselName = matches[2]
                                }
                            } else if l == 1 {
                                // Capture reason if present under the red text block
                                // Prefer the second <p> which often holds the reason
                                reason := strings.TrimSpace(td.Find("div.text-red p").Eq(1).Text())
                                if reason == "" {
                                    // Fallback to the whole red block text
                                    reason = strings.TrimSpace(td.Find("div.text-red").Text())
                                }
                                if reason != "" {
                                    sailing.VesselStatus = reason
                                }
                            }
                        } else if strings.Contains(row.Text(), "Arrived") {
                            sailing.SailingStatus = "past"

                            if l == 0 {
                                timeString := strings.Join(strings.Fields(strings.TrimSpace(td.Find("p").Text())), " ")

							re := regexp.MustCompile(`(?P<DepartureTime>\d{1,2}:\d{2} [ap]m) Departed (?P<ActualDepartureTime>\d{1,2}:\d{2} [ap]m) (?P<VesselName>.+)`)

							// Find the matches
							matches := re.FindStringSubmatch(strings.Join(strings.Fields(timeString), " "))

							if len(matches) == 0 {
								fmt.Println("No matches found, regex error")
							} else {
								// Extracting named groups
								actualDepartureTime := matches[2]
								vesselName := matches[3]

								sailing.DepartureTime = actualDepartureTime
								sailing.VesselName = vesselName
							}
						} else if l == 1 {
							arrivalString := td.Find("div.cc-message-updates").Text()

							re := regexp.MustCompile(`Arrived: (?P<ArrivalTime>\d{1,2}:\d{2} [ap]m)`)

							// Find the matches
							matches := re.FindStringSubmatch(strings.Join(strings.Fields(arrivalString), " "))

							if len(matches) == 0 {
								fmt.Println("No matches found, regex error")
							} else {
								// Extracting named group
								arrivalTime := matches[1]

								sailing.ArrivalTime = arrivalTime
							}
						}
                        } else if strings.Contains(row.Text(), "ETA") || strings.Contains(row.Text(), "...") {
                            sailing.SailingStatus = "current"

						if l == 0 {
							timeString := strings.Join(strings.Fields(strings.TrimSpace(td.Find("p").Text())), " ")

							re := regexp.MustCompile(`(?P<DepartureTime>\d{1,2}:\d{2} [ap]m) Departed (?P<ActualDepartureTime>\d{1,2}:\d{2} [ap]m) (?P<VesselName>.+)`)

							// Find the matches
							matches := re.FindStringSubmatch(strings.Join(strings.Fields(timeString), " "))

							if len(matches) == 0 {
								fmt.Println("No matches found, regex error")
							} else {
								// Extracting named groups
								actualDepartureTime := matches[2]
								vesselName := matches[3]

								sailing.DepartureTime = actualDepartureTime
								sailing.VesselName = vesselName
							}
						} else if l == 1 {
							etaString := td.Find("div.cc-message-updates").Text()

							re := regexp.MustCompile(`ETA : (?P<ETA>\d{1,2}:\d{2} [ap]m|Variable)`)

							// Find the matches
							matches := re.FindStringSubmatch(strings.Join(strings.Fields(etaString), " "))

							if len(matches) == 0 {
								sailing.ArrivalTime = "..."
							} else {
								// Extracting named group
								etaTime := matches[1]

								sailing.ArrivalTime = etaTime
							}
						}
                        } else if strings.Contains(row.Text(), "Details") || strings.Contains(row.Text(), "%") || strings.Contains(strings.ToLower(row.Text()), "full") {
                            sailing.SailingStatus = "future"

						if l == 0 {
							// schedule time, vessel
							timeString := strings.Join(strings.Fields(strings.TrimSpace(td.Text())), " ")

							re := regexp.MustCompile(`(?P<Time>\d{1,2}:\d{2} [ap]m)(?: \(Tomorrow\))? (?P<VesselName>.+)`)

							// Find the matches
							matches := re.FindStringSubmatch(strings.Join(strings.Fields(timeString), " "))

							if len(matches) == 0 {
								fmt.Println("No matches found, regex error")
							} else {
								// Extracting named groups
								time := matches[1]
								vesselName := matches[2]

								sailing.DepartureTime = time
								sailing.VesselName = vesselName
							}
						} else if l == 1 {
							// details link
							// if word "Details" is in row, request from link, otherwise take percentage
							fillDetailsString := td.Text()

                                if strings.Contains(fillDetailsString, "Details") {
                                    td.Find("a.vehicle-info-link").Each(func(m int, s *goquery.Selection) {
									href, exists := s.Attr("href")
									link := strings.ReplaceAll("https://www.bcferries.com"+href, " ", "%20")

									if exists {
										req, err := http.NewRequest("GET", link, nil)
										if err != nil {
											log.Printf("ScrapeCapacityRoute: failed to create details request for %s: %v", link, err)
											return
										}

										req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
										response, err := httpClient.Do(req)
										if err != nil {
											log.Printf("ScrapeCapacityRoute: failed to fetch details from %s: %v", link, err)
											return
										}

										defer response.Body.Close()

										fillDocument, err := goquery.NewDocumentFromReader(response.Body)
										if err != nil {
											log.Printf("ScrapeCapacityRoute: failed to parse fill details from %s: %v", link, err)
											return
										}

										// fmt.Println(fillDocument.Text())
										fillDocument.Find("p.vehicle-icon-text").Each(func(o int, percentageText *goquery.Selection) {
                                            if o == 0 {
                                                fillPercentage := strings.TrimSpace(percentageText.Text())

                                                if strings.Contains(strings.ToLower(fillPercentage), "full") {
                                                    sailing.Fill = 100
                                                    sailing.CarFill = 100
                                                    sailing.OversizeFill = 100
                                                } else {
                                                    fillPercentageInt, err := strconv.Atoi(strings.ReplaceAll(fillPercentage, "%", ""))
													if err != nil {
														// ... handle error
													}

													sailing.Fill = 100 - fillPercentageInt
												}
                                            } else if o == 1 {
                                                fillPercentage := strings.TrimSpace(percentageText.Text())

                                                if strings.Contains(strings.ToLower(fillPercentage), "full") {
                                                    sailing.CarFill = 100
                                                } else {
                                                    fillPercentageInt, err := strconv.Atoi(strings.ReplaceAll(fillPercentage, "%", ""))
													if err != nil {
														// ... handle error
													}

													sailing.CarFill = 100 - fillPercentageInt
												}
                                            } else if o == 2 {
                                                fillPercentage := strings.TrimSpace(percentageText.Text())

                                                if strings.Contains(strings.ToLower(fillPercentage), "full") {
                                                    sailing.OversizeFill = 100
                                                } else {
                                                    fillPercentageInt, err := strconv.Atoi(strings.ReplaceAll(fillPercentage, "%", ""))
													if err != nil {
														// ... handle error
													}

													sailing.OversizeFill = 100 - fillPercentageInt
												}
											}
										})

									}
								})
							} else {
                                if strings.Contains(strings.ToLower(fillDetailsString), "full") {
                                    sailing.Fill = 100
                                    sailing.CarFill = 100
                                    sailing.OversizeFill = 100
                                } else {
                                    fillPercentage := strings.TrimSpace(td.Find("span.cc-vessel-percent-full").Text())

									fillPercentageInt, err := strconv.Atoi(strings.ReplaceAll(fillPercentage, "%", ""))
									if err != nil {
										// ... handle error
									}

									sailing.Fill = 100 - fillPercentageInt
								}
							}
						}
					}
				})

				// Generate unique sailing ID
				if sailing.DepartureTime != "" {
					sailing.ID = generateSailingID(route.RouteCode, currentDate, sailing.DepartureTime)
				}

				// Add sailing to route
				route.Sailings = append(route.Sailings, sailing)
			})
		})
	})

    // Try to find sailing duration text in a case-insensitive way
    sailingDuration := ""
    document.Find("span").Each(func(_ int, s *goquery.Selection) {
        if sailingDuration != "" {
            return
        }
        txt := strings.ReplaceAll(s.Text(), "\u00A0", " ")
        if strings.Contains(strings.ToLower(txt), "sailing duration:") {
            sailingDuration = txt
        }
    })
    sailingDuration = strings.ReplaceAll(sailingDuration, "Sailing duration:", "")
    sailingDuration = strings.ReplaceAll(sailingDuration, "sailing duration:", "")
    sailingDuration = strings.TrimSpace(sailingDuration)

	sailingsJson, err := json.Marshal(route.Sailings)
	if err != nil {
		log.Printf("ScrapeCapacityRoute: failed to marshal sailings for route %s: %v", route.RouteCode, err)
		return
	}

	sqlStatement := `
		INSERT INTO capacity_routes (
			route_code,
			from_terminal_code,
			to_terminal_code,
			date,
			sailing_duration,
			sailings
		)
		VALUES
			($1, $2, $3, $4, $5, $6) ON CONFLICT (route_code) DO
		UPDATE
		SET
			route_code = EXCLUDED.route_code,
			from_terminal_code = EXCLUDED.from_terminal_code,
			to_terminal_code = EXCLUDED.to_terminal_code,
			date = EXCLUDED.date,
			sailing_duration = EXCLUDED.sailing_duration,
			sailings = EXCLUDED.sailings
		WHERE
			capacity_routes.route_code = EXCLUDED.route_code`
	_, err = db.Conn.Exec(sqlStatement, route.RouteCode, route.FromTerminalCode, route.ToTerminalCode, currentDate, sailingDuration, sailingsJson)
	if err != nil {
		log.Printf("ScrapeCapacityRoute: failed to insert route %s: %v", route.RouteCode, err)
		return
	}
}

/*
 * ScrapeNonCapacityRoutes
 *
 * Scrapes non-capacity routes
 *
 * @return void
 */
func ScrapeNonCapacityRoutes() {
	log.Println("ScrapeNonCapacityRoutes: Starting scrape of Southern Gulf Islands routes...")

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Build vessel database from departures pages
	log.Println("ScrapeNonCapacityRoutes: Building vessel database...")
	vesselDatabase := BuildVesselDatabase(ctx)
	log.Printf("ScrapeNonCapacityRoutes: Vessel database built with %d terminals", len(vesselDatabase))

	departureTerminals := staticdata.GetNonCapacityDepartureTerminals()
	destinationTerminals := staticdata.GetNonCapacityDestinationTerminals()

	successCount := 0
	totalAttempts := 0

	for i := 0; i < len(departureTerminals); i++ {
		for j := 0; j < len(destinationTerminals[i]); j++ {
			totalAttempts++
			link := MakeScheduleLink(departureTerminals[i], destinationTerminals[i][j])

			html, err := fetchWithChromedp(ctx, link)
			if err != nil {
				fmt.Printf("ScrapeBCNonCapacityRoutes: chromedp fetch failed for %s: %v\n", link, err)
				continue
			}

			document, err := goquery.NewDocumentFromReader(strings.NewReader(html))
			if err != nil {
				fmt.Printf("ScrapeBCNonCapacityRoutes: failed to parse HTML for %s: %v\n", link, err)
				continue
			}

			if ScrapeNonCapacityRoute(document, departureTerminals[i], destinationTerminals[i][j], vesselDatabase) {
				successCount++
			}
		}
	}

	log.Printf("ScrapeNonCapacityRoutes: Completed! Successfully scraped %d/%d routes", successCount, totalAttempts)
}

/*
 * ScrapeNonCapacityRoute
 *
 * Scrapes schedule data for a given route
 *
 * @param *goquery.Document document
 * @param string fromTerminalCode
 * @param string toTerminalCode
 * @param map[string]map[string]string vesselDatabase - Vessel database (terminal → time → vessel)
 *
 * @return bool - true if route was successfully scraped and saved, false otherwise
 */
func ScrapeNonCapacityRoute(document *goquery.Document, fromTerminalCode, toTerminalCode string, vesselDatabase map[string]map[string]string) bool {
	loc, err := time.LoadLocation("America/Vancouver")
	if err != nil {
		log.Printf("ScrapeNonCapacityRoute: failed to load PT location: %v", err)
		return false
	}

	normalizeDay := func(s string) string {
		s = strings.TrimSpace(strings.ToUpper(s))
		// Treat trailing "S" as optional: MONDAY == MONDAYS
		if strings.HasSuffix(s, "S") {
			s = strings.TrimSuffix(s, "S")
		}
		return s
	}
	today := time.Now().In(loc)
	todayNorm := normalizeDay(today.Weekday().String()) // e.g. "MONDAY"
	currentDate := today.Format("2006-01-02")

	route := models.NonCapacityRoute{
		Date:             currentDate,
		RouteCode:        fromTerminalCode + toTerminalCode,
		FromTerminalCode: fromTerminalCode,
		ToTerminalCode:   toTerminalCode,
		Sailings:         []models.NonCapacitySailing{},
	}

    // ---- Step 1: find the seasonal schedule table that contains weekday theads
    var scheduleTable *goquery.Selection
    document.Find("table.table-seasonal-schedule").Each(func(_ int, t *goquery.Selection) {
        if scheduleTable != nil {
            return
        }
        // Heuristic: a real schedule table has thead rows with day labels
        if t.Find("thead tr[data-schedule-day], thead [data-schedule-day], thead h4, thead b").Length() > 0 {
            scheduleTable = t
        }
    })
    // Fallback to the historical assumption (2nd table) if heuristic fails
    if scheduleTable == nil {
        scheduleTable = document.Find("table.table-seasonal-schedule").Eq(1)
    }
    if scheduleTable == nil || scheduleTable.Length() == 0 {
        log.Printf("ScrapeNonCapacityRoute: seasonal schedule table not found")
        return false
    }

	// ---- Step 2: find the <thead> whose day matches today (MONDAY vs MONDAYS, any case)
    var dayBody *goquery.Selection
    scheduleTable.Find("thead").Each(func(_ int, thead *goquery.Selection) {
		if dayBody != nil {
			return
		}

		// Prefer the attribute if present.
		dayAttr := thead.Find("tr").First().AttrOr("data-schedule-day", "")
		dayAttrNorm := normalizeDay(dayAttr)

		match := (dayAttrNorm != "" && dayAttrNorm == todayNorm)
		if !match {
			// Fallback: try visible text inside thead (e.g., MONDAY Depart)
			txt := thead.Find("h4, b, th").First().Text()
			txtNorm := normalizeDay(txt)
			// If the text contains the weekday token (e.g., "MONDAY DEPART"), accept it.
			match = (txtNorm == todayNorm) || strings.Contains(txtNorm, todayNorm)
		}

		if match {
			// ---- Step 3: go to the NEXT sibling under the table; skip to the first <tbody>
			tb := thead.Next()
			for tb.Length() > 0 && goquery.NodeName(tb) != "tbody" {
				tb = tb.Next()
			}
			if tb.Length() > 0 && goquery.NodeName(tb) == "tbody" {
				dayBody = tb
			}
		}
	})

	if dayBody == nil {
		log.Printf("ScrapeNonCapacityRoute: no tbody found for today (%s) in second table", todayNorm)
		return false
	}

    clean := func(s string) string {
        s = strings.ReplaceAll(s, "\u00a0", " ") // NBSP -> space
        return strings.TrimSpace(s)
    }

    // Parse a list of month/day mentions from a status string like
    // "Only on Sep 14, 28 & Oct 12" or "Except on Oct 13".
    // Returns a set keyed by "MM-DD" for quick lookup.
    parseMentionedDates := func(note string, year int) map[string]struct{} {
        res := make(map[string]struct{})
        if note == "" {
            return res
        }
        lower := strings.ToLower(note)

        monthMap := map[string]time.Month{
            "jan": time.January, "january": time.January,
            "feb": time.February, "february": time.February,
            "mar": time.March, "march": time.March,
            "apr": time.April, "april": time.April,
            "may": time.May,
            "jun": time.June, "june": time.June,
            "jul": time.July, "july": time.July,
            "aug": time.August, "august": time.August,
            "sep": time.September, "sept": time.September, "september": time.September,
            "oct": time.October, "october": time.October,
            "nov": time.November, "november": time.November,
            "dec": time.December, "december": time.December,
        }

        // 1) Find explicit Month Day pairs
        mdRe := regexp.MustCompile(`(?i)(jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|jun(?:e)?|jul(?:y)?|aug(?:ust)?|sep(?:t(?:ember)?)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\s+(\d{1,2})`)
        matches := mdRe.FindAllStringSubmatch(lower, -1)

        for _, m := range matches {
            monKey := m[1]
            dayStr := m[2]
            if mon, ok := monthMap[monKey]; ok {
                if d, err := strconv.Atoi(dayStr); err == nil {
                    key := fmt.Sprintf("%02d-%02d", int(mon), d)
                    res[key] = struct{}{}
                }
            }
        }

        // 2) Handle shorthand days following a month (e.g., "Sep 14, 28 & Oct 12")
        //    For each segment that starts with a month, capture trailing , <day> pieces until next month appears
        segRe := regexp.MustCompile(`(?i)(jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|jun(?:e)?|jul(?:y)?|aug(?:ust)?|sep(?:t(?:ember)?)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\s+\d{1,2}([^a-z]*)`)
        pos := 0
        for {
            loc := segRe.FindStringSubmatchIndex(lower[pos:])
            if loc == nil {
                break
            }
            // Extract month for this segment
            seg := lower[pos+loc[0] : pos+loc[1]]
            mon := mdRe.FindStringSubmatch(seg)
            if len(mon) >= 3 {
                monKey := mon[1]
                if monVal, ok := monthMap[monKey]; ok {
                    // After the first "Month DD", scan the tail for , DD patterns
                    tail := seg[len(mon[0]):]
                    // Match bare days like ", 28" without unsupported lookaheads
                    ddRe := regexp.MustCompile(`(?i)[,&\s]+(\d{1,2})\b`)
                    ddMatches := ddRe.FindAllStringSubmatch(tail, -1)
                    for _, dm := range ddMatches {
                        if d, err := strconv.Atoi(dm[1]); err == nil {
                            key := fmt.Sprintf("%02d-%02d", int(monVal), d)
                            res[key] = struct{}{}
                        }
                    }
                }
            }
            pos += loc[1]
        }

        return res
    }

    // ---- Step 4: parse rows in the found <tbody>
    dayBody.Find("tr.schedule-table-row").Each(func(_ int, row *goquery.Selection) {
        tds := row.Find("td")
        if tds.Length() < 3 {
            return
        }

        // Extract clean departure time (first time token) and any status notes
        depCell := tds.Eq(1)
        depRaw := clean(depCell.Text())

        // Capture red/black status notes if present (e.g., Only on..., Except on..., Foot passengers only, Dangerous goods only)
        var statuses []string
        var redNotes []string
        depCell.Find("p").Each(func(_ int, p *goquery.Selection) {
            txt := clean(p.Text())
            if txt == "" {
                return
            }
            // Only keep informative notes, skip if it's just whitespace
            // Common classes include red-text italic-style or text-black
            if p.HasClass("red-text") || p.HasClass("text-black") {
                statuses = append(statuses, txt)
                if p.HasClass("red-text") {
                    redNotes = append(redNotes, txt)
                }
            }
        })

        // Extract the first time-like token from the departure cell
        depTime := depRaw
        if re := regexp.MustCompile(`(?i)\b\d{1,2}:\d{2}\s*[ap]m\b`); re != nil {
            if m := re.FindString(depRaw); m != "" {
                depTime = m
            }
        }

        // Extract clean arrival time (first time token)
        arrCell := tds.Eq(2)
        arrRaw := clean(arrCell.Text())
        arrTime := arrRaw
        if re := regexp.MustCompile(`(?i)\b\d{1,2}:\d{2}\s*[ap]m\b`); re != nil {
            if m := re.FindString(arrRaw); m != "" {
                arrTime = m
            }
        }

        // Extract sailing duration from the 4th column
        var sailingDuration string
        if tds.Length() > 3 {
            durationCell := tds.Eq(3)
            sailingDuration = clean(durationCell.Text())
        }

        // Extract events from the 5th column (stops, thru fares, transfers)
        var events []models.SailingEvent
        if tds.Length() > 4 {
            eventsCell := tds.Eq(4)
            eventsCell.Find("p.mb-1").Each(func(_ int, p *goquery.Selection) {
                // Look for the event type span
                var eventType string
                if p.Find(".schedule-leg-type-thru-fare").Length() > 0 {
                    eventType = "thruFare"
                } else if p.Find(".schedule-leg-type-stop").Length() > 0 {
                    eventType = "stop"
                } else if p.Find(".schedule-leg-type-transfer").Length() > 0 {
                    eventType = "transfer"
                }

                // Extract the terminal name (the text after the link/icon)
                terminalName := ""
                p.Contents().Each(func(_ int, node *goquery.Selection) {
                    // Look for <span> elements that are NOT part of the link/icon
                    if goquery.NodeName(node) == "span" {
                        // Skip if it's one of the icon/type spans
                        if node.HasClass("bcf") || node.HasClass("schedule-leg-type-thru-fare") ||
                           node.HasClass("schedule-leg-type-stop") || node.HasClass("schedule-leg-type-transfer") {
                            return
                        }
                        text := clean(node.Text())
                        if text != "" {
                            terminalName = text
                        }
                    }
                })

                // Add the event if we found both type and terminal name
                if eventType != "" && terminalName != "" {
                    events = append(events, models.SailingEvent{
                        Type:         eventType,
                        TerminalName: terminalName,
                    })
                }
            })
        }

        // Filter: drop dangerous goods only sailings outright
        depLower := strings.ToLower(depCell.Text())
        if strings.Contains(depLower, "dangerous goods only") || strings.Contains(depLower, "no passengers permitted") {
            return
        }

        // Apply exception rules: "Only on <dates>" and "Except on <dates>"
        // Build a combined note string from red notes
        combinedRed := strings.ToLower(strings.Join(redNotes, "; "))
        todayKey := fmt.Sprintf("%02d-%02d", int(today.Month()), today.Day())

        // If there is an "only on" note, include only if today is listed
        if strings.Contains(combinedRed, "only on") {
            dates := parseMentionedDates(combinedRed, today.Year())
            if _, ok := dates[todayKey]; !ok {
                return
            }
        }
        // If there is an "except on" note, exclude if today is listed
        if strings.Contains(combinedRed, "except on") {
            dates := parseMentionedDates(combinedRed, today.Year())
            if _, ok := dates[todayKey]; ok {
                return
            }
        }

        // Pre-calculate dwell time for vessel lookup
        // We need to estimate this before building legs since BuildLegs needs it for time calculations
        sailingDurationMin := parseDurationToMinutes(sailingDuration)

        // Count stops/transfers (exclude thruFares)
        stopCount := 0
        for _, event := range events {
            if event.Type == "stop" || event.Type == "transfer" {
                stopCount++
            }
        }

        // Estimate total travel time from leg info
        // Build a temporary route to get leg durations
        originCode := route.RouteCode[:3]
        destinationCode := route.RouteCode[3:]
        estimatedTravelMin := 0

        if len(events) == 0 {
            // Direct sailing - just one leg
            if legInfo := staticdata.GetLegInfo(originCode, destinationCode); legInfo != nil {
                estimatedTravelMin = legInfo.AvgDurationMin
            }
        } else {
            // Multi-leg sailing - estimate each leg
            // First leg: origin to first event terminal
            firstEventCode := staticdata.GetTerminalCodeByName(events[0].TerminalName)
            if legInfo := staticdata.GetLegInfo(originCode, firstEventCode); legInfo != nil {
                estimatedTravelMin += legInfo.AvgDurationMin
            }

            // Intermediate legs: between event terminals
            for i := 0; i < len(events)-1; i++ {
                fromCode := staticdata.GetTerminalCodeByName(events[i].TerminalName)
                toCode := staticdata.GetTerminalCodeByName(events[i+1].TerminalName)
                if legInfo := staticdata.GetLegInfo(fromCode, toCode); legInfo != nil {
                    estimatedTravelMin += legInfo.AvgDurationMin
                }
            }

            // Final leg: last event terminal to destination
            lastEventCode := staticdata.GetTerminalCodeByName(events[len(events)-1].TerminalName)
            if legInfo := staticdata.GetLegInfo(lastEventCode, destinationCode); legInfo != nil {
                estimatedTravelMin += legInfo.AvgDurationMin
            }
        }

        // Calculate estimated average dwell time
        avgDwellMin := 0
        estimatedDwellMin := sailingDurationMin - estimatedTravelMin
        if stopCount > 0 && estimatedDwellMin > 0 {
            avgDwellMin = estimatedDwellMin / stopCount
        }

        // Now build legs with vessel lookup using calculated avg dwell time
        legs := models.BuildLegs(route.RouteCode, events, depTime, vesselDatabase, avgDwellMin)

        s := models.NonCapacitySailing{
            DepartureTime:   depTime,
            ArrivalTime:     arrTime,
            SailingDuration: sailingDuration,
            Events:          events,
            Legs:            legs,
        }
        if len(statuses) > 0 {
            s.VesselStatus = strings.Join(statuses, " | ")
        }

        // Generate unique sailing ID
        if s.DepartureTime != "" {
            s.ID = generateSailingID(route.RouteCode, currentDate, s.DepartureTime)
        }

        // Calculate actual dwell time (time spent at stops) from built legs
        totalTravelMin := 0
        for _, leg := range legs {
            if leg.AvgDurationMin != nil {
                totalTravelMin += *leg.AvgDurationMin
            }
        }

        s.TotalTravelMin = totalTravelMin
        s.TotalDwellMin = sailingDurationMin - totalTravelMin
        s.StopCount = stopCount

        // Calculate actual average dwell time per stop
        if stopCount > 0 && s.TotalDwellMin > 0 {
            avgDwell := s.TotalDwellMin / stopCount
            s.AvgDwellPerStopMin = &avgDwell
        }

        if s.DepartureTime != "" || s.ArrivalTime != "" {
            route.Sailings = append(route.Sailings, s)
        }
    })

	// Optional: route-level duration (from the first row's 4th cell, if present)
	sailingDuration := ""
	if firstRow := dayBody.Find("tr.schedule-table-row").First(); firstRow.Length() > 0 {
		if cell := firstRow.Find("td").Eq(3); cell.Length() > 0 {
			sailingDuration = clean(cell.Text())
		}
	}

	// ---- Step 5: save
	sailingsJSON, err := json.Marshal(route.Sailings)
	if err != nil {
		log.Printf("ScrapeNonCapacityRoute: marshal error for %s: %v", route.RouteCode, err)
		return false
	}

	sqlStatement := `
		INSERT INTO non_capacity_routes (
			route_code,
			from_terminal_code,
			to_terminal_code,
			date,
			sailing_duration,
			sailings
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (route_code) DO UPDATE SET
			from_terminal_code = EXCLUDED.from_terminal_code,
			to_terminal_code = EXCLUDED.to_terminal_code,
			date = EXCLUDED.date,
			sailing_duration = EXCLUDED.sailing_duration,
			sailings = EXCLUDED.sailings
	`
	_, err = db.Conn.Exec(sqlStatement,
		route.RouteCode, route.FromTerminalCode, route.ToTerminalCode, currentDate, sailingDuration, sailingsJSON,
	)
	if err != nil {
		log.Printf("ScrapeNonCapacityRoute: DB insert/update failed for %s: %v", route.RouteCode, err)
		return false
	}

	log.Printf("ScrapeNonCapacityRoute: ✓ %s scraped successfully with %d sailing(s)", route.RouteCode, len(route.Sailings))
	return true
}

/********************/
/* Helper Functions */
/********************/

/*
 * BuildVesselDatabase
 *
 * Scrapes BC Ferries departures pages for all non-capacity terminals to build a database
 * of vessel names indexed by terminal code and departure time.
 *
 * @param ctx context.Context - Chromedp context for rendering pages
 *
 * @return map[string]map[string]string - Map of terminal code → (departure time → vessel name)
 */
func BuildVesselDatabase(ctx context.Context) map[string]map[string]string {
	log.Println("BuildVesselDatabase: Starting to build vessel database...")

	vesselDB := make(map[string]map[string]string)
	terminals := staticdata.GetNonCapacityDepartureTerminals()

	for _, terminalCode := range terminals {
		url := fmt.Sprintf("https://www.bcferries.com/current-conditions/departures?terminalCode=%s", terminalCode)
		log.Printf("BuildVesselDatabase: Fetching departures for terminal %s", terminalCode)

		html, err := fetchWithChromedp(ctx, url)
		if err != nil {
			log.Printf("BuildVesselDatabase: Failed to fetch %s: %v", url, err)
			vesselDB[terminalCode] = make(map[string]string)
			continue
		}

		document, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			log.Printf("BuildVesselDatabase: Failed to parse HTML for %s: %v", terminalCode, err)
			vesselDB[terminalCode] = make(map[string]string)
			continue
		}

		// Initialize map for this terminal
		vesselDB[terminalCode] = make(map[string]string)
		sailingCount := 0

		// Find all sailing rows across all tables on the page
		document.Find("tr.padding-departures-td").Each(func(i int, row *goquery.Selection) {
			// Extract vessel name from first column
			vesselName := strings.TrimSpace(row.Find("td").Eq(0).Find("a[href*='/on-the-ferry/our-fleet/']").Text())

			// Extract SCHEDULED time from second column
			scheduledTime := ""
			row.Find("td").Eq(1).Find("ul.departures-time-ul").Each(func(j int, ul *goquery.Selection) {
				// Look for the UL that contains "SCHEDULED:"
				if strings.Contains(ul.Text(), "SCHEDULED:") {
					// Extract the time from the span
					timeText := strings.TrimSpace(ul.Find("span.text-lowercase").Text())
					if timeText != "" {
						// Convert to lowercase (e.g., "7:10 AM" → "7:10 am")
						scheduledTime = strings.ToLower(timeText)
					}
				}
			})

			// Only store if we found both vessel name and scheduled time
			if vesselName != "" && scheduledTime != "" {
				vesselDB[terminalCode][scheduledTime] = vesselName
				sailingCount++
			}
		})

		log.Printf("BuildVesselDatabase: Terminal %s - extracted %d sailings", terminalCode, sailingCount)
	}

	log.Printf("BuildVesselDatabase: Completed! Built database for %d terminals", len(vesselDB))
	return vesselDB
}

/*
 * FindVesselByTimeWindow
 *
 * Searches for a vessel departing from a terminal within a time window.
 * Returns the vessel name if found, or a pointer to "UNKNOWN" if not found.
 *
 * @param vesselDB map[string]string - Map of departure time → vessel name for a specific terminal
 * @param targetTime string - Target departure time in lowercase 12-hour format (e.g., "5:35 pm")
 * @param windowMinutes int - Search window in minutes (±windowMinutes from targetTime)
 *
 * @return *string - Pointer to vessel name, or pointer to "UNKNOWN" if not found
 */
func FindVesselByTimeWindow(vesselDB map[string]string, targetTime string, windowMinutes int) *string {
	unknown := "UNKNOWN"

	// Parse target time
	layouts := []string{"3:04 pm", "3:04 PM", "03:04 pm", "03:04 PM", "3:04pm", "3:04PM"}
	var targetParsed time.Time
	var err error
	for _, layout := range layouts {
		targetParsed, err = time.Parse(layout, targetTime)
		if err == nil {
			break
		}
	}
	if err != nil {
		log.Printf("FindVesselByTimeWindow: Failed to parse target time '%s': %v", targetTime, err)
		return &unknown
	}

	// Search for vessels within the time window
	type match struct {
		vessel   string
		timeDiff time.Duration
	}
	var matches []match

	for depTimeStr, vesselName := range vesselDB {
		// Parse departure time
		var depTimeParsed time.Time
		for _, layout := range layouts {
			depTimeParsed, err = time.Parse(layout, depTimeStr)
			if err == nil {
				break
			}
		}
		if err != nil {
			continue
		}

		// Calculate time difference
		diff := depTimeParsed.Sub(targetParsed)
		absDiff := diff
		if absDiff < 0 {
			absDiff = -absDiff
		}

		// Check if within window
		if absDiff <= time.Duration(windowMinutes)*time.Minute {
			matches = append(matches, match{vessel: vesselName, timeDiff: absDiff})
		}
	}

	// No matches found
	if len(matches) == 0 {
		log.Printf("FindVesselByTimeWindow: No vessel found within %d minutes of %s", windowMinutes, targetTime)
		return &unknown
	}

	// Find closest match
	closest := matches[0]
	for _, m := range matches[1:] {
		if m.timeDiff < closest.timeDiff {
			closest = m
		}
	}

	// Log if multiple matches
	if len(matches) > 1 {
		log.Printf("FindVesselByTimeWindow: Multiple matches (%d) within window for %s, picking closest: %s", len(matches), targetTime, closest.vessel)
	}

	return &closest.vessel
}

/*
 * fetchWithChromedp
 *
 * Uses a headless Chrome browser to fetch and render the full HTML content of a given URL.
 * This is used to bypass JavaScript-based protections like Queue-it by executing the page
 * in a real browser environment.
 *
 * @param string url - The URL to navigate to
 *
 * @return string - The full outer HTML of the rendered page
 * @return error - Any error encountered during navigation or retrieval
 */
func fetchWithChromedp(ctx context.Context, url string) (string, error) {
	var html string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.OuterHTML("html", &html),
	)

	return html, err
}

/*
 * convertTo24HourFormat
 *
 * Converts a 12-hour time format string to 24-hour format without colons.
 * Handles edge cases like "(Tomorrow)" suffix and various spacing.
 *
 * Examples:
 *   "7:00 am" → "0700"
 *   "3:15 pm" → "1515"
 *   "11:30 pm" → "2330"
 *   "7:00 am (Tomorrow)" → "0700"
 *
 * @param string time12h - The 12-hour format time string
 *
 * @return string - The 24-hour format time without colons (e.g., "0700", "1515")
 */
func convertTo24HourFormat(time12h string) string {
	// Remove "(Tomorrow)" suffix if present
	time12h = strings.ReplaceAll(time12h, "(Tomorrow)", "")
	time12h = strings.TrimSpace(time12h)

	// Parse the time using Go's time package
	// Expected formats: "3:04 pm", "3:04 PM", "03:04 pm", etc.
	layouts := []string{
		"3:04 pm",
		"3:04 PM",
		"03:04 pm",
		"03:04 PM",
		"3:04pm",
		"3:04PM",
	}

	var parsedTime time.Time
	var err error
	for _, layout := range layouts {
		parsedTime, err = time.Parse(layout, time12h)
		if err == nil {
			break
		}
	}

	if err != nil {
		log.Printf("convertTo24HourFormat: failed to parse time '%s': %v", time12h, err)
		return "0000"
	}

	// Format as 24-hour time without colons
	return parsedTime.Format("1504")
}

/*
 * generateSailingID
 *
 * Generates a unique sailing ID in the format: {routeCode}-{date}-{time24h}
 *
 * Examples:
 *   "TSAPOB", "2025-11-11", "7:00 am" → "TSAPOB-2025-11-11-0700"
 *   "HSBNÄN", "2025-11-11", "3:15 pm" → "HSBNÄN-2025-11-11-1515"
 *
 * @param string routeCode - The route code (e.g., "TSAPOB")
 * @param string date - ISO date string (e.g., "2025-11-11")
 * @param string departureTime - 12-hour format time (e.g., "7:00 am")
 *
 * @return string - Unique sailing ID
 */
func generateSailingID(routeCode, date, departureTime string) string {
	time24h := convertTo24HourFormat(departureTime)
	return fmt.Sprintf("%s-%s-%s", routeCode, date, time24h)
}

/*
 * parseDurationToMinutes
 *
 * Converts a duration string to total minutes.
 * Handles multiple formats from BC Ferries schedules.
 *
 * Examples:
 *   "2h 15m" → 135
 *   "1h 30m" → 90
 *   "55m" → 55
 *   "01:40" → 100
 *   "00:50" → 50
 *
 * @param string duration - Duration string in various formats
 *
 * @return int - Total duration in minutes (0 if parsing fails)
 */
func parseDurationToMinutes(duration string) int {
	if duration == "" {
		return 0
	}

	totalMinutes := 0

	// Format 1: "2h 15m" or "1h" or "45m"
	hoursMatch := regexp.MustCompile(`(\d+)h`).FindStringSubmatch(duration)
	if len(hoursMatch) > 1 {
		hours, _ := strconv.Atoi(hoursMatch[1])
		totalMinutes += hours * 60
	}

	minutesMatch := regexp.MustCompile(`(\d+)m`).FindStringSubmatch(duration)
	if len(minutesMatch) > 1 {
		minutes, _ := strconv.Atoi(minutesMatch[1])
		totalMinutes += minutes
	}

	// Format 2: "01:40" (HH:MM)
	if totalMinutes == 0 && strings.Contains(duration, ":") {
		parts := strings.Split(duration, ":")
		if len(parts) == 2 {
			hours, err1 := strconv.Atoi(parts[0])
			minutes, err2 := strconv.Atoi(parts[1])
			if err1 == nil && err2 == nil {
				totalMinutes = hours*60 + minutes
			}
		}
	}

	return totalMinutes
}
