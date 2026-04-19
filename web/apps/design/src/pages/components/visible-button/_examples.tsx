import { Code, VisibleButton } from "@unkey/ui";
import { useState } from "react";
import { Preview } from "../../../components/Preview";

const SECRET = "unkey_3ZaM7pQ1xK9VdLb2wTgRcY";
const mask = (value: string) => "\u2022".repeat(value.length);

export function StandaloneExample() {
  const [isVisible, setIsVisible] = useState(false);

  return (
    <Preview>
      <VisibleButton
        isVisible={isVisible}
        setIsVisible={setIsVisible}
        title="API key"
      />
    </Preview>
  );
}

export function InsideCodeExample() {
  const [isVisible, setIsVisible] = useState(false);

  return (
    <Preview>
      <Code
        visibleButton={
          <VisibleButton
            isVisible={isVisible}
            setIsVisible={setIsVisible}
            title="API key"
          />
        }
      >
        <span>{isVisible ? SECRET : mask(SECRET)}</span>
      </Code>
    </Preview>
  );
}

export function VariantExample() {
  const [isVisible, setIsVisible] = useState(false);

  return (
    <Preview>
      <VisibleButton
        isVisible={isVisible}
        setIsVisible={setIsVisible}
        title="signing secret"
        variant="ghost"
      />
    </Preview>
  );
}
