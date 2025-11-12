package db

import (
	"database/sql"
	"encoding/json"
	"log"

	"github.com/jeffcstock/bc-ferries-api/cmd/models"
	"github.com/lib/pq"
)

/*
 * GetCapacitySailings
 *
 * Retrieves all capacity route records from the database, including parsed sailing data.
 *
 * Queries the `capacity_routes` table and unmarshals the `sailings` JSON column
 * into a slice of `models.CapacitySailing` for each route.
 *
 * @return []models.CapacityRoute - a slice of capacity routes with their sailings
 */
func GetCapacitySailings() []models.CapacityRoute {
	var routes []models.CapacityRoute

	sqlStatement := `SELECT * FROM capacity_routes`

	rows, err := Conn.Query(sqlStatement)
	if err != nil {
		log.Printf("GetCapacitySailings: query failed: %v", err)
		return routes
	}
	defer rows.Close()

	for rows.Next() {
		var route models.CapacityRoute
		var sailings []uint8

		err := rows.Scan(&route.RouteCode, &route.FromTerminalCode, &route.ToTerminalCode, &route.Date, &route.SailingDuration, &sailings)
		if err != nil {
			log.Printf("GetCapacitySailings: row scan failed: %v", err)
			continue
		}

		var content []models.CapacitySailing
		if err := json.Unmarshal(sailings, &content); err != nil {
			log.Printf("GetCapacitySailings: JSON unmarshal failed: %v", err)
			continue
		}

		route.Sailings = content
		routes = append(routes, route)
	}

	if err := rows.Err(); err != nil {
		log.Printf("GetCapacitySailings: row iteration error: %v", err)
	}

	return routes
}

/*
 * GetNonCapacitySailings
 *
 * Retrieves all non-capacity route records from the database, including parsed sailing data.
 *
 * Queries the `non_capacity_routes` table and unmarshals the `sailings` JSON column
 * into a slice of `models.NonCapacitySailing` for each route.
 *
 * @return []models.NonCapacityRoute - a slice of non-capacity routes with their sailings
 */
func GetNonCapacitySailings() []models.NonCapacityRoute {
	var routes []models.NonCapacityRoute

	sqlStatement := `SELECT * FROM non_capacity_routes`

	rows, err := Conn.Query(sqlStatement)
	if err != nil {
		log.Printf("GetNonCapacitySailings: query failed: %v", err)
		return routes
	}
	defer rows.Close()

	for rows.Next() {
		var route models.NonCapacityRoute
		var sailings []uint8

		err := rows.Scan(&route.RouteCode, &route.FromTerminalCode, &route.ToTerminalCode, &route.Date, &route.SailingDuration, &sailings)
		if err != nil {
			log.Printf("GetNonCapacitySailings: row scan failed: %v", err)
			continue
		}

		var content []models.NonCapacitySailing
		if err := json.Unmarshal(sailings, &content); err != nil {
			log.Printf("GetNonCapacitySailings: JSON unmarshal failed: %v", err)
			continue
		}

		route.Sailings = content
		routes = append(routes, route)
	}

	if err := rows.Err(); err != nil {
		log.Printf("GetNonCapacitySailings: row iteration error: %v", err)
	}

	return routes
}

/*
 * GetCapacityRoutesInfo
 *
 * Retrieves capacity route metadata (without sailings) from the database.
 * Optionally filters by specific route codes if provided.
 *
 * @param routeCodes []string - optional list of route codes to filter by (empty slice = all routes)
 *
 * @return []models.CapacityRouteInfo - a slice of capacity route metadata
 */
func GetCapacityRoutesInfo(routeCodes []string) []models.CapacityRouteInfo {
	var routes []models.CapacityRouteInfo

	sqlStatement := `SELECT route_code, from_terminal_code, to_terminal_code, date, sailing_duration FROM capacity_routes`
	var rows *sql.Rows
	var err error

	if len(routeCodes) > 0 {
		sqlStatement += ` WHERE route_code = ANY($1)`
		rows, err = Conn.Query(sqlStatement, pq.Array(routeCodes))
	} else {
		rows, err = Conn.Query(sqlStatement)
	}

	if err != nil {
		log.Printf("GetCapacityRoutesInfo: query failed: %v", err)
		return routes
	}
	defer rows.Close()

	for rows.Next() {
		var route models.CapacityRouteInfo

		err := rows.Scan(&route.RouteCode, &route.FromTerminalCode, &route.ToTerminalCode, &route.Date, &route.SailingDuration)
		if err != nil {
			log.Printf("GetCapacityRoutesInfo: row scan failed: %v", err)
			continue
		}

		routes = append(routes, route)
	}

	if err := rows.Err(); err != nil {
		log.Printf("GetCapacityRoutesInfo: row iteration error: %v", err)
	}

	return routes
}

/*
 * GetNonCapacityRoutesInfo
 *
 * Retrieves non-capacity route metadata (without sailings) from the database.
 * Optionally filters by specific route codes if provided.
 *
 * @param routeCodes []string - optional list of route codes to filter by (empty slice = all routes)
 *
 * @return []models.NonCapacityRouteInfo - a slice of non-capacity route metadata
 */
func GetNonCapacityRoutesInfo(routeCodes []string) []models.NonCapacityRouteInfo {
	var routes []models.NonCapacityRouteInfo

	sqlStatement := `SELECT route_code, from_terminal_code, to_terminal_code, date, sailing_duration FROM non_capacity_routes`
	var rows *sql.Rows
	var err error

	if len(routeCodes) > 0 {
		sqlStatement += ` WHERE route_code = ANY($1)`
		rows, err = Conn.Query(sqlStatement, pq.Array(routeCodes))
	} else {
		rows, err = Conn.Query(sqlStatement)
	}

	if err != nil {
		log.Printf("GetNonCapacityRoutesInfo: query failed: %v", err)
		return routes
	}
	defer rows.Close()

	for rows.Next() {
		var route models.NonCapacityRouteInfo

		err := rows.Scan(&route.RouteCode, &route.FromTerminalCode, &route.ToTerminalCode, &route.Date, &route.SailingDuration)
		if err != nil {
			log.Printf("GetNonCapacityRoutesInfo: row scan failed: %v", err)
			continue
		}

		routes = append(routes, route)
	}

	if err := rows.Err(); err != nil {
		log.Printf("GetNonCapacityRoutesInfo: row iteration error: %v", err)
	}

	return routes
}
