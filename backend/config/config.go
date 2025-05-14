package config

import (
	"encoding/json"
	"fmt"
	"logTime-go/backend/api"
	"logTime-go/backend/security"
	"os"
	"path/filepath"
	"sync"
)

type Manager struct {
	configFile    string
	templatesFile string
	appConfig     *AppConfig
	templates     map[string]api.Template
	mutex         sync.RWMutex
}

type AppConfig struct {
	TeamworkConfig api.Config  `json:"teamworkConfig"`
	SavedTasks     []api.Task  `json:"savedTasks"`
	AppSettings    AppSettings `json:"appSettings"`
}

type AppSettings struct {
	DarkMode       bool   `json:"darkMode"`
	AutoUpdate     bool   `json:"autoUpdate"`
	StartMinimized bool   `json:"startMinimized"`
	Language       string `json:"language"`
}

func NewManager() (*Manager, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("erro ao criar diretório de configuração: %v", err)
	}

	configFile := filepath.Join(configDir, "config.json")
	templatesFile := filepath.Join(configDir, "templates.json")

	m := &Manager{
		configFile:    configFile,
		templatesFile: templatesFile,
		appConfig: &AppConfig{
			TeamworkConfig: api.Config{
				MinutosPorDia: 8 * 60,
			},
			SavedTasks: []api.Task{},
			AppSettings: AppSettings{
				Language: "pt-BR",
			},
		},
		templates: make(map[string]api.Template),
	}

	// Carregar configurações existentes
	m.Load()

	// Tentar migrar dados inseguros para formato seguro, com tratamento de erros
	if err := m.MigrateToSecureStorage(); err != nil {
		fmt.Printf("Aviso: Erro ao migrar para armazenamento seguro: %v\n", err)
	}

	return m, nil
}

func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("erro ao obter diretório do usuário: %v", err)
	}

	// Usar o mesmo diretório para manter compatibilidade com versões anteriores
	return filepath.Join(homeDir, ".teamwork-logger"), nil
}

func (m *Manager) GetTeamworkConfig() api.Config {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.appConfig.TeamworkConfig
}

func (m *Manager) SetTeamworkConfig(config api.Config) error {
	m.mutex.Lock()
	m.appConfig.TeamworkConfig = config
	m.mutex.Unlock()
	return m.Save()
}

func (m *Manager) GetSavedTasks() []api.Task {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.appConfig.SavedTasks
}

func (m *Manager) SetSavedTasks(tasks []api.Task) error {
	m.mutex.Lock()
	m.appConfig.SavedTasks = tasks
	m.mutex.Unlock()
	return m.Save()
}

func (m *Manager) AddSavedTask(task api.Task) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i, t := range m.appConfig.SavedTasks {
		if t.TaskID == task.TaskID {
			m.appConfig.SavedTasks[i] = task
			return m.Save()
		}
	}

	m.appConfig.SavedTasks = append(m.appConfig.SavedTasks, task)
	return m.Save()
}

func (m *Manager) RemoveSavedTask(taskID int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i, task := range m.appConfig.SavedTasks {
		if task.TaskID == taskID {
			m.appConfig.SavedTasks = append(m.appConfig.SavedTasks[:i], m.appConfig.SavedTasks[i+1:]...)
			return m.Save()
		}
	}

	return fmt.Errorf("tarefa não encontrada: %d", taskID)
}

func (m *Manager) GetAppSettings() AppSettings {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.appConfig.AppSettings
}

func (m *Manager) SetAppSettings(settings AppSettings) error {
	m.mutex.Lock()
	m.appConfig.AppSettings = settings
	m.mutex.Unlock()
	return m.Save()
}

func (m *Manager) GetTemplates() map[string]api.Template {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.templates
}

func (m *Manager) GetTemplate(name string) (api.Template, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	template, exists := m.templates[name]
	return template, exists
}

func (m *Manager) SaveTemplate(template api.Template) error {
	m.mutex.Lock()
	m.templates[template.Name] = template
	m.mutex.Unlock()
	return m.SaveTemplates()
}

func (m *Manager) DeleteTemplate(name string) error {
	m.mutex.Lock()
	delete(m.templates, name)
	m.mutex.Unlock()
	return m.SaveTemplates()
}

