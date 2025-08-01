// SPDX-License-Identifier: MPL-2.0

import React from 'react';

export const DarkModeContext = React.createContext<{
  isDarkMode: boolean;
  setDarkMode: (isDark: boolean) => void;
}>({
  isDarkMode: false,
  setDarkMode: () => {},
});




