import { Slider } from "@unkey/ui";
import { useState } from "react";
import { Preview } from "../../../components/Preview";

export function BasicExample() {
  const [value, setValue] = useState([40]);
  return (
    <Preview>
      <div className="w-72 flex flex-col gap-3">
        <div className="flex items-center justify-between text-xs text-gray-11">
          <span>Volume</span>
          <span className="font-mono text-gray-12">{value[0]}</span>
        </div>
        <Slider
          value={value}
          onValueChange={setValue}
          min={0}
          max={100}
          aria-label="Volume"
        />
      </div>
    </Preview>
  );
}

export function RangeExample() {
  const [range, setRange] = useState([20, 75]);
  return (
    <Preview>
      <div className="w-72 flex flex-col gap-3">
        <div className="flex items-center justify-between text-xs text-gray-11">
          <span>Price range</span>
          <span className="font-mono text-gray-12">
            ${range[0]} – ${range[1]}
          </span>
        </div>
        <Slider
          value={range}
          onValueChange={setRange}
          min={0}
          max={100}
          minStepsBetweenThumbs={1}
          aria-label="Price range"
        />
      </div>
    </Preview>
  );
}

export function StepExample() {
  const [coarse, setCoarse] = useState([50]);
  const [fine, setFine] = useState([0.5]);
  return (
    <Preview>
      <div className="w-72 flex flex-col gap-6">
        <div className="flex flex-col gap-3">
          <div className="flex items-center justify-between text-xs text-gray-11">
            <span>Step 10</span>
            <span className="font-mono text-gray-12">{coarse[0]}</span>
          </div>
          <Slider
            value={coarse}
            onValueChange={setCoarse}
            min={0}
            max={100}
            step={10}
            aria-label="Coarse slider"
          />
        </div>
        <div className="flex flex-col gap-3">
          <div className="flex items-center justify-between text-xs text-gray-11">
            <span>Step 0.05</span>
            <span className="font-mono text-gray-12">{fine[0].toFixed(2)}</span>
          </div>
          <Slider
            value={fine}
            onValueChange={setFine}
            min={0}
            max={1}
            step={0.05}
            aria-label="Fine slider"
          />
        </div>
      </div>
    </Preview>
  );
}

export function DisabledExample() {
  const [value, setValue] = useState([35]);
  return (
    <Preview>
      <div className="w-72 flex flex-col gap-3">
        <div className="flex items-center justify-between text-xs text-gray-11">
          <span>Read-only</span>
          <span className="font-mono text-gray-12">{value[0]}</span>
        </div>
        <Slider
          value={value}
          onValueChange={setValue}
          min={0}
          max={100}
          disabled
          aria-label="Disabled slider"
        />
      </div>
    </Preview>
  );
}
