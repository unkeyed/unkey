import { Highlight, type PrismTheme } from "prism-react-renderer";

export function CodeEditor({
  codeBlock,
  language,
  theme,
}: { codeBlock: string; language: string; theme?: PrismTheme }) {
  return (
    <Highlight theme={theme} code={codeBlock} language={language}>
      {({ tokens, getLineProps, getTokenProps }) => {
        const lineCount = tokens.length;
        const gutterPadLength = Math.max(String(lineCount).length, 2);
        return (
          <pre
            key={codeBlock} // Use codeBlock as a key to trigger animations on change
            className="leading-6"
          >
            {tokens.map((line, i) => {
              const lineNumber = i + 1;
              const paddedLineGutter = String(lineNumber).padStart(gutterPadLength, " ");
              return (
                // biome-ignore lint/suspicious/noArrayIndexKey: I got nothing better right now
                <div key={`${line}-${i}`} {...getLineProps({ line })}>
                  <span className="select-none line-number">{paddedLineGutter}</span>
                  {line.map((token, key) => (
                    <span key={`${key}-${token}`} {...getTokenProps({ token })} />
                  ))}
                </div>
              );
            })}
          </pre>
        );
      }}
    </Highlight>
  );
}
