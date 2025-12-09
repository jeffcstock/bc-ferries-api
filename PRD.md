# BC Ferries Consumer App - Product Requirements Document

## Overview

A mobile/web application that consumes the BC Ferries API to display route information, sailing schedules, and real-time vessel tracking.

---

## Feature Requirements

### 1. Routes Page

**Purpose**: Display all available ferry routes for user selection.

**Requirements**:
- List all routes (e.g., "TSA → POB", "SWB → PST")
- Show route metadata: origin terminal name, destination terminal name
- Tapping a route navigates to the Route Detail Page

**API Coverage**: ✅ AVAILABLE
- `GET /v2/routes/capacity` - capacity route metadata
- `GET /v2/routes/noncapacity` - non-capacity route metadata

---

### 2. Route Detail Page

**Purpose**: Show all sailings for a selected route with status-appropriate information.

**Requirements**:
- Display route header (e.g., "Tsawwassen → Otter Bay")
- Show number of sailings for the day
- List all sailings with status-specific information (see below)

**API Coverage**: ✅ AVAILABLE
- `GET /v2/capacity/:routeCode` - single capacity route with sailings
- `GET /v2/noncapacity/:routeCode` - single non-capacity route with sailings

---

### 3. Sailing Display Requirements by Status

#### 3.1 Past Sailings (Arrived)

| Requirement | Field Needed | API Status |
|-------------|--------------|------------|
| Scheduled departure time | `departureTime` | ✅ Available |
| Scheduled arrival time | `arrivalTime` | ✅ Available |
| Actual departure time | `actualDepartureTime` | ❌ **NOT AVAILABLE** |
| Actual arrival time | `actualArrivalTime` | ❌ **NOT AVAILABLE** |
| Minutes late/early | Calculated from actual vs scheduled | ❌ **NOT AVAILABLE** |
| Vessel name | `vesselName` | ✅ Available |

**Gap Analysis**: The API does not currently track or store actual departure/arrival times. Only scheduled times are available.

---

#### 3.2 Current Sailings (Underway)

| Requirement | Field Needed | API Status |
|-------------|--------------|------------|
| Scheduled departure time | `departureTime` | ✅ Available |
| Scheduled arrival time | `arrivalTime` | ✅ Available |
| Estimated departure (if not left) | `estimatedDepartureTime` | ❌ **NOT AVAILABLE** |
| Actual departure (if departed) | `actualDepartureTime` | ❌ **NOT AVAILABLE** |
| Estimated arrival time | `arrivalTime` (may be ETA) | ⚠️ Partial - unclear if this updates |
| Vessel name | `vesselName` | ✅ Available |
| Current vessel position | Live lat/lon | ❌ **NOT AVAILABLE** |
| Current leg information | Which leg vessel is on | ❌ **NOT AVAILABLE** |

**Gap Analysis**:
- No real-time vessel position tracking (would require AIS/MMSI integration)
- No distinction between scheduled vs estimated times
- No actual departure time tracking
- Vessel status field exists (`vesselStatus`) but only contains "On Schedule", "Delayed", or cancellation reason - not detailed timing

---

#### 3.3 Future Sailings

| Requirement | Field Needed | API Status |
|-------------|--------------|------------|
| Scheduled departure time | `departureTime` | ✅ Available |
| Scheduled arrival time | `arrivalTime` | ✅ Available |
| Estimated departure time | `estimatedDepartureTime` | ❌ **NOT AVAILABLE** |
| Estimated arrival time | `estimatedArrivalTime` | ❌ **NOT AVAILABLE** |
| Minutes late/early estimate | Calculated | ❌ **NOT AVAILABLE** |
| Vessel name | `vesselName` | ✅ Available |

**Gap Analysis**: No estimated times separate from scheduled times. The `vesselStatus` field indicates "Delayed" but doesn't provide the expected delay duration.

---

## Data Gap Summary

### Currently Available
| Data | Source |
|------|--------|
| All routes (capacity + non-capacity) | `/v2/routes/*` |
| Sailings per route | `/v2/capacity/:code`, `/v2/noncapacity/:code` |
| Scheduled departure/arrival times | `departureTime`, `arrivalTime` |
| Sailing status | `sailingStatus`: future, current, past, cancelled |
| Vessel name | `vesselName` |
| Vessel status text | `vesselStatus`: "On Schedule", "Delayed", etc. |
| Capacity fill percentages | `fill`, `carFill`, `oversizeFill` |
| Terminal coordinates | Available in leg data and vessel routes |
| Vessel MMSI numbers | `/vessels/:id` endpoint |

### Missing - Required for Full Feature Set

