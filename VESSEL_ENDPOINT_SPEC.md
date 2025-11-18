# Implementation Spec: `/vessels/:id` Endpoint

**Document Version**: 1.0
**Date**: 2025-11-17
**Project**: bc-ferries-api
**Status**: Ready for Implementation

---

## Table of Contents

1. [Product Goal](#1-product-goal)
2. [Technical Approach](#2-technical-approach)
3. [Data Structures](#3-data-structures)
4. [Implementation Steps](#4-implementation-steps)
5. [Testing Plan](#5-testing-plan)
6. [Files to Modify](#6-files-to-modify)
7. [Expected Output Example](#7-expected-output-example)
8. [Next Steps After Implementation](#8-next-steps-after-implementation)

---

## 1. Product Goal

### Problem
Inter-island ferry sailings show "UNKNOWN" vessels because the vessel database can't deduce which vessel operates multi-leg routes that don't appear on departures pages.

### Solution
Create an endpoint that reconstructs a vessel's complete daily route by aggregating all its departures across all terminals, allowing us to:
1. **Test** the vessel route reconstruction algorithm
2. **Verify** it produces accurate vessel timelines
3. **Later integrate** this logic to populate vessel database with inferred inter-island routes

### Success Criteria
Query **any vessel** by slug or MMSI and receive a complete chronological timeline of that vessel's movements for the day with accurate times and terminals.

**Example**: Query `/vessels/salish-eagle` (or `/vessels/316030626`) and see movements showing PLH → POB → PSB → TSA route. Query `/vessels/spirit-of-british-columbia` (or `/vessels/316001268`) and see TSA ↔ SWB runs.

---

## 2. Technical Approach

### Location
**bc-ferries-api** (Go) - vessel database already exists in memory after scraping

### Data Source
The `vesselDB` map structure built during scraping from BC Ferries departures pages:
```go
vesselDB[terminalCode][compositeKey] = vesselName
// Example:
vesselDB["PLH"]["3:35 pm-POB"] = "Salish Eagle"
vesselDB["POB"]["5:15 pm-PSB"] = "Salish Eagle"
```

**Note**: The composite key format is `"time-destinationCode"` (e.g., `"3:35 pm-POB"`). This was recently implemented to handle multiple vessels departing the same terminal at the same time to different destinations.

### Algorithm
1. Iterate through all terminals in `vesselDB`
2. For each terminal, iterate through all composite keys
3. If vessel name matches the requested vessel, extract movement data:
   - Origin terminal: from outer map key
   - Departure time + Destination: parse composite key
   - Arrival time: estimate from next movement's departure time (if available)
4. Sort movements chronologically by departure time
5. Post-process to set destination = next movement's origin (for missing destinations)
6. Convert to response format with full terminal objects

### Key Design Decisions

#### Time Format
- **Input**: 12-hour format from scraped data (e.g., "3:35 pm")
- **Output**: 24-hour format (e.g., "15:35")

#### Destination Deduction
- If composite key includes destination code → use it
- If not → deduce from next movement's origin terminal
- Last movement → `destination: null`

#### Error Handling
- **404**: Vessel name not found in MMSI mapping (invalid vessel)
- **200 with empty array**: Valid vessel name but no movements today

---

## 3. Data Structures

### API Endpoint
```
GET /vessels/:id
```

**Parameters**:
- `id` (path): Either MMSI (numeric) or vessel slug (kebab-case)
  - **MMSI**: `316005939`
  - **Slug**: `salish-eagle` (case-insensitive)

**Example Requests**:
```bash
# By MMSI (recommended for API clients)
curl http://localhost:8080/vessels/316030626 | jq

# By slug (human-readable, case-insensitive)
curl http://localhost:8080/vessels/salish-eagle | jq
curl http://localhost:8080/vessels/queen-of-cumberland | jq
```

### Response Format

```json
{
  "vesselName": "Salish Eagle",
  "mmsi": 316030626,
  "date": "2025-11-17",
  "movements": [
    {
      "origin": {
        "code": "PLH",
        "name": "Long Harbour",
        "service_area": "Southern Gulf Islands"
      },
      "destination": {
        "code": "POB",
        "name": "Otter Bay",
        "service_area": "Southern Gulf Islands"
      },
      "scheduledDepartureTime": "15:35",
      "scheduledArrivalTime": "16:25"
    }
  ]
}
```

### Go Type Definitions

Add these types to the appropriate files:

```go
// VesselMovement represents a single departure/leg
type VesselMovement struct {
    Origin                 staticdata.Terminal  `json:"origin"`
    Destination            *staticdata.Terminal `json:"destination"`            // nullable
    ScheduledDepartureTime string               `json:"scheduledDepartureTime"` // 24hr format "15:35"
    ScheduledArrivalTime   *string              `json:"scheduledArrivalTime"`   // nullable, estimated from next movement
}

// VesselRouteResponse is the API response
type VesselRouteResponse struct {
    VesselName string            `json:"vesselName"`
    MMSI       *int              `json:"mmsi"`       // nullable if not found in mapping
    Date       string            `json:"date"`       // ISO 8601 date "2025-11-17"
    Movements  []VesselMovement  `json:"movements"`
}

// Internal structure for sorting before conversion
type rawMovement struct {
    OriginCode      string
    DestinationCode string    // may be empty
    DepartureTime   string    // "3:35 pm" format from scraper
    TimeOrder       time.Time // for sorting
}
```

---

## 4. Implementation Steps

### Step 1: Add MMSI Mapping with Reverse Lookup

**Data Source**: Vessel MMSI data exists in `/Users/jeff/Projects/sgi-ferries/backend/config/vessels.yaml`

Create `cmd/staticdata/vessels.go` (or similar YAML-based approach):

```go
package staticdata

var vesselMMSI = map[string]int{
    // Tsawwassen to Southern Gulf Islands
    "Salish Raven":                316030628,
    "Salish Heron":                316047943,
    "Salish Eagle":                316030626,

    // Vancouver (Tsawwassen) to Victoria (Swartz Bay)
    "Spirit of British Columbia":  316001268,
    "Spirit of Vancouver Island":  316001269,

    // Vancouver (Tsawwassen) to Fulford Harbour (Salt Spring Island)
    "Skeena Queen":                316001267,

    // Victoria (Swartz Bay) to Southern Gulf Islands
    "Queen of Cumberland":         316001252,

    // Add more vessels as needed
}

// Reverse lookup map (built in init)
var mmsiToVessel map[int]string

func init() {
    // Build reverse lookup map for MMSI → vessel name
    mmsiToVessel = make(map[int]string)
    for name, mmsi := range vesselMMSI {
        mmsiToVessel[mmsi] = name
    }
}

// GetVesselMMSI returns the MMSI for a vessel name, or nil if not found
func GetVesselMMSI(vesselName string) *int {
    if mmsi, ok := vesselMMSI[vesselName]; ok {
        return &mmsi
    }
    return nil
}

// GetVesselNameByMMSI returns vessel name for given MMSI, or empty string if not found
func GetVesselNameByMMSI(mmsi int) string {
    return mmsiToVessel[mmsi]
}
```

**Important Notes**:
- Vessel names in the vessel database may vary in casing (e.g., "SALISH EAGLE" vs "Salish Eagle")
- The `normalizeSlugToVesselName()` function converts slugs to Title Case
- Consider adding case-insensitive vessel name lookup to handle variations
- MMSI data sourced from `/Users/jeff/Projects/sgi-ferries/backend/config/vessels.yaml`

---

### Step 2: Add Vessel Route Reconstruction Function

Add to `cmd/scraper/scraper.go`:

```go
// ReconstructVesselRoute traces a vessel's movements from the vessel database
// Returns movements sorted chronologically with full terminal objects
func ReconstructVesselRoute(vesselName string, vesselDB map[string]map[string]string) []VesselMovement {
    var rawMovements []rawMovement

    // Iterate through all terminals and all composite keys
    // Note: Use case-insensitive comparison since vessel names may vary in casing
    vesselNameLower := strings.ToLower(vesselName)
    for terminalCode, timeMap := range vesselDB {
        for compositeKey, vessel := range timeMap {
            if strings.ToLower(vessel) == vesselNameLower {
                // Parse composite key: "3:35 pm-POB" or "3:35 pm" (no destination)
                parts := strings.Split(compositeKey, "-")
                departureTime := parts[0]
                destinationCode := ""
                if len(parts) > 1 {
                    destinationCode = parts[1]
                }

                // Parse time for sorting
                timeOrder, err := parseTime(departureTime)
                if err != nil {
                    continue  // Skip if can't parse time
                }

                rawMovements = append(rawMovements, rawMovement{
                    OriginCode:      terminalCode,
                    DestinationCode: destinationCode,
                    DepartureTime:   departureTime,
                    TimeOrder:       timeOrder,
                })
            }
        }
    }

    // Sort by time
    sort.Slice(rawMovements, func(i, j int) bool {
        return rawMovements[i].TimeOrder.Before(rawMovements[j].TimeOrder)
    })

    // Convert to VesselMovement with full terminal objects
    terminals := staticdata.GetTerminals()
    movements := make([]VesselMovement, 0, len(rawMovements))

    for i, raw := range rawMovements {
        movement := VesselMovement{
            ScheduledDepartureTime: convertTo24Hour(raw.DepartureTime),
        }

        // Set origin terminal
        if originTerminal, ok := terminals[raw.OriginCode]; ok {
            movement.Origin = originTerminal
        } else {
            continue  // Skip if origin terminal not found
        }

        // Set destination terminal
        var destCode string
        if raw.DestinationCode != "" {
            // Use explicit destination from composite key
            destCode = raw.DestinationCode
        } else if i < len(rawMovements)-1 {
            // Deduce from next movement's origin
            destCode = rawMovements[i+1].OriginCode
        }

        if destCode != "" {
            if destTerminal, ok := terminals[destCode]; ok {
                movement.Destination = &destTerminal
            }
        }

        // Estimate scheduled arrival time from next movement's departure
        if i < len(rawMovements)-1 {
            arrivalTime := convertTo24Hour(rawMovements[i+1].DepartureTime)
            movement.ScheduledArrivalTime = &arrivalTime
        }

        movements = append(movements, movement)
    }

    return movements
}

// parseTime converts time string to time.Time for sorting
func parseTime(timeStr string) (time.Time, error) {
    layouts := []string{"3:04 pm", "3:04 PM", "03:04 pm", "03:04 PM"}
    for _, layout := range layouts {
        if t, err := time.Parse(layout, timeStr); err == nil {
            return t, nil
        }
    }
    return time.Time{}, fmt.Errorf("unable to parse time: %s", timeStr)
}

// convertTo24Hour converts 12-hour time to 24-hour format
func convertTo24Hour(time12hr string) string {
    layouts := []string{"3:04 pm", "3:04 PM", "03:04 pm", "03:04 PM"}
    for _, layout := range layouts {
        if t, err := time.Parse(layout, time12hr); err == nil {
            return t.Format("15:04")
        }
    }
    return time12hr  // Return as-is if can't parse
}
```

---

### Step 3: Expose Vessel Database from Scraper

Modify `cmd/scraper/scraper.go` to make vessel database accessible:

```go
// Add global variable at package level
var globalVesselDB map[string]map[string]string

// Modify BuildVesselDatabase function to store the vessel DB globally
func BuildVesselDatabase(...) map[string]map[string]string {
    // ... existing code that builds vesselDB ...

    // Store globally before returning
    globalVesselDB = vesselDB
    return vesselDB
}

// Add getter function for router access
func GetVesselDatabase() map[string]map[string]string {
    return globalVesselDB
}
```

**Note**: Ensure thread safety if scraping happens concurrently with API requests. Consider adding a mutex if necessary.

---

### Step 4: Add Router Handler

Add to `cmd/router/routes.go`:

```go
import (
    "net/url"
    "encoding/json"
    "strconv"
    "strings"
    "time"
    // ... other imports
)

// GetVesselRoute returns a vessel's complete daily route
// Supports lookup by MMSI (numeric) or vessel slug (kebab-case)
func GetVesselRoute(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    vesselID := ps.ByName("id")

    // URL decode
    vesselID, err := url.QueryUnescape(vesselID)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{"error": "Invalid vessel ID"})
        return
    }

    var vesselName string

    // Try to parse as MMSI (integer)
    if mmsi, err := strconv.Atoi(vesselID); err == nil {
        // It's an MMSI - lookup vessel name
        vesselName = staticdata.GetVesselNameByMMSI(mmsi)
        if vesselName == "" {
            w.WriteHeader(http.StatusNotFound)
            json.NewEncoder(w).Encode(map[string]string{"error": "MMSI not found"})
            return
        }
    } else {
        // It's a slug - normalize to proper vessel name
        vesselName = normalizeSlugToVesselName(vesselID)
    }

    // Get vessel database
    vesselDB := scraper.GetVesselDatabase()

    // Reconstruct vessel route
    movements := scraper.ReconstructVesselRoute(vesselName, vesselDB)

    // Check if vessel exists
    if len(movements) == 0 {
        // Check if vessel name is valid (exists in MMSI map)
        mmsi := staticdata.GetVesselMMSI(vesselName)
        if mmsi == nil {
            // Invalid vessel name
            w.WriteHeader(http.StatusNotFound)
            json.NewEncoder(w).Encode(map[string]string{"error": "Vessel not found"})
            return
        }
        // Valid vessel but no movements today - return empty array
    }

    // Build response
    response := VesselRouteResponse{
        VesselName: vesselName,
        MMSI:       staticdata.GetVesselMMSI(vesselName),
        Date:       time.Now().Format("2006-01-02"),  // Current date in ISO 8601
        Movements:  movements,
    }

    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// normalizeSlugToVesselName converts "salish-eagle" to "Salish Eagle"
func normalizeSlugToVesselName(slug string) string {
    // Split by hyphens
    words := strings.Split(slug, "-")

    // Title case each word
    for i, word := range words {
        words[i] = strings.Title(strings.ToLower(word))
    }

    // Join with spaces
    return strings.Join(words, " ")
}

// Helper: Case-insensitive vessel name lookup for ReconstructVesselRoute
// BC Ferries vessel database may have inconsistent casing (e.g., "SALISH EAGLE" vs "Salish Eagle")
func findVesselCaseInsensitive(vesselDB map[string]map[string]string, targetVessel string) []rawMovement {
    var rawMovements []rawMovement
    targetLower := strings.ToLower(targetVessel)

    for terminalCode, timeMap := range vesselDB {
        for compositeKey, vessel := range timeMap {
            if strings.ToLower(vessel) == targetLower {
                // ... movement extraction logic
            }
        }
    }

    return rawMovements
}
```

---

### Step 5: Register Route

Add to `cmd/router/router.go`:

```go
// Add to the router setup function
router.GET("/vessels/:id", GetVesselRoute)
```

**Suggested placement**: Add near other v2 routes for consistency.

---

## 5. Testing Plan

### Prerequisites

1. Ensure bc-ferries-api is built and running:
   ```bash
   cd ~/Projects/bc-ferries-api
   docker-compose up --build
   ```

2. Wait for scraping to complete (watch logs for "BuildVesselDatabase" completion)

### Manual Testing

#### Test 1: Known Active Vessel (by slug)
```bash
curl http://localhost:8080/vessels/salish-eagle | jq
```

**Expected**:
- 200 status code
- JSON response with vessel name, MMSI, date, and movements array
- Date in ISO 8601 format (YYYY-MM-DD)
- Movements sorted chronologically
- Each movement has origin and destination terminal objects
- Times in 24hr format

#### Test 2: Same Vessel (by MMSI)
```bash
curl http://localhost:8080/vessels/316030626 | jq
```

**Expected**: Same response as Test 1 (validates both ID types work)

#### Test 3: Another Known Vessel
```bash
curl http://localhost:8080/vessels/salish-raven | jq
# OR
curl http://localhost:8080/vessels/316030628 | jq
```

**Expected**: Similar structure with different route

#### Test 4: Major Route Vessel
```bash
curl http://localhost:8080/vessels/spirit-of-british-columbia | jq
# OR
curl http://localhost:8080/vessels/316001268 | jq
```

**Expected**: Fewer movements (usually TSA ↔ SWB runs)

#### Test 5: Invalid Vessel Slug
```bash
curl http://localhost:8080/vessels/fake-vessel | jq
```

**Expected**:
- 404 status code
- Error message: `{"error": "Vessel not found"}`

#### Test 6: Invalid MMSI
```bash
curl http://localhost:8080/vessels/999999999 | jq
```

**Expected**:
- 404 status code
- Error message: `{"error": "MMSI not found"}`

#### Test 7: Valid Vessel, No Movements
```bash
curl http://localhost:8080/vessels/coastal-celebration | jq
```

**Expected** (if vessel exists but isn't scheduled today):
- 200 status code
- Response: `{"vesselName": "Coastal Celebration", "mmsi": 316012824, "movements": []}`

### Verification Checklist

- [ ] Can trace Salish Eagle route: PLH → POB → PSB → TSA (or similar)
- [ ] Response includes `date` field in ISO 8601 format (YYYY-MM-DD)
- [ ] Date matches current day (or day vessel database was scraped)
- [ ] Scheduled departure times match BC Ferries departures pages
- [ ] Terminal objects include `code`, `name`, `service_area` fields
- [ ] Scheduled arrival times estimated correctly (next departure time)
- [ ] No duplicate movements
- [ ] Handles vessels with single movement gracefully
- [ ] Handles vessels with 10+ movements
- [ ] Last movement has `scheduledArrivalTime: null`
- [ ] Destination deduction works for inter-island routes
- [ ] MMSI included when vessel in mapping
- [ ] 404 for invalid vessel names
- [ ] 200 with empty array for valid but unscheduled vessels

### Data Validation

Compare endpoint output with BC Ferries departures pages:

1. Visit https://www.bcferries.com/current-conditions/departures?terminalCode=PLH
2. Find "Salish Eagle" departure time
3. Verify it matches the endpoint response
4. Repeat for POB, PSB, etc.

---

## 6. Files to Modify

### New Files
- `cmd/staticdata/vessels.go` - MMSI mapping (**only if doesn't already exist**)

### Modified Files
- `cmd/scraper/scraper.go`:
  - Add `ReconstructVesselRoute()` function
  - Add `parseTime()` helper
  - Add `convertTo24Hour()` helper
  - Add `globalVesselDB` variable
  - Modify `BuildVesselDatabase()` to store globally
  - Add `GetVesselDatabase()` getter

- `cmd/router/routes.go`:
  - Add `VesselMovement` type
  - Add `VesselRouteResponse` type
  - Add `rawMovement` type
  - Add `GetVesselRoute()` handler
  - Add `normalizeSlugToVesselName()` helper

- `cmd/router/router.go`:
  - Register `GET /vessels/:id` route

---

## 7. Expected Output Example

### Request (by slug)
```bash
GET /vessels/salish-eagle
```

### Request (by MMSI)
```bash
GET /vessels/316030626
```

### Response (Example)
```json
{
  "vesselName": "Salish Eagle",
  "mmsi": 316030626,
  "date": "2025-11-17",
  "movements": [
    {
      "origin": {
        "code": "PLH",
        "name": "Long Harbour",
        "service_area": "Southern Gulf Islands"
      },
      "destination": {
        "code": "POB",
        "name": "Otter Bay",
        "service_area": "Southern Gulf Islands"
      },
      "scheduledDepartureTime": "15:35",
      "scheduledArrivalTime": "16:25"
    },
    {
      "origin": {
        "code": "POB",
        "name": "Otter Bay",
        "service_area": "Southern Gulf Islands"
      },
      "destination": {
        "code": "PSB",
        "name": "Sturdies Bay",
        "service_area": "Southern Gulf Islands"
      },
      "scheduledDepartureTime": "16:25",
      "scheduledArrivalTime": "17:15"
    },
    {
      "origin": {
        "code": "PSB",
        "name": "Sturdies Bay",
        "service_area": "Southern Gulf Islands"
      },
      "destination": {
        "code": "TSA",
        "name": "Tsawwassen",
        "service_area": "Metro Vancouver"
      },
      "scheduledDepartureTime": "17:15",
      "scheduledArrivalTime": null
    }
  ]
}
```

### Response for Invalid Vessel Slug
```bash
GET /vessels/fake-vessel
```

```json
{
  "error": "Vessel not found"
}
```
**Status**: 404 Not Found

### Response for Invalid MMSI
```bash
GET /vessels/999999999
```

```json
{
  "error": "MMSI not found"
}
```
**Status**: 404 Not Found

### Response for Valid Vessel, No Movements
```bash
GET /vessels/coastal-celebration
# OR
GET /vessels/316012824
```

```json
{
  "vesselName": "Coastal Celebration",
  "mmsi": 316012824,
  "date": "2025-11-17",
  "movements": []
}
```
**Status**: 200 OK

---

## 8. Next Steps After Implementation

### Phase 1: Validation (Immediate)
1. ✅ Implement endpoint
2. ✅ Test with known vessels (Salish Eagle, Salish Raven)
3. ✅ Verify movements match BC Ferries departures pages
4. ✅ Confirm destination deduction works correctly

### Phase 2: Integration (After Validation)
Once the endpoint is working correctly:

1. **Integrate into vessel database population**:
   - Use `ReconstructVesselRoute()` logic during scraping
   - Populate missing inter-island vessel data
   - This will fix the original "UNKNOWN" vessel problem

2. **Update BuildLegs function**:
   - Ensure it uses the enriched vessel database
   - Test multi-leg sailings to verify vessels are no longer "UNKNOWN"

3. **Deploy to production**:
   - Push changes to bc-ferries-api repository
   - Rebuild and deploy to 192.18.154.57
   - Verify sgi-ferries backend receives correct vessel data

### Phase 3: Optimization (Future)
1. Consider caching reconstructed routes if endpoint is slow
2. Add more vessel metadata (vessel type, capacity, image URLs)
3. Support date parameter to view historical routes
4. Add vessel schedule predictions based on historical patterns

---

## Implementation Notes

### Context from Previous Work
This endpoint builds on recent work to fix the vessel database composite key bug:
- Previously: `vesselDB[terminal][time] = vessel` (single vessel per time)
- Now: `vesselDB[terminal][time-destination] = vessel` (multiple vessels per time)

This fix enables proper vessel tracking across terminals.

### Data Flow
```
BC Ferries Departures Pages
    ↓ (scraping)
Vessel Database (in-memory map)
    ↓ (ReconstructVesselRoute)
Movements Array (sorted, enriched)
    ↓ (JSON response)
Client Application
```

### Assumptions
- Vessel database is built fresh on each scrape
- Departures pages show current day only
- Date field represents the day the vessel database was scraped (typically current day)
- Terminal codes in staticdata are complete and accurate
- MMSI mappings are manually maintained
- All times are scheduled times from timetable data, not actual or real-time estimates

### Known Limitations
- Only shows today's movements (no historical data)
- Scheduled arrival times are estimates (derived from next movement's scheduled departure time)
- Only provides scheduled times, not actual or estimated real-time data
- Requires vessel database to be populated (scraping must complete)
- Slug normalization uses simple title case (may not handle special cases like "BC" or "II")
- Vessel name casing may vary between vessel database and MMSI mapping (handled via case-insensitive comparison)
- No pagination (assumes reasonable number of movements per vessel)

---

## Appendix: Relevant Terminal Codes

For reference during testing:

| Code | Name | Service Area |
|------|------|--------------|
| TSA | Tsawwassen | Metro Vancouver |
| SWB | Swartz Bay | Victoria |
| FUL | Fulford Harbour | Southern Gulf Islands |
| SGI | Sturdies Bay | Southern Gulf Islands |
| POB | Otter Bay | Southern Gulf Islands |
| PVB | Village Bay | Southern Gulf Islands |
| PSB | Sturdies Bay | Southern Gulf Islands |
| PLH | Long Harbour | Southern Gulf Islands |
| HSB | Horseshoe Bay | Metro Vancouver |
| NAN | Nanaimo (Departure Bay) | Mid Island |
| LNG | Langdale | Sunshine Coast |
| BOW | Bowen Island | Sunshine Coast |

---

**Document End**

For questions or clarifications, refer to the conversation history or reach out to the implementation team.
