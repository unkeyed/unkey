"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { DialogContainer } from "@unkey/ui";
import { Button, Input } from "@unkey/ui";
import { useState } from "react";

export function DialogContainerExample() {
  const [isOpen, setIsOpen] = useState(false);
  const [inputValue, setInputValue] = useState("");
  const [inputResult, setInputResult] = useState("");

  const handleSubmit = () => {
    setInputResult(inputValue);
    setIsOpen(false);
  };

  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-row gap-2 justify-center">
  <div className="flex flex-col gap-2 w-[200px]">
    <Button className="text-gray-11 text-[13px]" onClick={() => setIsOpen(!isOpen)}>
      Open Dialog
    </Button>

    <DialogContainer
      isOpen={isOpen}
      onOpenChange={() => setIsOpen(!isOpen)}
      subTitle="This is an example of a subTitle. Normally used to describe the dialog"
      title="Example Dialog Title"
      footer={
        <div className="flex flex-col w-full gap-2 items-center justify-center">
          <Button
            type="submit"
            variant="primary"
            size="lg"
            className="w-full"
            onClick={() => handleSubmit()}
          >
            Submit
          </Button>
          <div className="text-error-11 text-xs">
            This is an example of a footer with a button for actions needed to be done
          </div>
        </div>
      }
    >
      <div className="flex flex-col text-gray-11 text-[13px] gap-2">
        <p>Dialog Content</p>
        <Input
          placeholder="Example Input"
          type="text"
          onChange={(e) => setInputValue(e.target.value)}
        />
      </div>
    </DialogContainer>
    <p>
      Input Result: <span className="text-success-6">{inputResult}</span>
    </p>
  </div>
</div>`}
    >
      <div className="flex flex-row gap-2 justify-center">
        <div className="flex flex-col gap-2 w-[200px]">
          <Button className="text-gray-11 text-[13px]" onClick={() => setIsOpen(!isOpen)}>
            Open Dialog
          </Button>

          <DialogContainer
            isOpen={isOpen}
            onOpenChange={() => setIsOpen(!isOpen)}
            subTitle="This is an example of a subTitle. Normally used to describe the dialog"
            title="Example Dialog Title"
            footer={
              <div className="flex flex-col w-full gap-2 items-center justify-center">
                <Button
                  type="submit"
                  variant="primary"
                  size="lg"
                  className="w-full"
                  onClick={() => handleSubmit()}
                >
                  Submit
                </Button>
                <div className="text-error-11 text-xs">
                  This is an example of a footer with a button for actions needed to be done
                </div>
              </div>
            }
          >
            <div className="flex flex-col text-gray-11 text-[13px] gap-2">
              <p>Dialog Content</p>
              <Input
                placeholder="Example Input"
                type="text"
                onChange={(e) => setInputValue(e.target.value)}
              />
            </div>
          </DialogContainer>
          <p>
            Input Result: <span className="text-success-6">{inputResult}</span>
          </p>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}
