import type { IconProps } from "@unkey/icons";
import {
  BracketsCurly,
  CircleDotted,
  Code,
  LangElixir,
  LangGo,
  LangJava,
  LangJavascript,
  LangPhp,
  LangPython,
  LangRuby,
  LangRust,
  LangTypescript,
} from "@unkey/icons";

const languageIconMap: Record<string, (props: IconProps) => React.JSX.Element> = {
  TypeScript: LangTypescript,
  JavaScript: LangJavascript,
  Python: LangPython,
  Go: LangGo,
  Rust: LangRust,
  Java: LangJava,
  Ruby: LangRuby,
  PHP: LangPhp,
  Elixir: LangElixir,
};

export const LanguageIcon = ({ language }: { language: string | null }) => {
  const Icon = language ? languageIconMap[language] : undefined;

  return Icon ? (
    <div className="size-10 grid place-content-center mr-11">
      <Icon className="size-[26px]" />
    </div>
  ) : (
    <div className="size-10 grid place-content-center mr-11 ml-1">
      <div className="size-[26px] grid place-content-center rounded-lg border border-grayA-5 shadow-md">
        <BracketsCurly iconSize="sm-medium" className="text-gray-9" />
      </div>
    </div>
  );
};
