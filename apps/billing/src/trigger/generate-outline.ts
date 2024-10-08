import { db } from "@/lib/db-marketing/client";
import {
  firecrawlResponses,
  keywords,
  serperOrganicResults,
  serperSearchResponses,
} from "@/lib/db-marketing/schemas";
import { openai } from "@ai-sdk/openai";
import { task } from "@trigger.dev/sdk/v3";
import { generateObject, generateText } from "ai";
import { and, desc, eq, or } from "drizzle-orm";
import { z } from "zod";

// Define the schema for the expected outline structure
const outlineSchema = z.object({
  headings: z.array(
    z.object({
      heading: z.string(),
      subheadings: z
        .array(
          z.object({
            title: z.string(),
          }),
        )
        .optional(),
      description: z.string(),
      whyImportant: z.string(),
    }),
  ),
});

// Refined system prompt incorporating best practices
const systemPrompt = `
You are an **SEO Expert** and **Technical Content Writer** specializing in creating comprehensive and SEO-optimized content for Developer Tools and API-related topics.

**Your Task:**
Generate a structured outline for a glossary page dedicated to a specific API development term. The outline should dynamically create sections based on provided keywords from trusted sources (\`related_searches\` and \`auto_suggest\`) and insights from the headers and summarized content of the top 3 ranking pages. Ensure all sections are SEO-optimized and tailored to the specific term's context and search intent.

**Inputs Provided:**
1. **Glossary Term:** The specific API development term for which the glossary page is being created (e.g., "API Lifecycle Versioning").
2. **Keywords:**
   - **Trusted Keywords:** Keywords from \`related_searches\` and \`auto_suggest\` that must be included as exact matches.
   - **Header Keywords:** Keywords extracted from the headers of the top 3 ranking pages for the term.
3. **Top-Ranking Pages Content:** Summarized markdown content from the top 3 ranking pages for the term.

**Requirements for the Outline:**
1. **Dynamic and Customized Sections:**
   - Create unique sections based on keyword analysis and top-ranking pages' content.
   - Include engaging elements such as "fun facts" or "did you know?" snippets within relevant sections to enhance user engagement.

2. **Excluded Sections:** Do not include Definition, Introduction, or Title sections.

3. **SEO & Structure:**
   - **Keyword Integration:** Prioritize keywords based on relevance and search volume. Incorporate primary keywords into main headers and secondary keywords into content naturally.
   - **Header Guidelines:** Headers under 70 characters, descriptive, and keyword-inclusive without compromising readability.
   - **Avoid Keyword Stuffing:** Integrate keywords seamlessly to maintain a natural narrative.

4. **Clarity and Readability:**
   - Headers should be concise, descriptive, and informative.
   - **Flat Structure:** Adopt a flat structure with only headings unless a broad topic necessitates subheadings.

5. **Guardrails:**
   - **Header Count:** 3 to 10 major headers, adjusting based on keyword relevance and content needs.
   - **Section Flexibility:** Merge related sections or add subheadings only when necessary to ensure comprehensive coverage without redundancy.

6. **Content Instructions:**
   - **Heading:** [Provide heading title]
     - **Description:** [Brief description of the content to be included in this section, explaining its importance and referencing sources.]
     - *(Only include subheadings if the topic is broad and requires further breakdown.)*

**Additional Guidelines:**
- **Uniqueness:** Ensure each section offers distinct, tailored information without overlapping content.
- **Engagement:** Integrate engaging elements such as "fun facts" or "did you know?" snippets within relevant sections to enhance shareability.
- **Relevance:** Maintain a strong focus on the glossary term to ensure coherence.
- **Adaptability:** Adjust sections and headers based on keyword insights and the specific term's requirements.

**SEO Enhancements:**
- **Meta Descriptions:** Suggest a concise meta description incorporating primary keywords.
- **Tags:** Recommend relevant tags or categories based on keyword analysis.

`;

