import { UTApi, UTFile } from "uploadthing/server";

import { db } from "@/lib/db-marketing/client";
import { entries, sections } from "@/lib/db-marketing/schemas";
import { AbortTaskRunError, task } from "@trigger.dev/sdk/v3";
import { and, desc, eq, isNotNull } from "drizzle-orm";

export const compileMarkdownTask = task({
  id: "compile_markdown",
  retry: {
    maxAttempts: 0,
  },
  run: async ({ term }: { term: string }) => {
    // Fetch the entry
    const entry = await db.query.entries.findFirst({
      where: eq(entries.inputTerm, term),
    });

    if (!entry) {
      throw new Error(`Entry not found for term: ${term}`);
    }

    // Fetch the latest sections for each order value
    const latestSections = await db
      .select()
      .from(sections)
      .where(and(eq(sections.entryId, entry.id), isNotNull(sections.markdown)))
      .orderBy(desc(sections.order))
      .limit(10);

    console.info(`Found ${latestSections.length} sections for ${term}`);

    // Compile the markdown content
    let markdownContent = `# ${term}\n\n`;

    for (const section of latestSections) {
      markdownContent += `${section.markdown || ""}\n\n`;
    }


    // Convert the string content to a Blob
    const blob = new Blob([markdownContent], { type: 'text/markdown' });
    
    // Create a File object from the Blob
    const file = new File([blob], `${entry.inputTerm}.mdx`, { type: 'text/markdown' });
    

    const utapi = new UTApi({ token: process.env.UPLOADTHING_TOKEN });
    const [response] = await utapi.uploadFiles([file]);

    if (response.error) {
      throw new AbortTaskRunError(response.error.message);
    }
    await db
      .update(entries)
      .set({ utKey: response.data.key, utUrl: response.data.url })
      .where(eq(entries.id, entry.id));

    return response.data.url;
  },
});
