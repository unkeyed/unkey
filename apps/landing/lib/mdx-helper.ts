import { promises as fs } from "node:fs";
import path from "node:path";
import GithubSlugger from "github-slugger";
import type { MDXRemoteSerializeResult } from "next-mdx-remote";
import { serialize } from "next-mdx-remote/serialize";
import rehypeAutolinkHeadings from "rehype-autolink-headings";
import rehypeCodeTitles from "rehype-code-titles";
import rehypeSlug from "rehype-slug";
import remarkGfm from "remark-gfm";

export const BLOG_PATH = path.join(process.cwd(), "content", "blog");
export const CHANGELOG_PATH = path.join(process.cwd(), "content", "changelog");
export const POLICY_PATH = path.join(process.cwd(), "content", "policies");
export const JOBS_PATH = path.join(process.cwd(), "content", "jobs");

export const getFilePaths = async (contentPath: string) => {
  return await fs.readdir(contentPath);
};
export const raw = async ({
  contentPath,
  filepath,
}: {
  contentPath: string;
  filepath: string;
}) => {
  return await fs.readFile(`${contentPath}/${filepath}.mdx`, "utf-8");
};

type Headings = {
  slug: string | undefined;
  level: string;
  text: string | undefined;
};

export type Post<TFrontmatter> = {
  serialized: MDXRemoteSerializeResult;
  frontmatter: TFrontmatter;
  headings: Headings[];
};

type Changelog<TFrontmatter> = {
  serialized: MDXRemoteSerializeResult;
  frontmatter: TFrontmatter;
};

type ChangeLogFrontmatter = {
  title: string;
  date: string;
  description: string;
};

type Policy<TFrontmatter> = {
  serialized: MDXRemoteSerializeResult;
  frontmatter: TFrontmatter;
};

type PolicyFrontmatter = {
  title: string;
};

type Job<TFrontmatter> = {
  serialized: MDXRemoteSerializeResult;
  frontmatter: TFrontmatter;
};
type JobFrontmatter = {
  title: string;
  description: string;
  visible: boolean;
  salary: string;
};

export type Frontmatter = {
  title: string;
  date: string;
  description: string;
  author: string;
  visible: boolean | undefined;
  salary: string | undefined;
  level: string | undefined;
  image: string | undefined;
  tags: string | undefined;
};

export type Tags = "product" | "engineering" | "company" | "industry" | "security";

// const options = {
//   theme: themes.jettwaveDark,
//   getHighlighter: (options: HighlighterOptions) =>
//     getHighlighter({
//       ...options,
//       langs: [...BUNDLED_LANGUAGES],
//     }),
//   defaultLang: {
//     block: "typescript",
//   },
// };
// Serialize the MDX content and parse the frontmatter
export const mdxSerialized = async ({ rawMdx }: { rawMdx: string }) => {
  return await serialize(rawMdx, {
    parseFrontmatter: true,
    mdxOptions: {
      remarkPlugins: [remarkGfm],
      rehypePlugins: [
        // [rehypePrettyCode, options],
        rehypeAutolinkHeadings,
        rehypeSlug,
        rehypeCodeTitles,
      ],
    },
  });
};