/**
 * Generates a comprehensive and SEO-optimized outline for a glossary page.
 * The process consists of multiple steps to ensure a high-quality, engaging, and SEO-friendly outline.
 *
 * @remarks
 * This function performs the following steps:
 *
 * 1. Drafts the initial outline based on SERP results and topic knowledge:
 *    - Includes headings, content descriptions, and importance explanations
 *    - Describes content types (paragraph snippets, images, tables, listicles (list snippets), etc.) for each section
 *
 * 2. Refines headings editorially:
 *    - Enhances engagement and curiosity-inducing qualities
 *
 * 3. Optimizes for SEO coverage and structure:
 *    - Incorporates keywords into headers and content
 *    - Ensures optimal structure for search engines
 *
 * 4. Reviews the outline scientifically:
 *    - Verifies comprehensive topic coverage
 *    - Identifies potential separate page topics
 *    - Evaluates need for subheadings
 *
 * 5. Reviews the outline editorially:
 *    - Checks readability and engagement of headings
 *    - Ensures logical flow between sections
 *    - Avoids overly clickbait or unprofessional headings
 *    - Maintains authoritative and professional tone
 *
 * 6. Performs final SEO review:
 *    - Confirms coverage of all relevant keywords
 *    - Ensures natural integration of keywords in headers and content
 *
 * 7. Applies all necessary changes from previous steps
 *
 * 8. Outputs the final, refined outline
 *
 * @param {Object} input - The input object containing necessary data.
 * @param {string} input.term - The glossary term to create the outline for.
 *
 * @returns {Promise<string>} A promise that resolves to the final, optimized outline.
 *
 */
