import { task } from "@trigger.dev/sdk/v3";
import Exa, { type SearchResponse, type ContentsOptions, type RegularSearchOptions } from "exa-js";
import { z } from "zod";
import { createGoogleGenerativeAI } from "@ai-sdk/google";
import { generateObject } from "ai";
import fs from "node:fs";
import path from "node:path";

// Define cache directory and utilities for persisting data
const CACHE_DIR = path.join(process.cwd(), ".cache");
const SEARCH_CACHE_FILE = path.join(CACHE_DIR, "exa-search-results.json");
const EVALUATION_CACHE_FILE = path.join(CACHE_DIR, "evaluation-results.json");

// Ensure cache directory exists
const ensureCacheDir = () => {
  if (!fs.existsSync(CACHE_DIR)) {
    fs.mkdirSync(CACHE_DIR, { recursive: true });
  }
};

// Cache utilities
const cacheSearchResults = (query: string, results: SearchResponse<ContentsOptions & RegularSearchOptions>) => {
  ensureCacheDir();
  
  // Create a cache key using the query and current timestamp
  const cacheKey = `${query}_${new Date().toISOString()}`;
  let cache: Record<string, SearchResponse<ContentsOptions & RegularSearchOptions>> = {};
  
  if (fs.existsSync(SEARCH_CACHE_FILE)) {
    try {
      cache = JSON.parse(fs.readFileSync(SEARCH_CACHE_FILE, "utf8"));
    } catch (e) {
      console.warn("Error reading search cache file:", e);
    }
  }
  
  cache[cacheKey] = results;
  
  // Stringify with pretty formatting for readability
  const cacheData = JSON.stringify(cache, null, 2);
  fs.writeFileSync(SEARCH_CACHE_FILE, cacheData);
  
  // Get file stats for logging
  const stats = fs.statSync(SEARCH_CACHE_FILE);
  const fileSizeKB = (stats.size / 1024).toFixed(2);
  
  console.log(`📦 CACHE: Stored search results for "${query}"`);
  console.log(`📄 File: ${path.resolve(SEARCH_CACHE_FILE)} (${fileSizeKB} KB)`);
  console.log(`🔑 Cache key: ${cacheKey}`);
  console.log(`📊 Total cache entries: ${Object.keys(cache).length}`);
  
  return cacheKey;
};

const getSearchResultsFromCache = (query: string): SearchResponse<ContentsOptions & RegularSearchOptions> | null => {
  if (!fs.existsSync(SEARCH_CACHE_FILE)) {
    console.log("📦 CACHE: No search cache file exists yet");
    return null;
  }
  
  try {
    const cache = JSON.parse(fs.readFileSync(SEARCH_CACHE_FILE, "utf8")) as Record<string, SearchResponse<ContentsOptions & RegularSearchOptions>>;
    // Find the latest cache entry for this query
    const keys = Object.keys(cache)
      .filter(k => k.startsWith(`${query}_`))
      .sort()
      .reverse();
    
    if (keys.length > 0) {
      console.log(`📦 CACHE HIT: Using cached search results for "${query}"`);
      return cache[keys[0]];
    }
    
    console.log(`📦 CACHE MISS: No cached search results found for "${query}"`);
  } catch (e) {
    console.warn("Error reading from search cache:", e);
  }
  
  return null;
};

const cacheEvaluationResults = (query: string, results: EvaluatedSearchResponse) => {
  ensureCacheDir();
  
  // Create a cache key using the query and current timestamp
  const cacheKey = `${query}_${new Date().toISOString()}`;
  let cache: Record<string, EvaluatedSearchResponse> = {};
  
  if (fs.existsSync(EVALUATION_CACHE_FILE)) {
    try {
      cache = JSON.parse(fs.readFileSync(EVALUATION_CACHE_FILE, "utf8"));
    } catch (e) {
      console.warn("Error reading evaluation cache file:", e);
    }
  }
  
  cache[cacheKey] = results;
  
  // Stringify with pretty formatting for readability
  const cacheData = JSON.stringify(cache, null, 2);
  fs.writeFileSync(EVALUATION_CACHE_FILE, cacheData);
  
  // Get file stats for logging
  const stats = fs.statSync(EVALUATION_CACHE_FILE);
  const fileSizeKB = (stats.size / 1024).toFixed(2);
  
  console.log(`📦 CACHE: Stored evaluation results for "${query}"`);
  console.log(`📄 File: ${path.resolve(EVALUATION_CACHE_FILE)} (${fileSizeKB} KB)`);
  console.log(`🔑 Cache key: ${cacheKey}`);
  console.log(`📊 Total cache entries: ${Object.keys(cache).length}`);
  
  return cacheKey;
};

