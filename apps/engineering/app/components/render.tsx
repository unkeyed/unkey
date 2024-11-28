import { CodeBlock, Pre } from "fumadocs-ui/components/codeblock";
import type { PropsWithChildren } from "react";
import reactElementToJSXString from "react-element-to-jsx-string";

export const RenderComponentWithSnippet: React.FC<PropsWithChildren> = (props) => {
  return (
    <div>
      {props.children}
      <CodeBlock>
        <Pre>
          {reactElementToJSXString(props.children, {
            showFunctions: true,
            useBooleanShorthandSyntax: true,
          })}
        </Pre>
      </CodeBlock>
    </div>
  );
};
