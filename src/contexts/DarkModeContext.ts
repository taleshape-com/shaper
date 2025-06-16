import React from 'react';

export const DarkModeContext = React.createContext<{
  isDarkMode: boolean;
  setDarkMode: (isDark: boolean) => void;
}>({
  isDarkMode: false,
  setDarkMode: () => {},
});