const getEvaluationFromCache = (query: string): EvaluatedSearchResponse | null => {
  if (!fs.existsSync(EVALUATION_CACHE_FILE)) {
    console.log("📦 CACHE: No evaluation cache file exists yet");
    return null;
  }
  
  try {
    const cache = JSON.parse(fs.readFileSync(EVALUATION_CACHE_FILE, "utf8")) as Record<string, EvaluatedSearchResponse>;
    // Find the latest cache entry for this query
    const keys = Object.keys(cache)
      .filter(k => k.startsWith(`${query}_`))
      .sort()
      .reverse();
    
    if (keys.length > 0) {
      console.log(`📦 CACHE HIT: Using cached evaluation results for "${query}"`);
      return cache[keys[0]];
    }
    
    console.log(`📦 CACHE MISS: No cached evaluation results found for "${query}"`);
  } catch (e) {
    console.warn("Error reading from evaluation cache:", e);
  }
  
  return null;
};

// Define domain categories
type DomainCategory = {
  name: string;
  domains: string[];
  description: string;
};

const domainCategories: DomainCategory[] = [
  {
    name: "Official",
    domains: ["tools.ietf.org", "datatracker.ietf.org", "rfc-editor.org", "w3.org", "iso.org"],
    description: "Official standards and specifications sources",
  },
  {
    name: "Community",
    domains: ["stackoverflow.com", "github.com", "wikipedia.org", "news.ycombinator.com"],
    description: "Community-driven platforms and forums",
  },
  {
    name: "Neutral",
    domains: ["owasp.org", "developer.mozilla.org"],
    description: "Educational and vendor-neutral resources",
  },
  {
    name: "Google",
    domains: [], // Empty domains array to search without domain restrictions
    description: "General search results without domain restrictions",
  },
];

// Define type for function parameters
type ExaSearchParams = {
  searchQuery: string;
  numResults?: number;
  apiKey?: string;
  useCache?: boolean;
  forceRefresh?: boolean;
};

type ExaResultWithCategory = SearchResponse<ContentsOptions & RegularSearchOptions> & {
  categoryName: string;
  categoryDescription: string;
  category?: string; // Added for flexibility in the evaluation process
};

