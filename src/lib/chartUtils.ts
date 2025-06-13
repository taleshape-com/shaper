// Helper function to get computed CSS value
export const getComputedCssValue = (cssVar: string): string => {
  const root = document.documentElement;
  const computedValue = getComputedStyle(root).getPropertyValue(cssVar).trim();
  return computedValue || `var(${cssVar})`;
};

// Function to detect if dark mode is active
export const isDarkMode = (): boolean => {
  // Check if the body has dark mode classes or if we're in a dark theme context
  const body = document.body;
  return body.classList.contains('dark') ||
    body.classList.contains('dark-mode') ||
    window.matchMedia('(prefers-color-scheme: dark)').matches;
};

// Function to get theme-appropriate colors
export const getThemeColors = () => {
  const isDark = isDarkMode();

  if (isDark) {
    return {
      backgroundColor: getComputedCssValue('--shaper-dark-mode-background-color'),
      borderColor: getComputedCssValue('--shaper-dark-mode-border-color'),
      textColor: getComputedCssValue('--shaper-dark-mode-text-color'),
      textColorSecondary: getComputedCssValue('--shaper-dark-mode-text-color-secondary'),
      referenceLineColor: getComputedCssValue('--shaper-reference-line-color'),
    };
  } else {
    return {
      backgroundColor: getComputedCssValue('--shaper-background-color'),
      borderColor: getComputedCssValue('--shaper-border-color'),
      textColor: getComputedCssValue('--shaper-text-color'),
      textColorSecondary: getComputedCssValue('--shaper-text-color-secondary'),
      referenceLineColor: getComputedCssValue('--shaper-reference-line-color'),
    };
  }
};

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
    return getComputedCssValue(fallbackVar);
  }

  const cssVar = isDark ? color.dark : color.light;
  // Extract the CSS variable name from the var() function
  const varName = cssVar.replace('var(', '').replace(')', '');

  return getComputedCssValue(varName);
};