func (m *Manager) MigrateToSecureStorage() error {
	// Se não há token, não há nada para migrar
	if m.appConfig.TeamworkConfig.AuthToken == "" {
		return nil
	}

	// Verificar se o token já parece estar criptografado
	// Tokens criptografados em base64 tendem a ser longos
	if len(m.appConfig.TeamworkConfig.AuthToken) > 100 {
		// Tenta descriptografar para verificar
		_, err := security.Decrypt(m.appConfig.TeamworkConfig.AuthToken)
		if err == nil {
			// Parece que já está criptografado e é válido
			return nil
		}
	}

	// O token não está criptografado ou não é válido, então vamos criptografá-lo agora
	return m.Save()
}

func (m *Manager) Load() error {
	if _, err := os.Stat(m.configFile); err == nil {
		data, err := os.ReadFile(m.configFile)
		if err != nil {
			return fmt.Errorf("erro ao ler arquivo de configuração: %v", err)
		}

		var loadedConfig AppConfig
		if err := json.Unmarshal(data, &loadedConfig); err != nil {
			return fmt.Errorf("erro ao decodificar configurações: %v", err)
		}

		// Tenta descriptografar o token se existir
		if loadedConfig.TeamworkConfig.AuthToken != "" {
			decryptedToken, err := security.Decrypt(loadedConfig.TeamworkConfig.AuthToken)
			if err != nil {
				// Se falhar ao descriptografar, mantém o valor original
				// Isso permite compatibilidade com versões anteriores
				fmt.Printf("Aviso: não foi possível descriptografar o token. Assumindo que está em texto simples.\n")
			} else {
				loadedConfig.TeamworkConfig.AuthToken = decryptedToken
			}
		}

		*m.appConfig = loadedConfig
	}

	if _, err := os.Stat(m.templatesFile); err == nil {
		data, err := os.ReadFile(m.templatesFile)
		if err != nil {
			return fmt.Errorf("erro ao ler arquivo de templates: %v", err)
		}

		if err := json.Unmarshal(data, &m.templates); err != nil {
			return fmt.Errorf("erro ao decodificar templates: %v", err)
		}
	}

	return nil
}

func (m *Manager) Save() error {
	// Criar uma cópia da configuração para não alterar o objeto original
	configToSave := *m.appConfig

	// Criptografar o token apenas se existir
	if configToSave.TeamworkConfig.AuthToken != "" {
		// Primeiro, verificar se já não está criptografado
		if len(configToSave.TeamworkConfig.AuthToken) < 100 { // Tokens criptografados tendem a ser longos
			encryptedToken, err := security.Encrypt(configToSave.TeamworkConfig.AuthToken)
			if err != nil {
				return fmt.Errorf("erro ao criptografar token: %v", err)
			}
			configToSave.TeamworkConfig.AuthToken = encryptedToken
		}
	}

	// Serializar a configuração
	data, err := json.MarshalIndent(configToSave, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar configurações: %v", err)
	}

	// Salvar no arquivo
	if err := os.WriteFile(m.configFile, data, 0644); err != nil {
		return fmt.Errorf("erro ao salvar configurações: %v", err)
	}

	return nil
}

func (m *Manager) SaveTemplates() error {
	data, err := json.MarshalIndent(m.templates, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar templates: %v", err)
	}

	if err := os.WriteFile(m.templatesFile, data, 0644); err != nil {
		return fmt.Errorf("erro ao salvar templates: %v", err)
	}

	return nil
}

func CheckAndMoveConfigFromExecDir() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	execDir := filepath.Dir(execPath)
	oldConfigPath := filepath.Join(execDir, "config.json")
	oldTemplatesPath := filepath.Join(execDir, "templates.json")

	if _, err := os.Stat(oldConfigPath); err == nil {
		configDir, err := getConfigDir()
		if err != nil {
			return err
		}

		if err := os.MkdirAll(configDir, 0755); err != nil {
			return err
		}

		data, err := os.ReadFile(oldConfigPath)
		if err != nil {
			return err
		}

		newConfigPath := filepath.Join(configDir, "config.json")
		if err := os.WriteFile(newConfigPath, data, 0644); err != nil {
			return err
		}

		// Tenta remover, mas não trata como erro se falhar
		os.Remove(oldConfigPath)
	}

	if _, err := os.Stat(oldTemplatesPath); err == nil {
		configDir, err := getConfigDir()
		if err != nil {
			return err
		}

		data, err := os.ReadFile(oldTemplatesPath)
		if err != nil {
			return err
		}

		newTemplatesPath := filepath.Join(configDir, "templates.json")
		if err := os.WriteFile(newTemplatesPath, data, 0644); err != nil {
			return err
		}

		os.Remove(oldTemplatesPath)
	}

	return nil
}