// Define the main search task
export const exaDomainSearchTask = task({
  id: "exa_domain_search",
  run: async ({
    searchQuery,
    numResults = 10,
    apiKey,
    useCache = false,
    forceRefresh = false,
  }: ExaSearchParams) => {
    // Check cache first if allowed
    if (useCache && !forceRefresh) {
      const cachedResults = getSearchResultsFromCache(searchQuery);
      if (cachedResults) {
        return cachedResults;
      }
    } else if (forceRefresh) {
      console.log(`🔄 REFRESH: Forced refresh of search results for "${searchQuery}"`);
    } else if (!useCache) {
      console.log(`🌐 LIVE: Cache disabled, using live Exa API for "${searchQuery}"`);
    }

    // Initialize Exa client with API key from input or env var
    const exa = new Exa(apiKey || process.env.EXA_API_KEY || "");

    // Validate API key is available
    if (!apiKey && !process.env.EXA_API_KEY) {
      throw new Error("Exa API key not provided and EXA_API_KEY environment variable not set");
    }

    console.log("🌐 EXA API: Starting parallel searches", {
      query: searchQuery,
      categories: domainCategories.map((c) => c.name),
    });

    // Track total costs across all searches
    let totalCostDollars = 0;
    const costBreakdown: Record<string, any> = {};

    // Execute all searches in parallel
    const searchPromises = domainCategories.map(async (category) => {
      console.log(`Searching category: ${category.name}`, { domains: category.domains });

      // Base search options for all categories
      const searchOptions = {
        summary: true,
        numResults,
        type: "keyword",
        text: true,
        ...(category.domains.length > 0 ? { includeDomains: category.domains } : {}),
      } satisfies RegularSearchOptions & ContentsOptions;

      // Type assertion to avoid TypeScript errors with dynamic properties
      const searchResult = await exa.searchAndContents(searchQuery, searchOptions);

      // Log costs if they're available in the response
      if (searchResult.data?.costDollars) {
        const costs = searchResult.data.costDollars;
        console.log(`Exa API costs for ${category.name} search:`, {
          total: `$${costs.total.toFixed(4)}`,
          search: costs.search,
          contents: costs.contents,
        });

        // Add costs to totals
        totalCostDollars += costs.total;
        costBreakdown[category.name] = costs;
      }

      console.log(`Found ${searchResult.results?.length || 0} results for ${category.name}`);

      // Enrich each result with the category information
      const enrichedResults = (searchResult.results || []).map(
        (result): ExaResultWithCategory => ({
          ...result,
          categoryName: category.name,
          categoryDescription: category.description,
          category: category.name,
        }),
      );

      return {
        category: category.name,
        description: category.description,
        domains: category.domains,
        results: enrichedResults,
      };
    });

    // Wait for all searches to complete
    const results = await Promise.all(searchPromises);

    // Log the total costs
    if (totalCostDollars > 0) {
      console.log(`Total Exa API costs for all searches: $${totalCostDollars.toFixed(4)}`, {
        breakdown: costBreakdown,
      });
    }

    // Return aggregated results
    const finalResults = {
      query: searchQuery,
      timestamp: new Date().toISOString(),
      categoryResults: results,
      totalResults: results.reduce((sum, category) => sum + category.results.length, 0),
      costs: {
        total: totalCostDollars,
        breakdown: costBreakdown,
      },
    };

    if (useCache) {
      cacheSearchResults(searchQuery, finalResults);
    }

    return finalResults;
  },
});

// Evaluation schema for content quality and relevance
const evaluationSchema = z.object({
  /**
   * Rating from 1-10 on relevance and quality
   */
  rating: z.number().min(1).max(10),

  /**
   * Detailed justification for the rating
   */
  justification: z.string(),
});

// Type for the final response with evaluation
type EvaluatedSearchResponse = SearchResponse<ContentsOptions & RegularSearchOptions> & {
  evaluationSummary?: {
    totalEvaluated: number;
    included: number;
    excluded: number;
  };
  evaluationError?: string;
};

