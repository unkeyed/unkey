# MVP Definition: serperSearchTask v1

## Goal

Extract a comprehensive list of keywords from Google search results by utilizing Serper.dev's search endpoint. The task combines direct "Related Searches" from Google with an LLM analysis of the top 10 organic search results to identify relevant keywords, providing a dual-source approach to keyword discovery.

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
     "q": inputTerm,
     "gl": "us",
     "hl": "en",
     "num": 10,
     "type": "search"
   }
   ```

3. Extract keywords using two methods:
   - Direct inclusion of `relatedSearches` queries
   - LLM analysis of organic search results using a carefully crafted prompt:

   ```typescript
   const keywords = await generateObject({
     model: google("gemini-2.0-flash-lite-preview-02-05"),
     schema: keywordExtractionSchema,
     output: "array",
     prompt: `You are a keyword research expert. Analyze these search results and extract relevant keywords.

     Guidelines:
     - Focus on identifying distinct concepts and terms
     - Include both broad and specific keywords
     - Avoid duplicates and variations of the same term
     - Consider technical terms, common usage, and related technologies
     - Do not include generic words or stop words
     - Each keyword should be directly related to the topic

     Search Results:
     [Search results content here]

     Return an array of keywords that best represent the topic and its related concepts.`,
   });
   ```

## Output Schema

```typescript
// Serper API Parsed Response Schema
// This represents the validated response after a successful API call
const SerperSearchResultSchema = z.object({
  organic: z.array(z.object({
    title: z.string(),
    link: z.string(),
    snippet: z.string(),
    position: z.number().optional(),
    sitelinks: z.array(z.object({
      title: z.string(),
      link: z.string()
    })).optional()
  })),
  peopleAlsoAsk: z.array(z.object({
    question: z.string(),
    snippet: z.string(),
    title: z.string().optional(),
    link: z.string()
  })).optional(),
  relatedSearches: z.array(z.object({
    query: z.string()
  })).optional()
});

// Keyword Schema
const KeywordSchema = z.object({
  keyword: z.string(),
  source: z.enum(['related_search', 'llm_extracted']),
  confidence: z.number().min(0).max(1).optional(),
  context: z.string().optional() // Additional context about why this keyword was selected
});

// Task Output
{
  inputTerm: string;
  searchResult: z.infer<typeof SerperSearchResultSchema>;
  keywords: z.array(z.infer<typeof KeywordSchema>>;
}
```

## Error Handling

Use `AbortTaskRunError` for:

- Missing or invalid API key
- Invalid input parameters
- Failed API requests
- Failed LLM calls
- Invalid keyword extraction results

## Testing (`_keyword-research-test.ts`)

Update test cases:

1. `serperSearchBasicTest`:
   - Input: "MIME types"
   - Verify successful API response
   - Check response matches Zod schema
   - Verify presence of organic results
   - Validate extracted keywords structure
   - Check for relevant keywords from both sources
   - Ensure minimum number of keywords returned

2. `serperSearchEmptyInputTest`:
   - Test with empty input
   - Verify error handling

## Implementation Notes

- Store Serper API key in environment variables
- Use TypeScript for type safety
- Use Zod for runtime type validation
- Implement proper error handling for both API and LLM calls
- Consider rate limiting and retry strategies

## ✅ Success Criteria

1. Successfully calls Serper API and receives results
2. Extracts keywords from relatedSearches
3. Successfully uses LLM to extract additional keywords
4. Combines and deduplicates keywords from both sources
5. Maintains proper typing and validation
6. Passes all test cases
7. Handles errors gracefully

## Dependencies

- Serper.dev API key (environment variable: `SERPER_API_KEY`)
- Vercel AI SDK for LLM integration
- Zod for schema validation

## Next Steps

1. Update task implementation with LLM integration
2. Update test suite with new keyword validation
3. Integrate with parent keyword research task
4. Add telemetry for LLM calls
5. Consider caching strategies for similar searches
