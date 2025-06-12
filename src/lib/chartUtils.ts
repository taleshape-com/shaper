// Tremor Raw chartColors [v0.1.0]

import { AxisDomain } from "recharts/types/util/types"

export type ColorUtility = "bg" | "stroke" | "fill" | "text"

export const chartColors = {
  primary: {
    bg: "bg-cprimary",
    stroke: "stroke-cprimary",
    fill: "fill-cprimary",
    text: "text-cprimary",
  },
  color2: {
    bg: "bg-ctwo",
    stroke: "stroke-ctwo",
    fill: "fill-ctwo",
    text: "text-ctwo",
  },
  color3: {
    bg: "bg-cthree",
    stroke: "stroke-cthree",
    fill: "fill-cthree",
    text: "text-cthree",
  },
  color4: {
    bg: "bg-cfour",
    stroke: "stroke-cfour",
    fill: "fill-cfour",
    text: "text-cfour",
  },
  color5: {
    bg: "bg-cfive",
    stroke: "stroke-cfive",
    fill: "fill-cfive",
    text: "text-cfive",
  },
  color6: {
    bg: "bg-csix",
    stroke: "stroke-csix",
    fill: "fill-csix",
    text: "text-csix",
  },
  color7: {
    bg: "bg-cseven",
    stroke: "stroke-cseven",
    fill: "fill-cseven",
    text: "text-cseven",
  },
  color8: {
    bg: "bg-ceight",
    stroke: "stroke-ceight",
    fill: "fill-ceight",
    text: "text-ceight",
  },
  color9: {
    bg: "bg-cnine",
    stroke: "stroke-cnine",
    fill: "fill-cnine",
    text: "text-cnine",
  },
  color10: {
    bg: "bg-cten",
    stroke: "stroke-cten",
    fill: "fill-cten",
    text: "text-cten",
  },
  // indigo: {
  //   bg: "bg-indigo-500",
  //   stroke: "stroke-indigo-500",
  //   fill: "fill-indigo-500",
  //   text: "text-indigo-500",
  // },
  // emerald: {
  //   bg: "bg-emerald-300",
  //   stroke: "stroke-emerald-300",
  //   fill: "fill-emerald-300",
  //   text: "text-emerald-300",
  // },
  // pink: {
  //   bg: "bg-pink-400",
  //   stroke: "stroke-pink-400",
  //   fill: "fill-pink-400",
  //   text: "text-pink-400",
  // },
  // yellow: {
  //   bg: "bg-yellow-200",
  //   stroke: "stroke-yellow-200",
  //   fill: "fill-yellow-200",
  //   text: "text-yellow-200",
  // },
  // sky: {
  //   bg: "bg-sky-300",
  //   stroke: "stroke-sky-300",
  //   fill: "fill-sky-300",
  //   text: "text-sky-300",
  // },
  violet: {
    bg: "bg-violet-400",
    stroke: "stroke-violet-400",
    fill: "fill-violet-400",
    text: "text-violet-400",
  },
  blue: {
    bg: "bg-blue-500",
    stroke: "stroke-blue-500",
    fill: "fill-blue-500",
    text: "text-blue-500",
  },
  fuchsia: {
    bg: "bg-fuchsia-500",
    stroke: "stroke-fuchsia-500",
    fill: "fill-fuchsia-500",
    text: "text-fuchsia-500",
  },
  amber: {
    bg: "bg-amber-500",
    stroke: "stroke-amber-500",
    fill: "fill-amber-500",
    text: "text-amber-500",
  },
  cyan: {
    bg: "bg-cyan-500",
    stroke: "stroke-cyan-500",
    fill: "fill-cyan-500",
    text: "text-cyan-500",
  },
  gray: {
    bg: "bg-gray-500",
    stroke: "stroke-gray-500",
    fill: "fill-gray-500",
    text: "text-gray-500",
  },
  lime: {
    bg: "bg-lime-500",
    stroke: "stroke-lime-500",
    fill: "fill-lime-500",
    text: "text-lime-500",
  },
} as const satisfies {
  [color: string]: {
    [key in ColorUtility]: string
  }
}

export type AvailableChartColorsKeys = keyof typeof chartColors

