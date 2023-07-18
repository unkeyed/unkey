//@ts-nocheck 
import { MDXComponents } from '@/components/landingComponents/MDXComponents'

export function useMDXComponents(components) {
  return {
    ...components,
    ...MDXComponents,
  }
}
