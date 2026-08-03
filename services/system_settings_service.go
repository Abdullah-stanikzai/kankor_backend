package services

import (
	"context"
	"fmt"
	"kankor-backend/config"
	"kankor-backend/models"
)

// SystemSettingsService handles system settings-related business logic
type SystemSettingsService struct {
	config *config.Config
}

// NewSystemSettingsService creates a new SystemSettingsService instance
func NewSystemSettingsService(cfg *config.Config) *SystemSettingsService {
	return &SystemSettingsService{
		config: cfg,
	}
}

// GetSystemSettings gets all system settings
func (sss *SystemSettingsService) GetSystemSettings() ([]*models.SystemSetting, error) {
	rows, err := config.DBConnection.Query(context.Background(), 
		`SELECT id, setting_key, setting_value, description, updated_at FROM system_settings ORDER BY setting_key`)
	if err != nil {
		return nil, fmt.Errorf("failed to query system settings: %w", err)
	}
	defer rows.Close()

	var settings []*models.SystemSetting
	for rows.Next() {
		var setting models.SystemSetting
		err := rows.Scan(&setting.ID, &setting.SettingKey, &setting.SettingValue, &setting.Description, &setting.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan system setting: %w", err)
		}
		settings = append(settings, &setting)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating system setting rows: %w", err)
	}

	// If no settings exist in database, return default settings
	if len(settings) == 0 {
		settings = []*models.SystemSetting{
			{
				ID:          "1",
				SettingKey:  "app_name",
				SettingValue: "Kankor Exam Platform",
				Description: "Name of the application",
			},
			{
				ID:          "2",
				SettingKey:  "default_language",
				SettingValue: "fa",
				Description: "Default language code (fa=Persian/Dari, ps=Pashto)",
			},
			{
				ID:          "3",
				SettingKey:  "max_login_attempts",
				SettingValue: "5",
				Description: "Maximum login attempts before temporary lockout",
			},
			{
				ID:          "4",
				SettingKey:  "otp_expiry_minutes",
				SettingValue: "10",
				Description: "Number of minutes after which OTP expires",
			},
			{
				ID:          "5",
				SettingKey:  "exam_auto_submit_buffer",
				SettingValue: "30",
				Description: "Additional seconds to allow submission after exam ends",
			},
			{
				ID:          "6",
				SettingKey:  "min_password_length",
				SettingValue: "8",
				Description: "Minimum password length requirement",
			},
		}
	}

	return settings, nil
}

// UpdateSystemSettings updates system settings
func (sss *SystemSettingsService) UpdateSystemSettings(updates map[string]string) ([]*models.SystemSetting, error) {
	// Validate and update settings in the database
	for key, value := range updates {
		// Update the specific setting in the database
		_, err := config.DBConnection.Exec(context.Background(),
			`INSERT INTO system_settings (setting_key, setting_value, updated_at) 
			VALUES ($1, $2, NOW()) 
			ON CONFLICT (setting_key) DO UPDATE SET 
				setting_value = EXCLUDED.setting_value, 
				updated_at = EXCLUDED.updated_at`,
			key, value)
		if err != nil {
			return nil, fmt.Errorf("failed to update setting %s: %w", key, err)
		}
	}

	// Return all updated settings
	return sss.GetSystemSettings()
}