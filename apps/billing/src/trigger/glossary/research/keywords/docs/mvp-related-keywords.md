# MVP Definition: relatedKeywordsTask v1

## Goal

Scrape the massiveonlinemarketing.nl website to extract related keywords for a given input term, along with their volume, CPC, and competition data.

## Current Task

Implement and test the massiveonlinemarketing.nl scraping functionality to extract related keywords with their metrics.

## Input

- `{ inputTerm: string }`

## Core Steps

1. Receive `inputTerm` from payload.
2. Construct the URL: `https://www.massiveonlinemarketing.nl/en/tools/keyword-research/${encodeURIComponent(inputTerm)}`
3. Make an HTTP request to fetch the page content.
4. Parse the HTML to extract the data from the script tag:
   - Look for a script tag containing `self.__next_f.push`
   - Within that, find the section containing `keywordData`
   - Extract and parse the JSON data which is escaped inside the script tag
5. Transform the extracted data into our structured output format.
6. Return the structured data.

## Output

```typescript
{
  inputTerm: string;
  keywordIdeas: Array<{
    keyword: string;
    avg_monthly_searches: number;
    competition: number;
    competition_index: number;
    high_top_of_page_bid_micros: number;
    low_top_of_page_bid_micros: number;
  }>;
}
```

## Error Handling

- If the website is unreachable, use `AbortTaskRunError` to return a clear error message about connection failure
- If the term doesn't return results, return a clear error message
- If parsing fails, use `AbortTaskRunError` with details about the parsing failure - we should not return partial results
- Rely on Trigger.dev's `maxAttempts: 3` for retry logic on temporary failures

## Testing (`_keyword-research-test.ts`)

- Create a test case called `relatedKeywordsBasicTest`
- To optimize testing and avoid multiple HTTP requests:
  - Run the scraping task once with a common test term like `"MIME types"`
  - Store the result for use across multiple validation test cases
  - Implement multiple validation functions that check different aspects of the result

Validation checks:

- Verify `result.ok === true`
- Verify `result.output.inputTerm === "MIME types"`
- Verify `result.output.keywords` is an array with length > 0
- Check that each keyword object has the required fields (keyword, volume, cpc, competition)
- Verify some keywords are topically related to "MIME types"

## Implementation Notes

- The massiveonlinemarketing.nl site uses Next.js server components to include data inside script tags
- The data we need is embedded in a script tag with `self.__next_f.push` containing `keywordData`
- We preserve the original data structure from the source to maintain data fidelity
- The implementation successfully handles:
  - Finding and parsing escaped JSON in script tags
  - Robust error handling for missing or malformed data
  - Type safety through Zod schema validation
  - Retries for transient failures (max 3 attempts)

## ✅ Completed Tasks

- Basic implementation
- Error handling
- Input validation
- Output schema definition
- Test implementation with:
  - Basic success case ("MIME types")
  - Error case (empty input)
  - Structure validation
  - Content relevance checks

## Outstanding Tasks (Future MVPs)

1. **Serper Search (`serper-search.ts`)**: Move and simplify the existing Serper `/search` functionality to extract related search keywords.
2. **Serper Autosuggest (`serper-autosuggest.ts`)**: Implement integration with Serper's `/autosuggest` endpoint.
3. **Enrich Keywords (`enrich-keywords.ts`)**: Implement integration with Keywords Everywhere API to enrich keywords from Serper endpoints with volume, CPC, and competition data.
4. **Parent Task (`_research-keywords.ts`)**: Coordinate the execution of all subtasks and compile the final comprehensive keyword list.
