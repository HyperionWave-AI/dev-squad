import React, { useState, useEffect } from 'react';
import * as Switch from '@radix-ui/react-switch';
import * as Select from '@radix-ui/react-select';
import { Settings as SettingsIcon, Moon, Sun, Monitor } from 'lucide-react';
import { getSettings, saveSettings } from '@/utils/settings';
import type { Settings } from '@/utils/settings';
import { Button } from '@/components/atoms/Button';
import { Input } from '@/components/atoms/Input';
import { Label } from '@/components/atoms/Label';
import { PageHeader } from '@/components/organisms/PageHeader';

export function SettingsPage() {
  const [settings, setSettings] = useState<Settings>(getSettings());

  const handleSave = () => {
    saveSettings(settings);
    alert('Settings saved successfully!');
  };

  const handleReset = () => {
    if (confirm('Reset all settings to defaults?')) {
      localStorage.removeItem('hyperion-settings');
      setSettings(getSettings());
    }
  };

  const updateSetting = <K extends keyof Settings>(key: K, value: Settings[K]) => {
    setSettings({ ...settings, [key]: value });
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 via-white to-gray-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
      <div className="container mx-auto p-6 space-y-6 max-w-7xl">
        {/* Header */}
        <PageHeader
          title="Settings"
          description="Configure your preferences and application settings"
          icon={<SettingsIcon className="h-8 w-8" />}
          gradientFrom="#64748b"
          gradientTo="#475569"
        />

        {/* Appearance Section - Glassmorphic Container */}
        <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-6 shadow-lg">
          <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">Appearance</h3>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <Label>Theme</Label>
                <p className="text-sm text-gray-600 dark:text-gray-400">Choose your preferred color scheme</p>
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => updateSetting('theme', 'light')}
                  className={`p-2 rounded-md transition-colors ${settings.theme === 'light' ? 'bg-blue-500 text-white' : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300'}`}
                >
                  <Sun className="h-4 w-4" />
                </button>
                <button
                  onClick={() => updateSetting('theme', 'dark')}
                  className={`p-2 rounded-md transition-colors ${settings.theme === 'dark' ? 'bg-blue-500 text-white' : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300'}`}
                >
                  <Moon className="h-4 w-4" />
                </button>
                <button
                  onClick={() => updateSetting('theme', 'system')}
                  className={`p-2 rounded-md transition-colors ${settings.theme === 'system' ? 'bg-blue-500 text-white' : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300'}`}
                >
                  <Monitor className="h-4 w-4" />
                </button>
              </div>
            </div>

            <div className="flex items-center justify-between">
              <div>
                <Label>Display Mode</Label>
                <p className="text-sm text-gray-600 dark:text-gray-400">Spacing and density</p>
              </div>
              <Select.Root value={settings.displayMode} onValueChange={(v) => updateSetting('displayMode', v as any)}>
                <Select.Trigger className="px-3 py-2 border border-border rounded-md bg-background">
                  <Select.Value />
                </Select.Trigger>
                <Select.Portal>
                  <Select.Content className="bg-background border border-border rounded-md shadow-lg z-50">
                    <Select.Viewport className="p-1">
                      <Select.Item value="comfortable" className="px-3 py-2 hover:bg-accent cursor-pointer rounded">
                        <Select.ItemText>Comfortable</Select.ItemText>
                      </Select.Item>
                      <Select.Item value="compact" className="px-3 py-2 hover:bg-accent cursor-pointer rounded">
                        <Select.ItemText>Compact</Select.ItemText>
                      </Select.Item>
                    </Select.Viewport>
                  </Select.Content>
                </Select.Portal>
              </Select.Root>
            </div>
          </div>
        </div>

        {/* API Configuration Section - Glassmorphic Container */}
        <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-6 shadow-lg">
          <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">API Configuration</h3>
          <div className="space-y-4">
            <div>
              <Label>API Base URL</Label>
              <Input
                value={settings.apiBaseUrl}
                onChange={(e) => updateSetting('apiBaseUrl', e.target.value)}
                placeholder="http://localhost:5177"
              />
            </div>
            <div>
              <Label>API Timeout (ms)</Label>
              <Input
                type="number"
                value={settings.apiTimeout}
                onChange={(e) => updateSetting('apiTimeout', parseInt(e.target.value) || 30000)}
              />
            </div>
          </div>
        </div>

        {/* Developer Options Section - Glassmorphic Container */}
        <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-6 shadow-lg">
          <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">Developer Options</h3>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <Label>Debug Mode</Label>
                <p className="text-sm text-gray-600 dark:text-gray-400">Show detailed error messages</p>
              </div>
              <Switch.Root
                checked={settings.debugMode}
                onCheckedChange={(checked) => updateSetting('debugMode', checked)}
                className="w-11 h-6 bg-gray-300 dark:bg-gray-600 rounded-full relative data-[state=checked]:bg-blue-500"
              >
                <Switch.Thumb className="block w-5 h-5 bg-white rounded-full transition-transform data-[state=checked]:translate-x-5" />
              </Switch.Root>
            </div>
            <div className="flex items-center justify-between">
              <div>
                <Label>Show API Requests</Label>
                <p className="text-sm text-gray-600 dark:text-gray-400">Log API calls to console</p>
              </div>
              <Switch.Root
                checked={settings.showApiRequests}
                onCheckedChange={(checked) => updateSetting('showApiRequests', checked)}
                className="w-11 h-6 bg-gray-300 dark:bg-gray-600 rounded-full relative data-[state=checked]:bg-blue-500"
              >
                <Switch.Thumb className="block w-5 h-5 bg-white rounded-full transition-transform data-[state=checked]:translate-x-5" />
              </Switch.Root>
            </div>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex gap-2 justify-end">
          <Button variant="outline" onClick={handleReset}>Reset to Defaults</Button>
          <Button onClick={handleSave}>Save Settings</Button>
        </div>
      </div>
    </div>
  );
}
