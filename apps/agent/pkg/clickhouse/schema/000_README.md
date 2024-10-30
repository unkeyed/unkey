
# ClickHouse Table Naming Conventions

This document outlines the naming conventions for tables and materialized views in our ClickHouse setup. Adhering to these conventions ensures consistency, clarity, and ease of management across our data infrastructure.

## General Rules

1. Use lowercase letters and separate words with underscores.
2. Avoid ClickHouse reserved words and special characters in names.
3. Be descriptive but concise.

## Table Naming Convention

Format: `[prefix]_[domain]_[description]_[version]`

### Prefixes

- `raw_`: Input data tables
- `mv_`: Materialized views
- `tmp_{yourname}_`: Temporary tables for experiments, add your name, so it's easy to identify ownership.

### Domain/Category

Include the domain or category of the data when applicable.

Examples:
- `keys`
- `audit`
- `user`
- `gateway`

### Versioning

- Version numbers: `_v1`, `_v2`, etc.

### Aggregation Suffixes

For aggregated or summary tables, use suffixes like:
- `_per_day`
- `_per_month`
- `_summary`

## Materialized View Naming Convention

Format: `mv_[description]_[aggregation]`

- Always prefix with `mv_`
- Include a description of the view's purpose
- Add aggregation level if applicable

## Examples

1. Raw Data Table:
   `raw_sales_transactions_v1`

2. Materialized View:
   `mv_active_users_per_day_v2`

3. Temporary Table:
   `tmp_andreas_user_analysis_v1`

4. Aggregated Table:
   `mv_sales_summary_per_hour_v1`

## Consistency Across Related Objects

Maintain consistent naming across related tables, views, and other objects:

- `raw_user_activity_v1`
- `mv_user_activity_per_day_v1`

By following these conventions, we ensure a clear, consistent, and scalable naming structure for our ClickHouse setup.
