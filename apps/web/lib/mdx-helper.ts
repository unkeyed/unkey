import { promises as fs } from "fs";
import path from "path";
import GithubSlugger from "github-slugger";
import { serialize } from "next-mdx-remote/serialize";
import rehypeAutolinkHeadings from "rehype-autolink-headings";
import rehypeCodeTitles from "rehype-code-titles";
import rehypePrettyCode from "rehype-pretty-code";
import rehypeSlug from "rehype-slug";
import remarkGfm from "remark-gfm";
import { BUNDLED_LANGUAGES, type HighlighterOptions, getHighlighter } from "shiki";
import gitHubLight from "shiki/themes/github-light.json";

export const BLOG_PATH = path.join(process.cwd(), "content", "blog");
export const CHANGELOG_PATH = path.join(process.cwd(), "content", "changelog");

export const postFilePaths = fs.readdir(BLOG_PATH);
export const changelogFilePaths = fs.readdir(CHANGELOG_PATH);

export const raw = async ({ contentPath, filepath }: { contentPath: string; filepath: string }) => {
  return await fs.readFile(`${contentPath}/${filepath}.mdx`, "utf-8");
};

type Frontmatter = {
  title: string;
  date: string;
  description: string;
  author: string;
};

const options = {
  theme: gitHubLight,
  getHighlighter: (options: HighlighterOptions) =>
    getHighlighter({
      ...options,
      langs: [...BUNDLED_LANGUAGES],
    }),
  defaultLang: {
    block: "typescript",
  },
};
// Serialize the MDX content and parse the frontmatter
export const mdxSerialized = async ({ rawMdx }: { rawMdx: string }) => {
  return await serialize(rawMdx, {
    parseFrontmatter: true,
    mdxOptions: {
      remarkPlugins: [remarkGfm],
      rehypePlugins: [
        [rehypePrettyCode, options],
        rehypeAutolinkHeadings,
        rehypeSlug,
        rehypeCodeTitles,
      ],
    },
  });
};

export const getHeadings = async ({ rawMdx }: { rawMdx: string }) => {
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

const getMorePosts = async ({
  contentPath,
  filepath,
}: { contentPath: string; filepath: string }) => {
  const morePosts = await fs.readdir(contentPath);
  const morePostsFiltered = morePosts
    .filter((path) => /\.mdx?$/.test(path))
    .filter((post) => post !== filepath)
    .slice(0, 2);
  return morePostsFiltered;
};
export const getMorePostsData = async ({
  contentPath,
  filepath,
}: { contentPath: string; filepath: string }) => {
  const morePosts = await getMorePosts({ contentPath, filepath });
  const morePostsData = await Promise.all(
    morePosts.map(async (post) => {
      const rawMdx = await raw({ contentPath, filepath: post.replace(/\.mdx?$/, "") });
      const serializedMdx = await mdxSerialized({ rawMdx });
      const frontmatter = serializedMdx.frontmatter as Frontmatter;
      return {
        frontmatter,
        slug: post.replace(/\.mdx$/, ""),
      };
    }),
  );
  return morePostsData;
};
