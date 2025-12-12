import { fetchJSON } from './fetchUtils';

export interface AISettings {
  currentModel: string;
  availableModels: string[];
  temperature: number;
  maxTokens: number;
}

class AISettingsService {
  // Get current AI settings
  async getSettings(): Promise<AISettings> {
    const response = await fetchJSON<{ settings: AISettings }>('/ai/settings', {
      method: 'GET',
    });

    if (!response.settings) {
      throw new Error('Failed to fetch AI settings');
    }

    return response.settings;
  }

  // Get list of available models
  async getAvailableModels(): Promise<string[]> {
    const response = await fetchJSON<{ models: string[] }>('/ai/models', {
      method: 'GET',
    });

    if (!response.models) {
      throw new Error('Failed to fetch available models');
    }

    return response.models;
  }

  // Set current model
  async setCurrentModel(model: string): Promise<AISettings> {
    const response = await fetchJSON<{ settings: AISettings }>('/ai/settings', {
      method: 'PUT',
      body: JSON.stringify({ currentModel: model }),
    });

    if (!response.settings) {
      throw new Error('Failed to update current model');
    }

    return response.settings;
  }

  // Update AI settings
  async updateSettings(settings: Partial<AISettings>): Promise<AISettings> {
    const response = await fetchJSON<{ settings: AISettings }>('/ai/settings', {
      method: 'PUT',
      body: JSON.stringify(settings),
    });

    if (!response.settings) {
      throw new Error('Failed to update AI settings');
    }

    return response.settings;
  }
}

export const aiSettingsService = new AISettingsService();