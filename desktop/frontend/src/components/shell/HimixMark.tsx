import { forwardRef, useId } from "react";

/**
 * Stroke geometry of the Himix icon, traced on its 256px grid so the launch
 * animation can draw it. Grouped by the parts the launch sequence moves:
 * the trunk is drawn, then the two inbound links converge onto it.
 */
export const MARK_PATHS = {
  /** The merged egress: a thin run from the left that thickens past the merge point. */
  trunk: ["M58 128 H152", "M152 128 H196"],
  /** Two inbound links that converge onto the trunk. */
  arms: ["M58 76 H108 L152 128", "M58 180 H108 L152 128"],
} as const;

/** The icon's gradient, bottom-left cyan through to top-right indigo. */
const GRADIENT = ["#0ea5d8", "#4a54e1"] as const;

type Props = { className?: string; strokeWidth?: number };

export const HimixMark = forwardRef<SVGSVGElement, Props>(
  function HimixMark({ className, strokeWidth = 13 }, ref) {
    const gradientId = `${useId().replace(/:/g, "")}-mark`;
    const base = { fill: "none", stroke: `url(#${gradientId})`, strokeLinecap: "round", strokeLinejoin: "round" } as const;
    return (
      <svg ref={ref} className={className} viewBox="0 0 256 256" aria-hidden="true" focusable="false">
        <defs>
          <linearGradient id={gradientId} x1="0" y1="256" x2="256" y2="0" gradientUnits="userSpaceOnUse">
            {GRADIENT.map((color, index) => <stop key={color} offset={index / (GRADIENT.length - 1)} stopColor={color} />)}
          </linearGradient>
        </defs>
        <g data-part="trunk">
          <path d={MARK_PATHS.trunk[0]} {...base} strokeWidth={strokeWidth} opacity={0.85} />
          <path d={MARK_PATHS.trunk[1]} {...base} strokeWidth={strokeWidth * 1.7} />
        </g>
        <g data-part="arms">
          {MARK_PATHS.arms.map(d => <path key={d} d={d} {...base} strokeWidth={strokeWidth} opacity={0.62} />)}
        </g>
      </svg>
    );
  },
);
