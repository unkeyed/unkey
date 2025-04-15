# MVP Definition: enrichKeywordsTask v1

## Goal

Enrich keyword data from Serper tasks with volume, CPC, and competition metrics using the Keywords Everywhere API. This task takes keywords from both serper-search and serper-autosuggest tasks and adds valuable SEO metrics to help evaluate keyword potential.

## Input

```typescript
// Input is inferred from either Serper task's output
type TaskInput = ExtractFromTrigger<typeof serperAutosuggestTask> | ExtractFromTrigger<typeof serperSearchTask>;
```

## Core Steps

1. Receive task output from either Serper task
2. Extract keywords array from the input
3. Make API request to Keywords Everywhere API:

   ```typescript
   const params = new URLSearchParams();
   // Add each keyword as a separate kw[] parameter
   keywords.forEach(kw => params.append("kw[]", kw));
   params.append("country", "us");
   params.append("currency", "usd");
   params.append("dataSource", "gkp");

   fetch("https://api.keywordseverywhere.com/v1/get_keyword_data", {
     method: "POST",
     headers: {
       'Authorization': `Bearer ${process.env.KEYWORDS_EVERYWHERE_API_KEY}`,
     },
     body: params
   });
   ```

4. Process keywords in batches (max 100 per request)
5. Transform API response into our standardized format
6. Track API credit usage in metadata

## Output Schema

```typescript
// Keywords Everywhere API Response Schema
const KeywordsEverywhereResponseSchema = z.object({
  data: z.array(z.object({
    keyword: z.string(),
    vol: z.number(),
    cpc: z.object({
      currency: z.string(),
      value: z.string()
    }),
    competition: z.number(),
    trend: z.array(z.object({
      month: z.string(),
      year: z.number(),
      value: z.number()
    })).optional().default([])
  })),
  credits: z.number(),
  credits_consumed: z.number(),
  time: z.number()
});

// Reuse the RelatedKeyword schema from related-keywords task
import { RelatedKeywordSchema } from "./related-keywords";

// Task Output
// Task Output
type TaskOutput = {
  enrichedKeywords: z.infer<typeof z.array(RelatedKeywordSchema)>;
  metadata: {
    totalProcessed: number;
    creditsUsed: number;
    creditsRemaining: number;
    processingTime: number;
    timestamp: string;
  };
}
```

## Error Handling

Use `AbortTaskRunError` for:

- Missing or invalid API key
- Invalid input parameters
- Failed API requests
- Insufficient API credits

Use batch processing with partial success:

- Skip individual failed keywords
- Continue processing remaining keywords
- Include failure details in metadata

## Testing (`_keyword-research-test.ts`)

Add test cases:

1. `enrichKeywordsBasicTest`:
   - Input: Output from serperSearchTask with "MIME types"
   - Verify successful API response
   - Check response matches schema
   - Validate enriched data structure
   - Verify all metrics are present
   - Check source attribution is preserved

2. `enrichKeywordsAutosuggestTest`:
   - Input: Output from serperAutosuggestTask
   - Verify source is correctly preserved
   - Check data enrichment

3. `enrichKeywordsLargeTest`:
   - Test with 100+ keywords
   - Verify batch processing
   - Check all keywords are processed
   - Validate credit usage tracking

4. `enrichKeywordsInvalidTest`:
   - Test with invalid keywords
   - Verify error handling
   - Check partial success handling

Note: Need sample data from actual Serper task runs for accurate testing.

## Implementation Notes

- Store Keywords Everywhere API key in environment variables
- Use TypeScript for type safety
- Use Zod for runtime type validation
- Implement efficient batch processing
- Track and log API credit usage
- Consider implementing caching for frequently requested keywords
- Follow the .mdc rules for typescript.mdc, avoid-nesting.mdc, and try-catch-typescript.mdc
- Transform CPC string value to number (strip currency symbol)
- Simplify trend data to just the values array

## ✅ Success Criteria

1. Successfully calls Keywords Everywhere API
2. Processes keywords in efficient batches
3. Enriches keywords with all required metrics
4. Maintains proper typing and validation
5. Handles API errors gracefully
6. Tracks API credit usage
7. Passes all test cases
8. Ready for integration with parent task
9. Preserves source attribution from Serper tasks

## Dependencies

- Keywords Everywhere API key (environment variable: `KEYWORDS_EVERYWHERE_API_KEY`)
- Zod for schema validation
- Caching mechanism (optional)

## Next Steps

1. Implement task based on this MVP
2. Create and run test suite
3. Wait for user approval and commit
4. Make tests pass
