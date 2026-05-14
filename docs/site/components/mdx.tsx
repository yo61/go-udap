import defaultMdxComponents from 'fumadocs-ui/mdx';
import type { MDXComponents } from 'mdx/types';
import { HowTo } from './how-to';

export function getMDXComponents(components?: MDXComponents) {
  return {
    ...defaultMdxComponents,
    HowTo,
    ...components,
  } satisfies MDXComponents;
}

export const useMDXComponents = getMDXComponents;

declare global {
  type MDXProvidedComponents = ReturnType<typeof getMDXComponents>;
}
