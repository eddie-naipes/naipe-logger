package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"logTime-go/backend/api"
	"logTime-go/backend/security"
	"os"
	"path/filepath"
	"sync"
)

// Indireção sobre o cofre de credenciais para que os testes não dependam de um
// Keychain/Secret Service real — ambientes de CI headless não têm um.
var (
	storeToken  = security.StoreToken
	loadToken   = security.LoadToken
	deleteToken = security.DeleteToken
)

type Manager struct {
	configFile    string
	templatesFile string
	appConfig     *AppConfig
	templates     map[string]api.Template
	mutex         sync.RWMutex

	// legacyCredentialPurged registra que o config.json continha uma credencial
	// no formato antigo (email:senha, com criptografia derivável do código) e
	// que ela foi apagada do disco na inicialização.
	legacyCredentialPurged bool
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

	// Precisa vir antes do Load para que um config.json legado deixado no
	// diretório do executável também passe pelo expurgo da credencial antiga.
	if err := CheckAndMoveConfigFromExecDir(); err != nil {
		fmt.Printf("Aviso: não foi possível migrar configurações do diretório do executável: %v\n", err)
	}

	return newManagerAt(configDir)
}

// newManagerAt permite apontar o gerenciador para um diretório arbitrário, o
// que torna o caminho de carga e migração testável sem tocar no HOME real.
func newManagerAt(configDir string) (*Manager, error) {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("erro ao criar diretório de configuração: %v", err)
	}

	m := &Manager{
		configFile:    filepath.Join(configDir, "config.json"),
		templatesFile: filepath.Join(configDir, "templates.json"),
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

	if err := m.Load(); err != nil {
		return nil, err
	}

	return m, nil
}

// LegacyCredentialPurged informa se uma credencial no formato antigo foi
// encontrada e removida do disco, para que a UI possa orientar o usuário a
// gerar um token de API e trocar a senha comprometida.
func (m *Manager) LegacyCredentialPurged() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.legacyCredentialPurged
}

func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("erro ao obter diretório do usuário: %v", err)
	}

	return filepath.Join(homeDir, ".teamwork-logger"), nil
}

func (m *Manager) GetTeamworkConfig() api.Config {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.appConfig.TeamworkConfig
}

// SetConnection grava o token no cofre do sistema e persiste host e usuário em
// config.json. O host é normalizado (https obrigatório) antes de ser aceito.
func (m *Manager) SetConnection(host string, userID int, token string) error {
	normalizedHost, err := api.NormalizeHost(host)
	if err != nil {
		return err
	}

	if err := storeToken(token); err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.appConfig.TeamworkConfig.ApiHost = normalizedHost
	m.appConfig.TeamworkConfig.UserID = userID
	m.appConfig.TeamworkConfig.AuthToken = token
	m.legacyCredentialPurged = false

	return m.saveLocked()
}

// ClearConnection remove o token do cofre e limpa a configuração de conexão.
func (m *Manager) ClearConnection() error {
	if err := deleteToken(); err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.appConfig.TeamworkConfig.AuthToken = ""
	m.appConfig.TeamworkConfig.ApiHost = ""
	m.appConfig.TeamworkConfig.UserID = 0
	m.legacyCredentialPurged = false

	return m.saveLocked()
}

// SetMinutosPorDia ajusta a jornada diária usada nos cálculos.
func (m *Manager) SetMinutosPorDia(minutos int) error {
	if minutos <= 0 {
		return fmt.Errorf("jornada diária inválida: %d minutos", minutos)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.appConfig.TeamworkConfig.MinutosPorDia = minutos
	return m.saveLocked()
}

// GetSavedTasks devolve uma cópia: o slice interno não pode escapar do lock,
// senão o Wails o serializa enquanto outra chamada o modifica.
func (m *Manager) GetSavedTasks() []api.Task {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	tasks := make([]api.Task, len(m.appConfig.SavedTasks))
	copy(tasks, m.appConfig.SavedTasks)
	return tasks
}

func (m *Manager) SetSavedTasks(tasks []api.Task) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.appConfig.SavedTasks = tasks
	return m.saveLocked()
}

func (m *Manager) AddSavedTask(task api.Task) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i, t := range m.appConfig.SavedTasks {
		if t.TaskID == task.TaskID {
			m.appConfig.SavedTasks[i] = task
			return m.saveLocked()
		}
	}

	m.appConfig.SavedTasks = append(m.appConfig.SavedTasks, task)
	return m.saveLocked()
}