| Data | Priority | Notes |
|------|----------|-------|
| **Actual departure time** | HIGH | When ferry actually left |
| **Actual arrival time** | HIGH | When ferry actually arrived |
| **Estimated departure time** | HIGH | For delayed sailings |
| **Estimated arrival time** | HIGH | ETA for delayed/current sailings |
| **Delay duration (minutes)** | HIGH | How late the sailing is |
| **Real-time vessel position** | MEDIUM | Lat/lon of vessel in transit |
| **Current leg indicator** | MEDIUM | Which leg of multi-stop route |

---

## Proposed API Enhancements

### Option A: Enhance Existing Sailing Object

Add fields to `CapacitySailing`:

```json
{
  "departureTime": "7:00 am",           // scheduled
  "arrivalTime": "8:35 am",             // scheduled
  "actualDepartureTime": "7:05 am",     // NEW: actual (null if not departed)
  "actualArrivalTime": "8:42 am",       // NEW: actual (null if not arrived)
  "estimatedDepartureTime": "7:05 am",  // NEW: ETA for departure
  "estimatedArrivalTime": "8:40 am",    // NEW: ETA for arrival
  "delayMinutes": 5,                    // NEW: calculated delay
  "sailingStatus": "current",
  "vesselName": "Spirit of British Columbia",
  "vesselStatus": "Departed 5 min late"
}
```

### Option B: New Vessel Position Endpoint

Add real-time vessel tracking:

```
GET /vessels/:id/position
```

```json
{
  "vesselName": "Spirit of British Columbia",
  "mmsi": 316001268,
  "timestamp": "2025-12-03T15:30:00Z",
  "position": {
    "lat": 48.95,
    "lon": -123.25
  },
  "heading": 180,
  "speed": 18.5,
  "currentSailing": {
    "routeCode": "TSASWB",
    "departureTerminal": "TSA",
    "arrivalTerminal": "SWB",
    "scheduledDeparture": "3:00 pm",
    "actualDeparture": "3:05 pm",
    "estimatedArrival": "4:40 pm",
    "currentLeg": 1,
    "totalLegs": 1
  }
}
```

**Note**: This would require AIS data integration (e.g., MarineTraffic API, AISHub, or direct AIS receiver).

---

## Implementation Phases

### Phase 1: MVP with Available Data
- Routes list page
- Route detail page with sailings
- Show scheduled times, vessel name, status text
- Show capacity percentages
- Indicate cancelled sailings

### Phase 2: Enhanced Timing Data
- Scrape/parse estimated times from BC Ferries (if available on their site)
- Add delay tracking fields to API
- Calculate and display delay minutes

### Phase 3: Real-Time Vessel Tracking
- Integrate AIS data source
- Add vessel position endpoint
- Display vessel on map during sailing
- Show current leg for multi-stop routes

---

## BC Ferries Data Source Analysis

### What BC Ferries Provides (Current Conditions Page)

The BC Ferries website shows:
- Scheduled departure time
- Vessel name
- Status: "On Time", "Delayed", "Cancelled"
- Capacity percentages

**Does NOT clearly show**:
- Actual departure/arrival times
- Estimated times with specific minutes
- Real-time vessel position

### Potential Data Sources for Enhancement

1. **BC Ferries Website Scraping**
   - Parse "Delayed" status for timing info
   - Check if ETA is shown anywhere

2. **AIS (Automatic Identification System)**
   - Real-time vessel positions
   - Speed and heading
   - Requires: MarineTraffic API ($), AISHub, or own AIS receiver

3. **BC Ferries API** (if available)
   - Check if BC Ferries has an official API
   - May provide more detailed timing data

---

## Questions to Resolve

1. Does BC Ferries show estimated times anywhere on their site that we could scrape?
2. Is real-time vessel tracking a hard requirement or nice-to-have?
3. For past sailings, is showing "was X minutes late" critical, or is scheduled time sufficient?
4. Budget for AIS data integration?

---

## Appendix: Current API Response Examples

### Capacity Sailing (Current)
```json
{
  "id": "TSASWB-2025-12-03-0700",
  "time": "7:00 am",
  "arrivalTime": "8:35 am",
  "sailingStatus": "past",
  "fill": 45,
  "carFill": 60,
  "oversizeFill": 30,
  "vesselName": "Spirit of British Columbia",
  "vesselStatus": "On Schedule"
}
```

### Vessel Route (Current)
```json
{
  "vesselName": "Salish Eagle",
  "mmsi": 316030626,
  "date": "2025-12-03",
  "movements": [
    {
      "origin": {"code": "TSA", "name": "Tsawwassen", "lat": 49.007, "lon": -123.131},
      "destination": {"code": "PSB", "name": "Sturdies Bay", "lat": 48.877, "lon": -123.315},
      "scheduledDepartureTime": "07:00",
      "scheduledArrivalTime": "07:55"
    }
  ]
}
```
