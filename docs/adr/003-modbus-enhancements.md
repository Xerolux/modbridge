# ADR 003: Modbus Register Enhancements

## Status
Accepted

## Context
Raw Modbus registers have several limitations:
- Values need scaling and unit conversion
- No semantic meaning (register 123 vs "temperature")
- No caching strategy for read-heavy workloads
- No historical data tracking

## Decision
Implement three complementary features:
1. **Register Mapping** - Transform and annotate registers
2. **Register Caching** - Reduce device load with TTL-based cache
3. **Time-Series Data** - Historical tracking and trending

## Consequences
**Positive:**
- Reduced Modbus device load (caching)
- Better data semantics (mappings)
- Historical analysis and trending
- Data export capabilities

**Negative:**
- Increased memory usage
- Complexity in cache invalidation
- Storage growth for time-series data

## Implementation
- Packages: `pkg/mapping`, `pkg/caching`, `pkg/timeseries`
- Database tables: `register_mappings`, `register_cache`, `timeseries_data`
- API: New endpoints for mappings, cache management
- Frontend: Mapping configuration UI
