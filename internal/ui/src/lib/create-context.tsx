import React from "react";

export function createContext<ContextValueType extends object | null>(
  rootComponentName: string,
  defaultContext?: ContextValueType,
) {
  const Context = React.createContext<ContextValueType | undefined>(defaultContext);

  const Provider = (props: ContextValueType & { children: React.ReactNode }) => {
    const { children } = props;
    // Only re-memoize when actual prop values change, not the object reference
    // biome-ignore lint/correctness/useExhaustiveDependencies: props object reference changes every render; we track individual prop values instead
    const value = React.useMemo(
      () => {
        const { children: _, ...contextValue } = props;
        return contextValue as ContextValueType;
      },
      Object.keys(props)
        .filter((k) => k !== "children")
        .map((k) => (props as Record<string, unknown>)[k]),
    );
    return <Context.Provider value={value}>{children}</Context.Provider>;
  };

  function useContext(consumerName: string) {
    const context = React.useContext(Context);
    if (context !== undefined) {
      return context;
    }
    if (defaultContext !== undefined) {
      return defaultContext;
    }
    // if a defaultContext wasn't specified, it's a required context.
    throw new Error(`\`${consumerName}\` must be used within \`${rootComponentName}\``);
  }

  Provider.displayName = `${rootComponentName}Provider`;
  return [Provider, useContext] as const;
}