const getHeadings = async ({ rawMdx }: { rawMdx: string }) => {
  const slugger = new GithubSlugger();
  const regXHeader = /\n(?<flag>#{1,6})\s+(?<content>.+)/g;
  const headings = Array.from(rawMdx.matchAll(regXHeader)).map(({ groups }) => {
    const flag = groups?.flag;
    const content = groups?.content;
    return {
      level: flag?.length === 1 ? "one" : flag?.length === 2 ? "two" : "three",
      text: content,
      slug: content ? slugger.slug(content) : undefined,
    };
  });
  return headings;
};

const getMoreContent = async ({
  contentPath,
  filepath,
}: {
  contentPath: string;
  filepath: string;
}) => {
  const moreContent = await fs.readdir(contentPath);
  const moreContentFiltered = moreContent
    .filter((path) => /\.mdx?$/.test(path))
    .filter((post) => post !== filepath)
    .slice(0, 2);
  return moreContentFiltered;
};

export const getAllMDXData = async ({
  contentPath,
}: {
  contentPath: string;
}) => {
  const allPosts = await fs.readdir(contentPath);
  const allPostsFiltered = allPosts.filter((path) => /\.mdx?$/.test(path));
  const allPostsData = await Promise.all(
    allPostsFiltered.map(async (post) => {
      const rawMdx = await raw({
        contentPath,
        filepath: post.replace(/\.mdx?$/, ""),
      });
      const serializedMdx = await mdxSerialized({ rawMdx });
      const frontmatter = serializedMdx.frontmatter as Frontmatter;
      return {
        frontmatter,
        slug: post.replace(/\.mdx$/, ""),
      };
    }),
  );
  return allPostsData;
};

export const getContentData = async ({
  contentPath,
  filepath,
}: {
  contentPath: string;
  filepath: string;
}) => {
  const moreContent = await getMoreContent({ contentPath, filepath });
  const moreContentData = await Promise.all(
    moreContent.map(async (content) => {
      const rawMdx = await raw({
        contentPath,
        filepath: content.replace(/\.mdx?$/, ""),
      });
      const serializedMdx = await mdxSerialized({ rawMdx });
      const frontmatter = serializedMdx.frontmatter as Frontmatter;
      return {
        frontmatter,
        slug: content.replace(/\.mdx$/, ""),
      };
    }),
  );
  return moreContentData;
};

export const getMeta = async (filepath: string) => {
  const rawMdx = await raw({ contentPath: BLOG_PATH, filepath: filepath });
  const serialized = await mdxSerialized({ rawMdx });
  const frontmatter = serialized.frontmatter as Frontmatter;
  return {
    frontmatter,
  };
};

export const getPost = async (filepath: string): Promise<Post<Frontmatter>> => {
  const rawMdx = await raw({ contentPath: BLOG_PATH, filepath: filepath });
  // Serialize the MDX content and parse the frontmatter
  const serialized = await mdxSerialized({ rawMdx });
  const frontmatter = serialized.frontmatter as Frontmatter;
  const headings = await getHeadings({ rawMdx });

  return {
    frontmatter,
    serialized,
    headings,
  };
};

export const getChangelog = async (filepath: string): Promise<Changelog<ChangeLogFrontmatter>> => {
  const rawMdx = await raw({ contentPath: CHANGELOG_PATH, filepath: filepath });
  // Serialize the MDX content and parse the frontmatter
  const serialized = await mdxSerialized({ rawMdx });
  const frontmatter = serialized.frontmatter as ChangeLogFrontmatter;

  return {
    frontmatter,
    serialized,
  };
};

export const getPolicy = async (filepath: string): Promise<Policy<PolicyFrontmatter>> => {
  const rawMdx = await raw({ contentPath: POLICY_PATH, filepath: filepath });
  // Serialize the MDX content and parse the frontmatter
  const serialized = await mdxSerialized({ rawMdx });
  const frontmatter = serialized.frontmatter as PolicyFrontmatter;

  return {
    frontmatter,
    serialized,
  };
};

export const getJob = async (filepath: string): Promise<Job<JobFrontmatter>> => {
  const rawMdx = await raw({ contentPath: JOBS_PATH, filepath: filepath });
  // Serialize the MDX content and parse the frontmatter
  const serialized = await mdxSerialized({ rawMdx });
  const frontmatter = serialized.frontmatter as JobFrontmatter;

  return {
    frontmatter,
    serialized,
  };
};

export const getAllJobsData = async ({
  contentPath,
}: {
  contentPath: string;
}) => {
  const allJobs = await fs.readdir(contentPath);
  const allJosFiltered = allJobs.filter((path) => /\.mdx?$/.test(path));
  const allPostsData = await Promise.all(
    allJosFiltered.map(async (post) => {
      const rawMdx = await raw({
        contentPath,
        filepath: post.replace(/\.mdx?$/, ""),
      });
      const serializedMdx = await mdxSerialized({ rawMdx });
      const frontmatter = serializedMdx.frontmatter as JobFrontmatter;
      if (frontmatter.visible === false) {
        return;
      }
      return {
        frontmatter,
        slug: post.replace(/\.mdx$/, ""),
      };
    }),
  );
  return allPostsData;
};
