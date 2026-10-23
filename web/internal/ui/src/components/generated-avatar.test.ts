import { type ReactElement, createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { expect, test } from "vitest";
import { GeneratedAvatar } from "./generated-avatar";

// Golden SVGs rendered by boring-avatars 2.0.4 before it was vendored. Existing avatars must not change.
const BEAM_KEBAP =
  '<svg viewBox="0 0 36 36" fill="none" role="img" xmlns="http://www.w3.org/2000/svg" width="100%" height="100%"><mask id="ID" maskUnits="userSpaceOnUse" x="0" y="0" width="36" height="36"><rect width="36" height="36" rx="72" fill="#FFFFFF"></rect></mask><g mask="url(#ID)"><rect width="36" height="36" fill="#C271B4"></rect><rect x="0" y="0" width="36" height="36" transform="translate(5 5) rotate(55 18 18) scale(1.1)" fill="#92A1C6" rx="6"></rect><g transform="translate(7 1) rotate(5 18 18)"><path d="M13,20 a1,0.75 0 0,0 10,0" fill="#000000"></path><rect x="14" y="14" width="1.5" height="2" rx="1" stroke="none" fill="#000000"></rect><rect x="20" y="14" width="1.5" height="2" rx="1" stroke="none" fill="#000000"></rect></g></g></svg>';
const MARBLE_KEBAP_SQUARE =
  '<svg viewBox="0 0 80 80" fill="none" role="img" xmlns="http://www.w3.org/2000/svg" width="100%" height="100%"><mask id="ID" maskUnits="userSpaceOnUse" x="0" y="0" width="80" height="80"><rect width="80" height="80" fill="#FFFFFF"></rect></mask><g mask="url(#ID)"><rect width="80" height="80" fill="#92A1C6"></rect><path filter="url(#filter_ID)" d="M32.414 59.35L50.376 70.5H72.5v-71H33.728L26.5 13.381l19.057 27.08L32.414 59.35z" fill="#146A7C" transform="translate(6 6) rotate(110 40 40) scale(1.3)"></path><path filter="url(#filter_ID)" style="mix-blend-mode:overlay" d="M22.216 24L0 46.75l14.108 38.129L78 86l-3.081-59.276-22.378 4.005 12.972 20.186-23.35 27.395L22.215 24z" fill="#F0AB3D" transform="translate(-5 5) rotate(-165 40 40) scale(1.3)"></path></g><defs><filter id="filter_ID" filterUnits="userSpaceOnUse" color-interpolation-filters="sRGB"><feFlood flood-opacity="0" result="BackgroundImageFix"></feFlood><feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape"></feBlend><feGaussianBlur stdDeviation="7" result="effect1_foregroundBlur"></feGaussianBlur></filter></defs></svg>';

function renderWithStableIds(element: ReactElement) {
  const svg = renderToStaticMarkup(element);
  const id = svg.match(/<mask id="([^"]+)"/)?.[1];
  expect(id).toBeDefined();
  return svg.split(id ?? "").join("ID");
}

test("beam matches boring-avatars", () => {
  expect(
    renderWithStableIds(createElement(GeneratedAvatar, { name: "KEBAP", variant: "beam" })),
  ).toBe(BEAM_KEBAP);
});

test("square marble matches boring-avatars", () => {
  expect(
    renderWithStableIds(
      createElement(GeneratedAvatar, { name: "KEBAP", variant: "marble", square: true }),
    ),
  ).toBe(MARBLE_KEBAP_SQUARE);
});
