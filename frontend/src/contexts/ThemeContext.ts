import {createContext} from 'react';

export interface ThemeContextValue {
    darkMode: boolean;
    toggleDarkMode: () => void;
}

export const ThemeContext = createContext<ThemeContextValue>({
    darkMode: false,
    toggleDarkMode: () => {},
});