func (m *Manager) RemoveSavedTask(taskID int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i, task := range m.appConfig.SavedTasks {
		if task.TaskID == taskID {
			m.appConfig.SavedTasks = append(m.appConfig.SavedTasks[:i], m.appConfig.SavedTasks[i+1:]...)
			return m.saveLocked()
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
	defer m.mutex.Unlock()

	m.appConfig.AppSettings = settings
	return m.saveLocked()
}

// GetTemplates devolve uma cópia do mapa: devolver o mapa interno permitiria
// que o Wails o lesse enquanto SaveTemplate escreve, causando panic de
// "concurrent map read and map write".
func (m *Manager) GetTemplates() map[string]api.Template {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	templates := make(map[string]api.Template, len(m.templates))
	for name, template := range m.templates {
		templates[name] = template
	}
	return templates
}

func (m *Manager) GetTemplate(name string) (api.Template, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	template, exists := m.templates[name]
	return template, exists
}

func (m *Manager) SaveTemplate(template api.Template) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.templates[template.Name] = template
	return m.saveTemplatesLocked()
}

func (m *Manager) DeleteTemplate(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.templates, name)
	return m.saveTemplatesLocked()
}

// hasLegacyCredential detecta o campo "authToken" gravado pelas versões antigas
// em config.json. api.Config já não serializa esse campo, então ele precisa ser
// procurado à parte para poder ser expurgado.
func hasLegacyCredential(data []byte) bool {
	var probe struct {
		TeamworkConfig struct {
			AuthToken string `json:"authToken"`
		} `json:"teamworkConfig"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return false
	}
	return probe.TeamworkConfig.AuthToken != ""
}

func (m *Manager) Load() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if data, err := os.ReadFile(m.configFile); err == nil {
		var loadedConfig AppConfig
		if err := json.Unmarshal(data, &loadedConfig); err != nil {
			return fmt.Errorf("erro ao decodificar configurações: %v", err)
		}
		if loadedConfig.TeamworkConfig.MinutosPorDia == 0 {
			loadedConfig.TeamworkConfig.MinutosPorDia = 8 * 60
		}
		*m.appConfig = loadedConfig

		if hasLegacyCredential(data) {
			// A credencial antiga é o par email:senha, protegido por uma chave
			// derivável do código-fonte. Deve ser tratada como comprometida:
			// apagamos do disco e exigimos um token de API no lugar.
			m.legacyCredentialPurged = true
			if err := m.saveLocked(); err != nil {
				return fmt.Errorf("erro ao remover credencial antiga do disco: %v", err)
			}
			fmt.Println("Aviso: credencial antiga (email:senha) removida de config.json. Gere um token de API e troque sua senha do Teamwork.")
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("erro ao ler arquivo de configuração: %v", err)
	}

	// O segredo vive no cofre do sistema, nunca em config.json.
	token, err := loadToken()
	switch {
	case err == nil:
		m.appConfig.TeamworkConfig.AuthToken = token
	case errors.Is(err, security.ErrNoToken):
		// Ainda não configurado: o usuário será levado à tela de configuração.
	default:
		fmt.Printf("Aviso: %v\n", err)
	}

	if data, err := os.ReadFile(m.templatesFile); err == nil {
		if err := json.Unmarshal(data, &m.templates); err != nil {
			return fmt.Errorf("erro ao decodificar templates: %v", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("erro ao ler arquivo de templates: %v", err)
	}

	return nil
}

// Save grava config.json. O token nunca é incluído: api.Config o marca como
// `json:"-"` e ele reside apenas no cofre de credenciais do sistema.
func (m *Manager) Save() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.saveLocked()
}

// saveLocked exige que m.mutex já esteja travado para escrita.
func (m *Manager) saveLocked() error {
	data, err := json.MarshalIndent(m.appConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar configurações: %v", err)
	}

	if err := os.WriteFile(m.configFile, data, 0600); err != nil {
		return fmt.Errorf("erro ao salvar configurações: %v", err)
	}

	return nil
}

func (m *Manager) SaveTemplates() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.saveTemplatesLocked()
}

// saveTemplatesLocked exige que m.mutex já esteja travado para escrita.
func (m *Manager) saveTemplatesLocked() error {
	data, err := json.MarshalIndent(m.templates, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar templates: %v", err)
	}

	if err := os.WriteFile(m.templatesFile, data, 0600); err != nil {
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
		if err := os.WriteFile(newConfigPath, data, 0600); err != nil {
			return err
		}

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
		if err := os.WriteFile(newTemplatesPath, data, 0600); err != nil {
			return err
		}

		os.Remove(oldTemplatesPath)
	}

	return nil
}
