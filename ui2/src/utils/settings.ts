export interface Settings {
  theme: 'light' | 'dark' | 'system';
  displayMode: 'comfortable' | 'compact';
  fontSize: 'small' | 'medium' | 'large';
  apiBaseUrl: string;
  apiTimeout: number;
  debugMode: boolean;
  showApiRequests: boolean;
}

const SETTINGS_KEY = 'hyperion-settings';

const defaultSettings: Settings = {
  theme: 'system',
  displayMode: 'comfortable',
  fontSize: 'medium',
  apiBaseUrl: '',
  apiTimeout: 30000,
  debugMode: false,
  showApiRequests: false,
};

export function getSettings(): Settings {
  const stored = localStorage.getItem(SETTINGS_KEY);
  if (!stored) return defaultSettings;
  return { ...defaultSettings, ...JSON.parse(stored) };
}

export function saveSettings(settings: Partial<Settings>): void {
  const current = getSettings();
  const updated = { ...current, ...settings };
  localStorage.setItem(SETTINGS_KEY, JSON.stringify(updated));
}

export function resetSettings(): void {
  localStorage.removeItem(SETTINGS_KEY);
}
