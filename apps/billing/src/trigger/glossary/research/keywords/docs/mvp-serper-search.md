# MVP Definition: serperSearchTask v1

## Goal

Integrate with Serper.dev's search API to extract related keywords from search results for a given input term.

## Current Task

Implement and test the Serper.dev search integration to extract relevant keywords from Google search results.

## Input

```typescript
{
  inputTerm: string;
}
```

## Core Steps

1. Receive `inputTerm` from payload
2. Make API request to Serper.dev's `/search` endpoint:

   ```
   POST https://google.serper.dev/search
   Headers: {
     'X-API-KEY': process.env.SERPER_API_KEY,
     'Content-Type': 'application/json'
   }
   Body: {
     "q": inputTerm
   }
   ```

3. Return the full Serper response with type safety using Zod schema

## Output Schema

```typescript
// Serper API Response Schema
const SerperSearchResultSchema = z.object({
  searchParameters: z.object({
    q: z.string(),
    gl: z.string(),
    hl: z.string(),
    num: z.number(),
    type: z.string()
  }),
  searchInformation: z.object({
    totalResults: z.string(),
    formattedTotalResults: z.string(),
    formattedSearchTime: z.string()
  }),
  organic: z.array(z.object({
    title: z.string(),
    link: z.string(),
    snippet: z.string(),
    position: z.number(),
    sitelinks: z.array(z.object({
      title: z.string(),
      link: z.string()
    })).optional()
  })),
  peopleAlsoAsk: z.array(z.object({
    question: z.string(),
    snippet: z.string(),
    title: z.string(),
    link: z.string()
  })).optional(),
  relatedSearches: z.array(z.object({
    query: z.string()
  })).optional()
});

// Task Output
{
  inputTerm: string;
  result: z.infer<typeof SerperSearchResultSchema>;
}
```

## Error Handling

Use `AbortTaskRunError` for:

- Missing or invalid API key
- Invalid input parameters
- Failed API requests

## Testing (`_keyword-research-test.ts`)

Create test cases:

1. `serperSearchBasicTest`:
   - Input: "MIME types"
   - Verify successful API response
   - Check response matches Zod schema
   - Verify searchParameters.q matches input
   - Check for presence of organic results

2. `serperSearchErrorTest`:
   - Test with empty input
   - Test with invalid API key

## Implementation Notes

- Store Serper API key in environment variables
- Use TypeScript for type safety
- Use Zod for runtime type validation of API response

## ✅ Success Criteria

1. Successfully calls Serper API and receives results
2. Validates response against Zod schema
3. Properly handles error cases
4. Passes all test cases
5. Maintains type safety

## Dependencies

- Serper.dev API key (environment variable: `SERPER_API_KEY`)
- Zod for schema validation

## Next Steps

1. Implement base task with core functionality
2. Add test suite
3. Integrate with parent keyword research task
