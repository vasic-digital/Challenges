# SQL Definitions

## Module Overview

This module does not directly use SQL databases for its core functionality.

## Database Usage

### Primary Storage
- **Technology**: In-memory challenge registry, filesystem storage for scripts
- **Purpose**: Challenge execution, assertion validation, reporting, monitoring, metrics
- **Schema**: No SQL schema is required

### Optional SQL Integration
- Challenge results and metrics can be stored in SQL databases for long-term analysis
- User flow test results may be persisted to SQL for trend analysis
- Assertion engine evaluations can be logged to SQL audit tables
- Plugin system v2.0.0 metadata may be stored in SQL

## Related Modules

For SQL database functionality, see the [Database module](../Database/README.md).

## Migration Notes

If SQL support is added in the future:
1. Create migration scripts in `migrations/` directory
2. Follow versioned migration pattern (`001_initial.sql`, `002_add_feature.sql`)
3. Use the `digital.vasic.database` module for database operations
4. Update this document with schema definitions