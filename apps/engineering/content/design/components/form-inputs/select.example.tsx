"use client";

import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import {
  CircleWarning,
  TriangleWarning,
  CircleCheck,
  Envelope,
} from "@unkey/icons";
import { useState } from "react";

export function SelectExample() {
  const [value, setValue] = useState<string>("");

  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`{/* Basic Select */}
<div className="space-y-2">
  <h3 className="text-sm font-medium">Basic Select</h3>
  <Select value={value} onValueChange={setValue}>
    <SelectTrigger variant="default">
      <SelectValue placeholder="Select 1" />
    </SelectTrigger>
    <SelectContent>
      <SelectItem value="option1">Option 1</SelectItem>
      <SelectItem value="option2">Option 2</SelectItem>
      <SelectItem value="option3">Option 3</SelectItem>
    </SelectContent>
  </Select>
</div>

{/* Select with Left Icon */}
<div className="space-y-2">
  <h3 className="text-sm font-medium">Select with Left Icon</h3>
  <Select>
    <SelectTrigger variant="default" leftIcon={<Envelope className="w-4 h-4" />}>
      <SelectValue placeholder="Select 1" />
    </SelectTrigger>
    <SelectContent>
      <SelectItem value="personal">Personal Email</SelectItem>
      <SelectItem value="work">Work Email</SelectItem>
    </SelectContent>
  </Select>
</div>

{/* Select with Groups */}
<div className="space-y-2">
  <h3 className="text-sm font-medium">Select with Groups</h3>
  <Select>
    <SelectTrigger variant="default">
      <SelectValue placeholder="Select 1" />
    </SelectTrigger>
    <SelectContent>
      <SelectGroup>
        <SelectLabel>Fruits</SelectLabel>
        <SelectItem value="apple">Apple</SelectItem>
        <SelectItem value="banana">Banana</SelectItem>
        <SelectItem value="orange">Orange</SelectItem>
      </SelectGroup>
      <SelectGroup>
        <SelectLabel>Vegetables</SelectLabel>
        <SelectItem value="carrot">Carrot</SelectItem>
        <SelectItem value="potato">Potato</SelectItem>
        <SelectItem value="tomato">Tomato</SelectItem>
      </SelectGroup>
    </SelectContent>
  </Select>
</div>

{/* Disabled Select */}
<div className="space-y-2">
  <h3 className="text-sm font-medium">Disabled Select</h3>
  <Select disabled>
    <SelectTrigger variant="default">
      <SelectValue placeholder="Disabled select" />
    </SelectTrigger>
    <SelectContent>
      <SelectItem value="disabled1">Option 1</SelectItem>
      <SelectItem value="disabled2">Option 2</SelectItem>
    </SelectContent>
  </Select>
</div>`}
    >
      <Row>
        {/* Basic Select */}
        <div className="space-y-2">
          <h3 className="text-sm font-medium">Basic Select</h3>
          <Select value={value} onValueChange={setValue}>
            <SelectTrigger variant="default">
              <SelectValue placeholder="Select 1" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="option1">Option 1</SelectItem>
              <SelectItem value="option2">Option 2</SelectItem>
              <SelectItem value="option3">Option 3</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* Select with Left Icon */}
        <div className="space-y-2">
          <h3 className="text-sm font-medium">Select with Left Icon</h3>
          <Select>
            <SelectTrigger
              variant="default"
              leftIcon={<Envelope className="w-4 h-4" />}
            >
              <SelectValue placeholder="Select 1" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="personal">Personal Email</SelectItem>
              <SelectItem value="work">Work Email</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* Select with Groups */}
        <div className="space-y-2">
          <h3 className="text-sm font-medium">Select with Groups</h3>
          <Select>
            <SelectTrigger variant="default">
              <SelectValue placeholder="Select 1" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectLabel>Fruits</SelectLabel>
                <SelectItem value="apple">Apple</SelectItem>
                <SelectItem value="banana">Banana</SelectItem>
                <SelectItem value="orange">Orange</SelectItem>
              </SelectGroup>
              <SelectGroup>
                <SelectLabel>Vegetables</SelectLabel>
                <SelectItem value="carrot">Carrot</SelectItem>
                <SelectItem value="potato">Potato</SelectItem>
                <SelectItem value="tomato">Tomato</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>

        {/* Disabled Select */}
        <div className="space-y-2">
          <h3 className="text-sm font-medium">Disabled Select</h3>
          <Select disabled>
            <SelectTrigger variant="default">
              <SelectValue placeholder="Disabled select" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="disabled1">Option 1</SelectItem>
              <SelectItem value="disabled2">Option 2</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </Row>
    </RenderComponentWithSnippet>
  );
}

export function SelectExampleVariants() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`{/* Select with Variants */}
<div className="flex flex-row justify-between w-full gap-4">
  <Select>
    <SelectTrigger variant="success" leftIcon={<CircleCheck className="w-4 h-4" />}>
      <SelectValue placeholder="Success state" />
    </SelectTrigger>
    <SelectContent>
      <SelectItem value="success1">Success Option 1</SelectItem>
      <SelectItem value="success2">Success Option 2</SelectItem>
    </SelectContent>
  </Select>

  <Select>
    <SelectTrigger variant="warning" leftIcon={<TriangleWarning className="w-4 h-4" />}>
      <SelectValue placeholder="Warning state" />
    </SelectTrigger>
    <SelectContent>
      <SelectItem value="warning1">Warning Option 1</SelectItem>
      <SelectItem value="warning2">Warning Option 2</SelectItem>
    </SelectContent>
  </Select>

  <Select>
    <SelectTrigger variant="error" leftIcon={<CircleeWarning className="w-4 h-4" />}>
      <SelectValue placeholder="Error state" />
    </SelectTrigger>
    <SelectContent>
      <SelectItem value="error1">Error Option 1</SelectItem>
      <SelectItem value="error2">Error Option 2</SelectItem>
    </SelectContent>
  </Select>
</div>`}
    >
      <Row>
        {/* Select with Variants */}
        <div className="flex flex-row justify-between w-full gap-4">
          <Select>
            <SelectTrigger
              variant="success"
              leftIcon={<CircleCheck className="w-4 h-4" />}
            >
              <SelectValue placeholder="Success state" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="success1">Success Option 1</SelectItem>
              <SelectItem value="success2">Success Option 2</SelectItem>
            </SelectContent>
          </Select>

          <Select>
            <SelectTrigger
              variant="warning"
              leftIcon={<TriangleWarning className="w-4 h-4" />}
            >
              <SelectValue placeholder="Warning state" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="warning1">Warning Option 1</SelectItem>
              <SelectItem value="warning2">Warning Option 2</SelectItem>
            </SelectContent>
          </Select>

          <Select>
            <SelectTrigger
              variant="error"
              leftIcon={<CircleWarning className="w-4 h-4" />}
            >
              <SelectValue placeholder="Error state" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="error1">Error Option 1</SelectItem>
              <SelectItem value="error2">Error Option 2</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </Row>
    </RenderComponentWithSnippet>
  );
}
