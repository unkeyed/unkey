import { PropsWithChildren } from "react";

type Props =
  | {
      h1: true;
      h2?: never;
      h3?: never;
      h4?: never;
    }
  | {
      h1?: never;
      h2: true;
      h3?: never;
      h4?: never;
    }
  | {
      h1?: never;
      h2?: never;
      h3: true;
      h4?: never;
    }
  | {
      h1?: never;
      h2?: never;
      h3?: never;
      h4: true;
    };

export const Heading: React.FC<PropsWithChildren<Props>> = ({ children, h1, h2, h3, h4 }) => {
  if (h1) {
    return (
      <h1 className="text-4xl font-extrabold tracking-tight scroll-m-20 lg:text-5xl">{children}</h1>
    );
  }
  if (h2) {
    return (
      <h2 className="pb-2 mt-10 text-3xl font-semibold tracking-tight border-b transition-colors scroll-m-20 border-b-zinc-200 first:mt-0 dark:border-b-zinc-700">
        {children}{" "}
      </h2>
    );
  }
  if (h3) {
    return <h3 className="mt-8 text-2xl font-semibold tracking-tight scroll-m-20">{children}</h3>;
  }
  if (h4) {
    return <h4 className="mt-8 text-xl font-semibold tracking-tight scroll-m-20">{children} </h4>;
  }

  return null;
};
