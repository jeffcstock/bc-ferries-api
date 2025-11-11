# Deployment Guide - API Changes (Date & Sailing ID)

This guide covers deploying the API changes that add:
- `date` field at the route level
- `id` field for each sailing

## Summary of Changes

### API Response Changes (Backward Compatible ✅)

**Before:**
```json
{
  "routeCode": "TSAPOB",
  "fromTerminalCode": "TSA",
  "toTerminalCode": "POB",
  "sailingDuration": "4h 40m",
  "sailings": [
    {
      "time": "7:00 am",
      "arrivalTime": "11:40 am",
      ...
    }
  ]
}
```

**After:**
```json
{
  "date": "2025-11-11",
  "routeCode": "TSAPOB",
  "fromTerminalCode": "TSA",
  "toTerminalCode": "POB",
  "sailingDuration": "4h 40m",
  "sailings": [
    {
      "id": "TSAPOB-2025-11-11-0700",
      "time": "7:00 am",
      "arrivalTime": "11:40 am",
      ...
    }
  ]
}
```

## Deployment Steps

### Step 1: Commit and Push Changes

```bash
# Review changes
git status
git diff

# Commit changes
git add .
git commit -m "Add date and id fields to API response

- Add date field to route structs (ISO 8601 format)
- Add unique id field to sailing structs (format: routeCode-date-time24h)
- Implement helper functions for time conversion and ID generation
- Update database schema to include date column
- Update scraper functions to generate dates and IDs
- Update database queries to handle new date field

This is a backward-compatible change that adds stability for future AIS integration."

# Push to GitHub
git push origin master
```

### Step 2: SSH to Server

```bash
ssh bc-ferries-api
cd ~/bc-ferries-api
```

### Step 3: Run Database Migration

**IMPORTANT:** Run the migration BEFORE deploying the new code to avoid errors.

```bash
# Connect to PostgreSQL container
docker exec -it bc-ferries-api-db-1 psql -U bcferries_prod bcferries

# Run the migration SQL
ALTER TABLE capacity_routes
ADD COLUMN date DATE NOT NULL DEFAULT CURRENT_DATE;

ALTER TABLE non_capacity_routes
ADD COLUMN date DATE NOT NULL DEFAULT CURRENT_DATE;

# Verify the changes
\d capacity_routes
\d non_capacity_routes

# Exit psql
\q
```

**Alternative:** Use the migration file directly

```bash
# Pull latest code first
git pull origin master

# Run migration from file
cat migration.sql | docker exec -i bc-ferries-api-db-1 psql -U bcferries_prod bcferries
```

### Step 4: Deploy New Code

```bash
# Use the deployment script
./deploy.sh

# OR manually:
git pull origin master
docker compose down
docker compose up -d --build

# Monitor logs
docker compose logs -f api
```

### Step 5: Verify Deployment

```bash
# Test health check
curl http://192.18.154.57/healthcheck/

# Test non-capacity routes (should include date and id fields)
curl http://192.18.154.57/v2/noncapacity/ | jq '.'

# Test capacity routes (should include date and id fields)
curl http://192.18.154.57/v2/capacity/ | jq '.'

# Check specific sailing IDs
curl http://192.18.154.57/v2/noncapacity/ | jq '.routes[0].sailings[0].id'
# Should output something like: "TSAPOB-2025-11-11-0700"

# Verify date field
curl http://192.18.154.57/v2/noncapacity/ | jq '.routes[0].date'
# Should output today's date like: "2025-11-11"
```

### Step 6: Monitor for Errors

```bash
# Watch API logs for any errors
docker compose logs -f api

# Check for database errors
docker compose logs db | grep ERROR

# Verify scraper is working
# Wait for next scheduled scrape (check cron settings)
# Or manually trigger if you have a trigger endpoint
```

## Rollback Plan

If issues arise, you can quickly rollback:

### Rollback Code

```bash
# On server
cd ~/bc-ferries-api

# Revert to previous commit
git log --oneline -5  # Find the previous commit hash
git checkout <previous-commit-hash>

# Rebuild and restart
docker compose down
docker compose up -d --build
```

### Rollback Database (Optional)

The date column has a default value, so old code can still work. However, if you want to remove it:

```bash
docker exec -it bc-ferries-api-db-1 psql -U bcferries_prod bcferries

ALTER TABLE capacity_routes DROP COLUMN date;
ALTER TABLE non_capacity_routes DROP COLUMN date;

\q
```

## Testing Checklist

- [ ] Database migration completed successfully
- [ ] API starts without errors
- [ ] `/v2/capacity/` returns data with `date` field
- [ ] `/v2/noncapacity/` returns data with `date` field
- [ ] Sailings have `id` field in format `ROUTECODE-YYYY-MM-DD-HHMM`
- [ ] Date reflects current Pacific Time date
- [ ] Sailing IDs are unique within each route
- [ ] No errors in API logs
- [ ] Existing API consumers still work (backward compatible)

## Notes

- **Backward Compatibility**: These changes are additive only. Existing API consumers will ignore the new fields.
- **Date Format**: Uses ISO 8601 format (`YYYY-MM-DD`) in Pacific Time
- **ID Format**: `{routeCode}-{date}-{time24h}` e.g., `TSAPOB-2025-11-11-0700`
- **Time Zone**: All dates use `America/Vancouver` timezone (Pacific Time)
- **Default Value**: Database uses `CURRENT_DATE` as default for the date column

## Future Enhancements

With these stable IDs in place, you can now:
- Build AIS vessel tracking correlation
- Create historical sailing records
- Track delays and on-time performance
- Build analytics dashboards
- Cache individual sailings reliably

## Questions?

If you encounter issues:
1. Check API logs: `docker compose logs -f api`
2. Check database logs: `docker compose logs db`
3. Verify database schema: `docker exec -it bc-ferries-api-db-1 psql -U bcferries_prod bcferries -c "\d capacity_routes"`
4. Test time conversion: Ensure times like "7:00 am (Tomorrow)" are handled correctly
