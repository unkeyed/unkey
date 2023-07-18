//@ts-nocheck 
import glob from 'fast-glob'

const exportNames = {
  blog: 'article',
  changelog : 'changelog',
}

export async function loadMDXMetadata(directory : 'blog' | 'changelog') {
  return (
    await Promise.all(
      (
        await glob('**/page.mdx', { cwd: `app/(landing)/${directory}` })
      ).map(async (filename: string) => {
        let id = filename.replace(/\/page\.mdx$/, '')
        return {
          id,
          href: `/${directory}/${id}`,
          ...(await import(`app/(landing)/${directory}/${filename}`))[
            exportNames[directory]
          ],
        }
      })
    )
  ).sort((a, b) => b.date.localeCompare(a.date))
}
