export type CacheNamespaces = {
  completion: {
    id: string;
    content: string;
  };
};

export type CacheNamespace = keyof CacheNamespaces;
