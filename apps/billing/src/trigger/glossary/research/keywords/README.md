# Keyword Research Feature

This module provides a comprehensive keyword research workflow that combines multiple data sources to generate rich keyword insights.
It's designed to provide contextual data for SEO optimization of an arbitrary glossary term.

## Workflow

1. **Input**: Takes a single glossary term as input
2. **Parallel Data Collection**: Simultaneously queries multiple sources:
   - Related Keywords (via massiveonlinemarketing.nl utilizing Google Keyword Planner)
   - Google Related Searches (via Serper.dev)
   - Google SERP Analysis (via Serper.dev)
   - Google Autosuggest Results (via Serper.dev)
3. **Data Enrichment**: Keywords from Google get enriched with keyword metrics via api.keywordseverywhere.com -- data reliability is questionable
4. **Deduplication**: Ensures unique keywords while preserving best data, we prioritize data from massiveonlinemarketing.nl the most as that comes from Google's Keyword Planner
5. **Result Aggregation**: Combines all sources into unified output format

## Folder Structure

```
research/keywords/
├── README.md                 # This documentation
├── _research-keywords.ts     # Parent task orchestrating the workflow
├── _keyword_research-test.ts # Test file for the entire workflow as well as each sub task
├── related-keywords.ts      # Scrape massiveonlinemarketing.nl for related keywords
├── serper-search.ts         # Query serper.dev's API for related searches & SERP
├── serper-autosuggest.ts    # Query serper.dev's API for autosuggestions
├── enrich-keywords.ts       # Enrich keywords with keyword metrics
```

## Testing

To run the tests:

```bash
pnpm -F billing dev
```

Go into Trigger's [Test Pane](https://cloud.trigger.dev/orgs/unkey-9e78/projects/billing-IzvK/env/dev/test) and run the `research_keywords_test` test with an empty payload.

To test only a subtask, run:

- `related_keywords_test`
- `serper_search_test`
- `serper_autosuggest_test`
- `enrich_keywords_test`

## Logic Overview

### Parent Task (`_research-keywords.ts`)

- Orchestrates the parallel execution of data collection tasks
- Handles deduplication and data normalization
- Manages error cases and partial failures
- Provides unified output format

### Child Tasks

1. **Related Keywords**
   - Primary source of keyword data
   - Provides high-quality metrics
   - Uses massiveonlinemarketing.nl's site to scrape related keywords
   - They have their data from Google Keyword Planner as the high_top_of_page_bid_micros data suggests

2. **SERP Analysis**
   - Extracts keywords from search results
   - Extracts related searches directly
   - Extracts keywords by having an LLM analyze the SERP's results

3. **Autosuggest**
   - Uses serper.dev's API to get autosuggestions

4. **Enrichment**
   - Adds volume/CPC/competition metrics
   - api.keywordseverywhere.com is used to get the metrics

## Sample Input/Output

### Input

```json
{
  "inputTerm": "MIME types"
}
```

### Output

```json
{
  "keywords": [
    {
      "keyword": "mime types",
      "volume": 33100,
      "cpc": 0,
      "competition": 2,
      "source": "massiveonlinemarketing.nl"
    },
    {
      "keyword": "mime types iis",
      "volume": 10,
      "cpc": 0,
      "competition": 0,
      "confidence": 1,
      "source": "autosuggest"
    }
  ],
  "metadata": {
    "totalKeywords": 124,
    "sources": {
      "relatedKeywords": 104,
      "serperSearch": 15,
      "serperAutosuggest": 10
    },
    "deduplication": {
      "total": 124,
      "skippedEnrichment": 11,
      "duplicatesRemoved": 0
    }
  }
}
```

## MVP Documentation

There's is /docs folder in this repo that contains the specs that were used to build the feature together with cursor and are kept there for reference.