// Define a scheduled job that runs single-sign-on search specifically
// and performs evaluation of the results
export const singleSignOnSearchTask = task({
  id: "scheduled_sso_domain_research",
  run: async ({
    searchQuery = "single-sign-on",
    cacheSearch = true, // Use cached search by default
    cacheEvaluation = false, // Don't cache evaluation by default
    forceRefreshSearch = false, // Don't force refresh search by default
  }: {
    searchQuery?: string;
    cacheSearch?: boolean;
    cacheEvaluation?: boolean;
    forceRefreshSearch?: boolean;
  } = {}): Promise<EvaluatedSearchResponse> => {
    console.log("Starting domain research:", {
      query: searchQuery,
      cacheSearch,
      cacheEvaluation,
    });

    // PART 1: GET SEARCH RESULTS (from Exa or cache)
    let searchResults: any; // Using any temporarily to handle type mismatch between cache and exa-js

    // Check cache first if enabled
    if (cacheSearch && !forceRefreshSearch) {
      const cachedResults = getSearchResultsFromCache(searchQuery);
      if (cachedResults) {
        console.log("Using cached search results");
        searchResults = cachedResults;
      } else {
        console.log("No cached results found, fetching from Exa");
        const searchResultTask = await exaDomainSearchTask.triggerAndWait({
          searchQuery,
          numResults: 10,
          useCache: cacheSearch,
        });

        if (!searchResultTask.ok) {
          throw new Error(`Search task failed: ${searchResultTask.error}`);
        }

        searchResults = searchResultTask.output;
        
        // Cache the results if needed
        if (cacheSearch) {
          cacheSearchResults(searchQuery, searchResults as SearchResponse<ContentsOptions & RegularSearchOptions>);
        }
      }
    } else {
      console.log("Fetching fresh search results from Exa");
      const searchResultTask = await exaDomainSearchTask.triggerAndWait({
        searchQuery,
        numResults: 10,
        useCache: false,
      });

      if (!searchResultTask.ok) {
        throw new Error(`Search task failed: ${searchResultTask.error}`);
      }

      searchResults = searchResultTask.output;
      
      // Cache the results if needed
      if (cacheSearch) {
        cacheSearchResults(searchQuery, searchResults as SearchResponse<ContentsOptions & RegularSearchOptions>);
      }
    }

    // Log the total results - handle both old and new structure
    const totalResults = 'totalResults' in searchResults 
      ? searchResults.totalResults 
      : (searchResults.results?.length || 0);
    console.log(`Got ${totalResults} search results to evaluate`);

    // PART 2: EVALUATE SEARCH RESULTS with Gemini
    try {
      // Create a flat list of all results with category info
      // Handle both old and new structure
      const allResults = 'categoryResults' in searchResults
        ? searchResults.categoryResults.flatMap((category: any) =>
            category.results.map((result: any) => ({
              ...result,
              category: category.category,
            }))
          )
        : searchResults.results.map((result: any) => ({
            ...result,
            category: 'general', // Default category for new structure
          }));

      // Prepare data for evaluation
      const resultsForEvaluation = allResults.map((result, index) => {
        const content = result.summary || result.text || "";
        return {
          id: index,
          title: result.title || "No title",
          url: result.url,
          category: result.category,
          publishedDate: result.publishedDate,
          content: content.slice(0, 1500),
        };
      });

      // Set up the evaluation schema
      const batchEvaluationSchema = z.object({
        resultId: z.number(),
        evaluation: evaluationSchema,
      });

      console.log("Calling Gemini to evaluate all results");

      // Call Gemini for evaluation
      const google = createGoogleGenerativeAI({
        apiKey: process.env.GEMINI_API_KEY,
      });

      const geminiResponse = await generateObject({
        model: google("gemini-1.5-flash") as any,
        schema: batchEvaluationSchema,
        output: "array",
        prompt: `
          Evaluate these search results for relevance to: "${searchQuery}"
          
          For each result below, return an evaluation with:
          - resultId: The ID number shown in brackets
          - evaluation:
            - rating: 1-10 scale (10 = highly relevant, 1 = irrelevant)
            - justification: Brief explanation why, including noting if content is outdated
          
          GUIDANCE ON EVALUATING CONTENT:
          - Generally prioritize content from recent years (2020-present) as they contain the most relevant and up-to-date information
          - Be cautious with older content (pre-2020), especially for technical implementations that change quickly
          - Only give high ratings (7+) to older content if it's truly foundational, authoritative, or contains timeless information
          - Consider the source quality - an older article from a highly respected source might deserve inclusion
          - The ideal content is both highly relevant to "${searchQuery}" AND reasonably current
          
          In short: Rate based primarily on relevance and quality, with recency as an important but not absolute factor.
          Don't automatically exclude older content if it's exceptional, but generally prefer more recent sources.
          
          Here are the results:
          
          ${resultsForEvaluation
            .map(
              (r) => `[Result ID: ${r.id}]
          Title: ${r.title}
          URL: ${r.url}
          Category: ${r.category}
          Published: ${r.publishedDate || "Unknown date"}
          Content: ${r.content}
          `,
            )
            .join("\n\n")}
          
          IMPORTANT: You must return evaluations for ALL ${resultsForEvaluation.length} results.
          CRITICAL: Return a flat array of objects, not an array of arrays.
        `,
      });

      // Get the evaluations from the response
      const evaluations = geminiResponse.object;
      if (!Array.isArray(evaluations)) {
        throw new Error("Gemini did not return an array");
      }
      if (evaluations.length === 0) {
        // inspect the evaluations object:
        console.log("Evaluations object:", evaluations);
        throw new Error("Gemini did not return any evaluations");
      }

      console.log(`Received ${evaluations.length} evaluations from Gemini`);

      // Map evaluations back to results - keep it simple
      const evaluatedResults = allResults.map((result, index) => {
        // Find evaluation with matching resultId
        const evaluation = evaluations.find((e) => e?.resultId === index)?.evaluation;

        return {
          ...result, // Keep all original properties
          evaluation, // No fallback - if it's undefined, it's undefined
        };
      });

      // log the evaluated results:
      console.log(
        "Evaluated results (first 10):",
        evaluatedResults.slice(0, 10).map((r) => ({
          title: r.title,
          url: r.url,
          category: r.category,
          publishedDate: r.publishedDate,
          summary: r.summary,
          evaluation: r.evaluation,
        })),
      );
      console.log(
        "Evaluated results (next 10):",
        evaluatedResults.slice(10, 20).map((r) => ({
          title: r.title,
          url: r.url,
          category: r.category,
          publishedDate: r.publishedDate,
          summary: r.summary,
          evaluation: r.evaluation,
        })),
      );
      console.log(
        "Evaluated results (next 10):",
        evaluatedResults.slice(20, 30).map((r) => ({
          title: r.title,
          url: r.url,
          category: r.category,
          publishedDate: r.publishedDate,
          summary: r.summary,
          evaluation: r.evaluation,
        })),
      );
      console.log(
        "Evaluated results (next 10):",
        evaluatedResults.slice(30, 40).map((r) => ({
          title: r.title,
          url: r.url,
          category: r.category,
          publishedDate: r.publishedDate,
          summary: r.summary,
          evaluation: r.evaluation,
        })),
      );

      // Filter results by evaluation rating only
      const filteredResults = evaluatedResults.filter((result) => {
        // Skip results without evaluation
        if (!result.evaluation) {
          return false;
        }

        // Only filter based on quality rating (7+)
        return result.evaluation.rating >= 7;
      });

      console.log(`Filtered to ${filteredResults.length} relevant results (rating ≥7)`);
      console.log(`Count by category:
        ${JSON.stringify(filteredResults.reduce((acc, result) => {
          const category = result.category;
          if (!acc[category]) {
            acc[category] = {
              count: 0,
              urls: []
            };
          }
          acc[category].count++;
          acc[category].urls.push(result.url);
          return acc;
        }, {} as Record<string, {count: number, urls: string[]}>),
          null,
          2,
        )}
      `);
      const categoryMap = new Map<string, CategoryResult>();

      for (const result of filteredResults) {
        const category = result.category;

        if (!categoryMap.has(category)) {
          const originalCategory = searchResults.categoryResults.find(
            (cat) => cat.category === category,
          );

          if (originalCategory) {
            categoryMap.set(category, {
              ...originalCategory,
              results: [],
            });
          }
        }

        const categoryData = categoryMap.get(category);
        if (categoryData) {
          const { category: _, ...resultWithoutCategory } = result;
          categoryData.results.push(resultWithoutCategory as ExaResultWithCategory);
        }
      }

      // Prepare the final response
      const finalResponse: EvaluatedSearchResponse = {
        query: searchQuery,
        timestamp: new Date().toISOString(),
        categoryResults: Array.from(categoryMap.values()),
        totalResults: filteredResults.length,
        evaluationSummary: {
          totalEvaluated: evaluatedResults.length,
          included: filteredResults.length,
          excluded: evaluatedResults.length - filteredResults.length,
        },
      };

      // Cache the evaluation results if requested
      if (cacheEvaluation) {
        console.log(`💾 Storing final evaluation results for "${searchQuery}" with ${filteredResults.length} filtered results`);
        console.log(`📊 Evaluation summary: ${filteredResults.length}/${evaluatedResults.length} results included (${((filteredResults.length / evaluatedResults.length) * 100).toFixed(1)}%)`);
        cacheEvaluationResults(searchQuery, finalResponse);
      } else {
        console.log(`ℹ️ Skipping cache for evaluation results (cacheEvaluation=${cacheEvaluation})`);
      }

      return finalResponse;
    } catch (error) {
      console.error("Error during evaluation:", error);

      // Return original results without evaluation if there's an error
      return {
        ...searchResults,
        evaluationError: `Evaluation failed: ${
          error instanceof Error ? error.message : String(error)
        }`,
      };
    }
  },
});