export const generateOutlineTask = task<"generate_outline", { term: string }>({
  id: "generate_outline",
  retry: {
    maxAttempts: 0,
  },
  run: async ({ term }) => {
    // Fetch top-ranking pages' markdown content
    const organicResults = await db.query.firecrawlResponses.findMany({
      where: eq(firecrawlResponses.inputTerm, term),
      with: {
        serperOrganicResult: {
          columns: { position: true },
        },
      },
    });
    console.info(`Step 1/10 - ORGANIC RESULTS: ${organicResults?.length} results`);

    const summarizerSystemPrompt = ({ term, position }: { term: string; position: number }) => `You are the **Chief Technology Officer (CTO)** of a leading API Development Tools Company with extensive experience in API development using programming languages such as Go, TypeScript, and Elixir and other backend languages. You have a PhD in computer science from MIT. Your expertise ensures that the content you summarize is technically accurate, relevant, and aligned with best practices in API development and computer science.

**Your Task:**
Accurately and concisely summarize the content from the page that ranks #${position} for the term "${term}". Focus on technical details, including how the content is presented (e.g., text, images, tables). Ensure factual correctness and relevance to API development.

**Instructions:**
- Provide a clear and concise summary of the content.
- Highlight key technical aspects and insights related to API development.
- Mention the types of content included, such as images, tables, code snippets, etc.
- Cite the term the content is ranking for and its position in the SERP.
`;
    // Summarize the markdown content to manage token limits
    const summaryPromises = organicResults?.map((result) => {
      const system = summarizerSystemPrompt({
        term,
        position: result.serperOrganicResult.position,
      });
      const prompt = `Summarize the following content for the term "${term}" that's ranking #${result.serperOrganicResult.position}:
      =======
      ${result.markdown}
      =======
      `;
      console.info(`Step 2/10 - SUMMARIZING: 
        SYSTEM: ${system}
        ---
        PROMPT: ${prompt}`);
      return generateText({
        model: openai("gpt-4o"),
        system,
        prompt,
        maxTokens: 500,
      });
    });

    const summariesCompletions = await Promise.all(summaryPromises);
    const contentKeywords = await db.query.keywords.findMany({
      where: and(
        or(eq(keywords.source, "headers"), eq(keywords.source, "title")),
        eq(keywords.inputTerm, term),
      ),
    });
    const topRankingContent = summariesCompletions.map((s) => s.text).join("=========\n\n");
    console.info(`Step 3/10 - SUMMARIES: ${topRankingContent}`);

    const initialOutlineSystem = `You are a **Technical SEO Content Writer** specializing in API development and computer science.
    Your objective is to create a flat, comprehensive outline for a glossary page based on summarized content from top-ranking pages.
    Ensure factual correctness, clarity, and SEO optimization without unnecessary subheadings.`;

    const initialOutlinePrompt = `
    Generate a comprehensive and factually accurate outline for a glossary page dedicated to the term: **${term}**.
    
    **Instructions:**
    - Analyze the summarized content from the top-ranking pages.
    - Create a flat, customized outline with sections that best address the search intent and provide comprehensive coverage of the term.
    - Ensure all sections are factually correct, unique, and tailored to the specific term's context in API development and computer science.
    - Describe recommended content types (e.g., text, images, tables) for each section.
    - Adopt a flat structure with only headings unless a broad topic necessitates subheadings.
    - Include a short description under each heading that outlines the content to be included, explains its importance, and references sources.
    - Ensure headers are under 70 characters, descriptive, and maintain clarity and readability.
    
    =====
    TOP RANKING PAGES CONTENT:
    =====
    ${topRankingContent}
    
    =====
    KEYWORDS USED IN HEADERS:
    =====
    FROM PAGE TITLES:
    ${contentKeywords
      .filter((k) => k.source === "title")
      .map((k) => `- ${k.keyword}`)
      .join("\n")}
    FROM HEADERS:
    ${contentKeywords
      .filter((k) => k.source === "headers")
      .map((k) => `- ${k.keyword}`)
      .join("\n")}
    `;

    console.info(`Step 4/10 - INITIAL OUTLINE: 
      SYSTEM: ${initialOutlineSystem}
      ---
      PROMPT: ${initialOutlinePrompt}`);

    // Generate the initial outline
    const initialOutline = await generateObject({
      model: openai("gpt-4o"),
      system: initialOutlineSystem,
      prompt: initialOutlinePrompt,
      schema: outlineSchema,
    });
    console.info(
      `Step 4/10 - INITIAL OUTLINE RESULT: ${initialOutline.object.headings.map((h) => h.heading).join(", ")}`,
    );

    // const refineHeadingsPrompt = `
    // Refine the following initial outline to make the headings more engaging and curiosity-inducing. 
    
    // Initial Outline:
    // ${JSON.stringify(initialOutline.object.headings.map((h) => ({initial: h.heading})))}
    // `;

    // console.info("Step 5/10 - REFINE HEADINGS: Prompt:", refineHeadingsPrompt);

    // const refinedOutline = await generateObject({
    //   model: openai("gpt-4o"),
    //   system:
    //     "You are a skilled Senior Technical Content Writer with a background in journalism and write about API development and computer science. You know API development well since you are a practitioner as well having built multiple APIs & backend systems.",
    //   prompt: refineHeadingsPrompt,
    //   schema: z.object({
    //     headings: z.array(
    //       z.object({
    //         heading: z.object({
    //           refined: z.string(),
    //           initial: z.string(),
    //           benefitsOfRefining: z.string(),
    //           subheadings: z
    //             .array(
    //               z.object({
    //                 refined: z.string(),
    //                 initial: z.string(),
    //                 benefitsOfRefining: z.string(),
    //               }),
    //             )
    //             .optional(),
    //         })
    //       }),
    //     ),
    //   }),
    // });

    // console.info(
    //   `Step 5/10 - REFINED OUTLINE RESULT: ${refinedOutline.object.headings.map((h) => h.heading.refined).join(", ")}`,
    // );
    console.info(`
      Step 5/10 - SKIPPED`)

    const seoKeywords = await db.query.keywords.findMany({
      where: and(
        or(eq(keywords.source, "related_searches"), eq(keywords.source, "auto_suggest")),
        eq(keywords.inputTerm, term),
      ),
    });

    const relatedSearches = seoKeywords.filter((k) => k.source === "related_searches");
    const autoSuggest = seoKeywords.filter((k) => k.source === "auto_suggest");

    // const seoRefinementSystem = `
    // You are a **Senior SEO Strategist & Technical Content Specialist** with over 10 years of experience in optimizing content for API development and computer science domains. Your expertise lies in structuring content with optimal heading hierarchies to maximize SEO performance while ensuring technical accuracy and user engagement.
    
    // **Your Task:**
    // Optimize the technical headers of the provided outline to ensure they are SEO-optimized around the given list of keywords. Adhere strictly to SEO best practices to enhance search engine visibility and user experience.
    
    // **Heading Best Practices to Follow:**
    // - **Single H1 Usage:** Only one H1 tag should be used in the entire outline. This should be the main title of the glossary page.
    // - **Keyword Inclusion:** Incorporate primary keywords into main headings (H2, H3, etc.) and secondary keywords within descriptions naturally.
    // - **Heading Hierarchy:** Maintain a logical and clear hierarchy using appropriate heading levels.
    // - **Avoid Keyword Stuffing:** Integrate keywords seamlessly without overusing them to maintain readability and avoid penalties.
    // - **Optimize for Search Intent:** Ensure that headings align with the user's search intent, providing clear answers and relevant information.
    // - **Optimize for Featured Snippets:** Structure headings and descriptions to increase the likelihood of appearing in featured snippets.
    // - **Engaging and Compelling:** Craft headings that are interesting and curiosity-inducing to encourage user interaction.
    // - **Power Words:** Utilize power words to make headings more compelling and clickable.
    // - **Consistency:** Use heading tags consistently throughout the outline to maintain a professional and organized structure.
    // - **Split Text with Header Tags:** Use header tags to break up large blocks of text, enhancing readability and SEO.
    // - **Avoid Overusing Header Tags:** Use header tags judiciously to maintain a clean and uncluttered outline.
    
    // **Optimization Requirements:**
    // 1. **Review Each Heading:**
    //    - Ensure relevant keywords are included in each heading based on the provided list.
    //    - Maintain only one H1 tag for the main title.
    
    // 2. **Enhance Headings:**
    //    - Make headings more engaging and curiosity-inducing.
    //    - Use power words to increase clickability.
    
    // 3. **Maintain Structure:**
    //    - Keep the outline flat, minimizing the use of subheadings unless absolutely necessary.
    
    // 4. **Technical Accuracy:**
    //    - Ensure that optimized headings accurately reflect the content and maintain technical correctness.
    
    // 5. **SEO Best Practices:**
    //    - Avoid keyword stuffing.
    //    - Align headings with search intent and optimize for featured snippets.
    
    // **Instructions:**
    // - Review each heading in the current outline.
    // - Optimize each heading based on the provided keywords and the best practices outlined above.
    // - Ensure that the overall structure remains flat, minimizing the use of subheadings unless absolutely necessary.
    // - Maintain a balance between SEO optimization and user engagement to create a compelling and authoritative glossary page.
    
    
    // `;
    // const seoRefinementPrompt = `
    // Optimize the following outline's headers using the provided keywords. Ensure that the headers are SEO-optimized according to best practices, focusing on search intent and maximizing visibility for API developers seeking technical understanding.
    
    // **Glossary Term:** **${term}**
    
    // **Provided Keywords:**
    // FROM RELATED SEARCHES:
    // ${relatedSearches.map((k) => `- ${k}`).join("\n")}
    // FROM AUTO SUGGEST:
    // ${autoSuggest.map((k) => `- ${k}`).join("\n")}
    
    // **Current Outline:**
    // ${JSON.stringify(initialOutline.object.headings.map((h) => ({current: h.heading})))}
    // `;
    // console.info(`Step 6/10 - SEO REFINEMENT: 
    //   SYSTEM: ${seoRefinementSystem}
    //   ---
    //   PROMPT: ${seoRefinementPrompt}`);

    // const seoOptimizedOutline = await generateObject({
    //   model: openai("gpt-4o"),
    //   system: seoRefinementSystem,
    //   prompt: seoRefinementPrompt,
    //   schema: z.object({
    //     headings: z.array(
    //       z.object({
    //         heading: z.object({
    //           current: z.string(),
    //           refined: z.string(),
    //           keywordsUsedInHeading: z.array(z.string()),
    //           keywordsToUseInContent: z.array(z.string()),
    //         }),
    //         subheadings: z
    //           .array(
    //             z.object({
    //               current: z.string(),
    //               refined: z.string(),
    //               keywordsUsedInHeading: z.array(z.string()),
    //               keywordsToUseInContent: z.array(z.string()),
    //             }),
    //           )
    //           .optional(),
    //       }),
    //     ),
    //   }),
    // });
    // console.info(`Step 6/10 - SEO OPTIMIZED OUTLINE RESULT: ${seoOptimizedOutline.object.headings.map((h) => h.heading.refined).join(", ")}`);
    console.info(`Step 6/10 - SKIPPED`)

    // System prompt for the SEO review task
    const seoReviewSystem = `
    You are a **Senior SEO Strategist & Technical Content Specialist** with over 10 years of experience in optimizing content for API development and computer science domains. Your expertise lies in structuring content with optimal heading hierarchies to maximize SEO performance while ensuring technical accuracy and user engagement.
    
    **Your Task:**
    Review the following outline and perform these checks:
    1. Ensure all provided keywords are allocated to headings or content.
    2. Verify there are no duplicate keyword allocations (each keyword should only be used in one heading).
    3. Check that no unrelated keywords are used.
    4. Confirm that the headings alone provide a comprehensive overview for an API developer seeking to understand the term.
    5. Assess if the headings sufficiently cover the provided keywords, suggesting new headings if necessary.

    **Important:**
    - Add new headings sparingly, only if it significantly benefits the SEO perspective.
    - You have the authority to add new headers if deemed highly beneficial for SEO, after careful consideration.

    Focus on maximizing visibility and addressing the search intent of API developers seeking technical understanding of the term.
    `;

    const seoReviewPrompt = `
    Review the following SEO-optimized outline for the term "${term}":

    ${JSON.stringify(initialOutline.object.headings.map((h) => ({initial: h.heading})))}

    Provided keywords:
    ${JSON.stringify([...relatedSearches, ...autoSuggest])}

    Please provide your analysis and any recommendations for improvement.
    `;

    console.info(`Step 7/10 - SEO REVIEW: 
      SYSTEM: ${seoReviewSystem}
      ---
      PROMPT: ${seoReviewPrompt}`);

    const seoReview = await generateObject({
      model: openai("gpt-4o"),
      system: seoReviewSystem,
      prompt: seoReviewPrompt,
      schema: z.object({
        analysis: z.object({
          keywordCoverage: z.string(),
          duplicateAllocations: z.string(),
          unrelatedKeywords: z.string(),
          comprehensiveness: z.string(),
          keywordSufficiency: z.string(),
        }),
        recommendations: z.array(z.object({
          type: z.enum(['addHeading', 'modifyHeading', 'reallocateKeyword']),
          description: z.string(),
          suggestedChange: z.string().optional(),
        })),
        previousHeadings: z.array(z.object({
          heading: z.string(),
          keywordsUsedInHeading: z.array(z.string()),
          keywordsToUseInContent: z.array(z.string()),
        })),
        newHeadings: z.array(z.object({
          heading: z.string(),
          justification: z.string(),
        })).optional(),
      }),
    });
    console.info(`Step 7/10 - SEO REVIEW RESULT: Analysis: ${JSON.stringify(seoReview.object.analysis)}, Recommendations: ${seoReview.object.recommendations.length}`);
    // console.info(`Step 7/10 - SKIPPED`)

    const factualReviewSystem = `
    You are a **Senior Technical Content Writer** with a PhD in computer science from MIT. You are an expert in API development and computer science.

    **Your Task:**
    Review the following outline for the term "${term}" and perform these checks:
    1. Check if the outline covers all important aspects of the topic
    2. Check if the topic is covered comprehensively (no important details are missed)
    3. Check if the topic is covered concisely (no fluff, no unnecessary details)
    4. Check if the outline doesn't include unrelated topics or terms better suited for a separate entry
    5. Check if the outline is flat and doesn't include subheadings unless absolutely necessary
    
    **Considerations:**
    - ensure using various patterns to improve for SEO
    - '<keyword>: <some title>', using questions, using power words, using numbers are all valid patterns, but don't overuse any one too much.
    `;

    const factualReviewPrompt = `
    Review the following outline for the term "${term}":

    ${JSON.stringify(initialOutline.object.headings.map((h) => ({initial: h.heading})))}

    Please provide your analysis and any recommendations for improvement.

    ====
    TOP RANKING PAGES CONTENT:
    ====
    ${topRankingContent}

    ====
    KEYWORDS USED IN HEADERS:
    ====
    ${contentKeywords.map((k) => `- ${k.keyword}`).join("\n")}
    `;

    console.info(`Step 8/10 - FACTUAL REVIEW: 
      SYSTEM: ${factualReviewSystem}
      ---
      PROMPT: ${factualReviewPrompt}`);

    const factualReview = await generateObject({
      model: openai("gpt-4o"),
      system: factualReviewSystem,
      prompt: factualReviewPrompt,
      schema: z.object({
        analysis: z.object({
          topicCoverage: z.string(),
          comprehensiveness: z.string(),
          conciseness: z.string(),
          relatedness: z.string(),
        }),
        recommendations: z.array(z.object({
          type: z.enum(['addHeading', 'modifyHeading', 'removeHeading']),
          description: z.string(),
          suggestedChange: z.string(),
        })).optional(),
      }),
    });
    console.info(`Step 8/10 - FACTUAL REVIEW RESULT: Analysis: ${JSON.stringify(factualReview.object.analysis)}, Recommendations: ${factualReview.object.recommendations.length}`);

    const editorialReviewSystem = `
    You are a **Senior Technical Content Writer** with a PhD in computer science from MIT. You are an expert in API development and computer science.

    **Your Task:**
    Review the following outline for the term "${term}" and perform these checks:
    1. Ensure that the reader gets a good understanding of the topic when seeing a tabel of contents (readers are api developers looking up api development related terms)
    2. Ensure that there's good flow between the headings. A question in one heading should be answered in a later heading.
    3. Ensure that the wording is authoritatively and professionally
    4. Ensure that headings are curiosity-inducing though and engaging
    5. Ensure that headings are not click-baity or cringe-worthy
    6. Ensure that there's content idnicated that share-worthy (e.g. little-known facts, fun facts, etc.)
    `;

    const editorialReviewPrompt = `
    Review the following outline for the term "${term}":

    ${JSON.stringify(initialOutline.object.headings.map((h) => ({initial: h.heading})))}

    Please provide your analysis and any recommendations for improvement.
    `;

    console.info(`Step 9/10 - EDITORIAL REVIEW: 
      SYSTEM: ${editorialReviewSystem}
      ---
      PROMPT: ${editorialReviewPrompt}`);

    const editorialReview = await generateObject({
      model: openai("gpt-4o"),
      system: editorialReviewSystem,
      prompt: editorialReviewPrompt,
      schema: z.object({
        analysis: z.object({
          readability: z.string(),
          flow: z.string(),
          clickbait: z.string(),
          authoritativeness: z.string(),
        }),
        recommendations: z.array(z.object({
          type: z.enum(['modifyHeading', 'addContent']),
          description: z.string(),
          suggestedChange: z.string().optional(),
        })),
      }),
    });
    console.info(`Step 9/10 - EDITORIAL REVIEW RESULT: Analysis: ${JSON.stringify(editorialReview.object.analysis)}, Recommendations: ${editorialReview.object.recommendations.length}`);

    const finalOutlineSystem = `
    You are the **Content Strategy Lead** for a DevTool startup specializing in Key Management for APIs. Your responsibility is to create a comprehensive and authoritative glossary of API-related terms on the company's website. You have a strong background in API development, security, and technical writing.

    **Your Task:**
    Review and finalize the outline for the glossary entry on "${term}". You have received feedback from three key stakeholders:
    1. The CTO (domain expert)
    2. A senior content writer (editorial expert)
    3. An SEO specialist

    Your goal is to create a final outline that balances technical accuracy, SEO optimization, and readability, with a priority on factual correctness and search engine visibility.

    **Priority:**
    1. **Factual Correctness:** Ensure all content aligns with the CTO's feedback.
    2. **SEO Optimization:** Incorporate relevant keywords and address search intent based on the SEO Specialist's review.
    3. **Readability and Engagement:** Enhance structure and flow as per the Content Writer's feedback without compromising technical accuracy or SEO goals.

    **Key Considerations:**
    - Align content with the company's expertise in API Key Management.
    - Maintain a clear and engaging structure.
    `;

    const finalOutlinePrompt = `
    Please review and refine the following outline for the term "${term}":

    Current Outline:
    ${JSON.stringify(initialOutline)}

    Stakeholder Feedback:
    1. CTO's Factual Review: ${JSON.stringify(factualReview)}
    2. Editorial Review: ${JSON.stringify(editorialReview)}
    3. SEO Expert Review: ${JSON.stringify(seoReview)}

    Based on this input, please provide:
    1. A final, refined outline
    2. A brief explanation of key changes and why they were made
    3. Any additional recommendations for content creation

    Strike a balance between SEO optimization and readability. 
    SEO optimization is very important but you want to have a good balance and ensure that it uses different patterns for headings without oversuing one particular pattern. 
    
    It should read like a natural language text from a high quality technical writer.

    If you can define a heading that satisfies all stakeholders, do so.
    `;

    console.info(`Step 10/10 - FINAL OUTLINE: 
      SYSTEM: ${finalOutlineSystem}
      ---
      PROMPT: ${finalOutlinePrompt}`);

    const finalOutline = await generateObject({
      model: openai("gpt-4o"),
      system: finalOutlineSystem,
      prompt: finalOutlinePrompt,
      schema: z.object({
        headings: z.array(
          z.object({
            heading: z.string(),
          subheadings: z
            .array(
              z.object({
                title: z.string(),
              }),
            )
            .optional(),
          description: z.string(),
          changesMade: z.string().optional(),
          additionalRecommendations: z.string().optional(),
        }),
      ),
    })});
    console.info(`Step 10/10 - FINAL OUTLINE RESULT: ${finalOutline.object.headings.map((h) => h.heading).join(", ")}`);
    return finalOutline;
  },
});
