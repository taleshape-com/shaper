// Tremor Raw chartColors [v0.1.0]

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
) => {
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