export const AvailableChartColors: AvailableChartColorsKeys[] = Object.keys(
  chartColors,
) as Array<AvailableChartColorsKeys>

export const constructCategoryColors = (
  categories: string[],
  colors: AvailableChartColorsKeys[],
): Map<string, AvailableChartColorsKeys> => {
  const categoryColors = new Map<string, AvailableChartColorsKeys>()
  categories.forEach((category, index) => {
    categoryColors.set(category, colors[index % colors.length])
  })
  return categoryColors
}

export const getColorClassName = (
  color: AvailableChartColorsKeys,
  type: ColorUtility,
): string => {
  const fallbackColor = {
    bg: "bg-gray-500",
    stroke: "stroke-gray-500",
    fill: "fill-gray-500",
    text: "text-gray-500",
  }
  return chartColors[color]?.[type] ?? fallbackColor[type]
}

// Tremor Raw getYAxisDomain [v0.0.0]

export const getYAxisDomain = (
  autoMinValue: boolean,
  minValue: number | undefined,
  maxValue: number | undefined,
): AxisDomain => {
  const minDomain = autoMinValue ? "auto" : minValue ?? 0
  const maxDomain = maxValue ?? "auto"
  return [minDomain, maxDomain]
}

// Tremor Raw hasOnlyOneValueForKey [v0.1.0]

export function hasOnlyOneValueForKey(
  array: any[],
  keyToCheck: string,
): boolean {
  const val: any[] = []

  for (const obj of array) {
    if (Object.prototype.hasOwnProperty.call(obj, keyToCheck)) {
      val.push(obj[keyToCheck])
      if (val.length > 1) {
        return false
      }
    }
  }

  return true
}

// ECharts color utilities
export const echartsColors = {
  primary: {
    light: 'var(--shaper-primary-color)',
    dark: 'var(--shaper-dark-mode-primary-color)',
  },
  color2: {
    light: 'var(--shaper-color-two)',
    dark: 'var(--shaper-color-two)',
  },
  color3: {
    light: 'var(--shaper-color-three)',
    dark: 'var(--shaper-color-three)',
  },
  color4: {
    light: 'var(--shaper-color-four)',
    dark: 'var(--shaper-color-four)',
  },
  color5: {
    light: 'var(--shaper-color-five)',
    dark: 'var(--shaper-color-five)',
  },
  color6: {
    light: 'var(--shaper-color-six)',
    dark: 'var(--shaper-color-six)',
  },
  color7: {
    light: 'var(--shaper-color-seven)',
    dark: 'var(--shaper-color-seven)',
  },
  color8: {
    light: 'var(--shaper-color-eight)',
    dark: 'var(--shaper-color-eight)',
  },
  color9: {
    light: 'var(--shaper-color-nine)',
    dark: 'var(--shaper-color-nine)',
  },
  color10: {
    light: 'var(--shaper-color-ten)',
    dark: 'var(--shaper-color-ten)',
  },
} as const;

export type EChartsColorKey = keyof typeof echartsColors;

export const AvailableEChartsColors: EChartsColorKey[] = Object.keys(
  echartsColors,
) as Array<EChartsColorKey>;

export const constructEChartsCategoryColors = (
  categories: string[],
  colors: EChartsColorKey[],
): Map<string, EChartsColorKey> => {
  const categoryColors = new Map<string, EChartsColorKey>();
  categories.forEach((category, index) => {
    categoryColors.set(category, colors[index % colors.length]);
  });
  return categoryColors;
};

export const getEChartsColor = (
  colorKey: EChartsColorKey,
  isDark: boolean = false,
): string => {
  const color = echartsColors[colorKey];
  if (!color) {
    const fallbackVar = isDark ? '--shaper-dark-mode-primary-color' : '--shaper-primary-color';
    const root = document.documentElement;
    const computedValue = getComputedStyle(root).getPropertyValue(fallbackVar).trim();
    return computedValue || `var(${fallbackVar})`;
  }
  
  const cssVar = isDark ? color.dark : color.light;
  // Extract the CSS variable name from the var() function
  const varName = cssVar.replace('var(', '').replace(')', '');
  const root = document.documentElement;
  const computedValue = getComputedStyle(root).getPropertyValue(varName).trim();
  
  return computedValue || cssVar;
};
