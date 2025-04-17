# MVP Definition: serperAutosuggestTask v1

## Goal

Utilize Serper.dev's autosuggest endpoint to extract Google's autocomplete suggestions for a given input term. These suggestions are direct from Google and thus have high confidence as keyword ideas.

## Input

```typescript
{
  inputTerm: string;
}
```

## Core Steps

1. Receive `inputTerm` from payload
2. Make API request to Serper.dev's `/autocomplete` endpoint:

   ```
   POST https://google.serper.dev/autocomplete
   Headers: {
     'X-API-KEY': process.env.SERPER_API_KEY,
     'Content-Type': 'application/json'
   }
   Body: {
     "q": inputTerm,
     "gl": "us",
     "hl": "en"
   }
   ```

3. Transform autosuggest results into our standard keyword format:
   - Set source as "autosuggest"
   - Set confidence to 1.0 (direct from Google)
   - Add context about being a Google autocomplete suggestion

## Output Schema

```typescript
// Serper Autosuggest Response Schema
const SerperAutosuggestResultSchema = z.object({
  queries: z.array(z.string())
});

// Using our existing KeywordSchema from serper-search
const KeywordSchema = z.object({
  keyword: z.string(),
  source: z.enum(['related_search', 'llm_extracted', 'autosuggest']),
  confidence: z.number().min(0).max(1).optional(),
  context: z.string().optional()
});

// Task Output
{
  inputTerm: string;
  searchResult: z.infer<typeof SerperAutosuggestResultSchema>;
  keywords: z.array(z.infer<typeof KeywordSchema>);
}
```

## Error Handling

Use `AbortTaskRunError` for:

- Missing or invalid API key
- Invalid input parameters
- Failed API requests
- Empty or invalid response format

## Testing (`_keyword-research-test.ts`)

Add test cases:

1. `serperAutosuggestBasicTest`:
   - Input: "MIME types"
   - Verify successful API response
   - Check response matches Zod schema
   - Verify presence of autocomplete suggestions
   - Validate transformed keywords structure
   - Ensure all keywords have:
     - source: "autosuggest"
     - confidence: 1
     - appropriate context

2. `serperAutosuggestEmptyInputTest`:
   - Test with empty input
   - Verify error handling

3. `serperAutosuggestValidationTest`:
   - Verify each keyword object has required fields
   - Check confidence values are exactly 1.0
   - Verify source is "autosuggest"
   - Check context mentions Google autocomplete

## Implementation Notes

- Store Serper API key in environment variables
- Use TypeScript for type safety
- Use Zod for runtime type validation
- Keep consistent structure with serper-search task for later integration
- Follwo the .mdc rules for .cursor/rules/typescript.mdc, .cursor/rules/avoid-nesting.mdc, cursor/rules/vercel-ai-sdk.mdc, .cursor/rules/trigger.mdc

## ✅ Success Criteria

1. Successfully calls Serper autocomplete API
2. Receives and validates autocomplete suggestions
3. Transforms suggestions into standard keyword format
4. Maintains proper typing and validation
5. Passes all test cases
6. Handles errors gracefully
7. Ready for integration with keyword enrichment task

## Dependencies

- Serper.dev API key (environment variable: `SERPER_API_KEY`)
- Zod for schema validation

## Next Steps

1. Implement task based on this MVP
2. Create and run test suite inside `_keyword-research-test.ts`
