/**
 * User Settings Service Client
 *
 * Provides REST API interface for managing user preferences
 * including conversation mode (debug vs default)
 */

const BASE_URL = '/api/v1';

export type ConversationMode = 'debug' | 'default';

export interface UserSettings {
  id: string;
  userId: string;
  companyId: string;
  conversationMode: ConversationMode;
  createdAt: string;
  updatedAt: string;
}

/**
 * User Settings Service Client class
 */
class UserSettingsService {
  /**
   * Generic fetch wrapper with error handling
   */
  private async fetchJSON<T>(
    endpoint: string,
    options?: RequestInit
  ): Promise<T> {
    const url = `${BASE_URL}${endpoint}`;

    try {
      const response = await fetch(url, {
        ...options,
        headers: {
          'Content-Type': 'application/json',
          ...options?.headers,
        },
      });

      if (!response.ok) {
        const errorText = await response.text();
        let errorMessage: string;
        try {
          const errorData = JSON.parse(errorText);
          errorMessage = errorData.error || errorData.message || `HTTP ${response.status}`;
        } catch {
          errorMessage = errorText || `HTTP ${response.status}`;
        }
        throw new Error(`User Settings Service Error: ${errorMessage}`);
      }

      return await response.json();
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error(`User Settings Service Request failed: ${String(error)}`);
    }
  }

  /**
   * Get current user settings
   * Creates default settings if none exist
   */
  async getSettings(): Promise<UserSettings> {
    return await this.fetchJSON<UserSettings>('/user/settings');
  }

  /**
   * Update conversation mode preference
   */
  async updateConversationMode(mode: ConversationMode): Promise<void> {
    await this.fetchJSON('/user/settings', {
      method: 'PATCH',
      body: JSON.stringify({ conversationMode: mode }),
    });
  }

  /**
   * Update multiple user settings at once
   */
  async updateSettings(settings: Partial<UserSettings>): Promise<void> {
    await this.fetchJSON('/user/settings', {
      method: 'PATCH',
      body: JSON.stringify(settings),
    });
  }
}

// Export singleton instance
export const userSettingsService = new UserSettingsService();
export default userSettingsService;
