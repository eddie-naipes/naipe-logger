package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"logTime-go/backend/api"
	"logTime-go/backend/security"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Permissões dos arquivos de configuração: só o dono do perfil acessa. O token
// não fica nesses arquivos, mas eles revelam host, ID de usuário e tarefas.
const (
	dirPerm  os.FileMode = 0700
	filePerm os.FileMode = 0600
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

	// corruptedBackups guarda o caminho para onde foi movido cada arquivo de
	// configuração que não pôde ser decodificado na carga. O app segue com a
	// configuração padrão e a UI pode avisar o usuário.
	corruptedBackups []string
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
	legacyPurged, err := CheckAndMoveConfigFromExecDir()
	if err != nil {
		fmt.Printf("Aviso: não foi possível migrar configurações do diretório do executável: %v\n", err)
	}

	m, err := newManagerAt(configDir)
	if err != nil {
		return nil, err
	}
	if legacyPurged {
		m.mutex.Lock()
		m.legacyCredentialPurged = true
		m.mutex.Unlock()
	}
	return m, nil
}

func defaultAppConfig() *AppConfig {
	return &AppConfig{
		TeamworkConfig: api.Config{
			MinutosPorDia: 8 * 60,
		},
		SavedTasks: []api.Task{},
		AppSettings: AppSettings{
			Language: "pt-BR",
		},
	}
}

// newManagerAt permite apontar o gerenciador para um diretório arbitrário, o
// que torna o caminho de carga e migração testável sem tocar no HOME real.
func newManagerAt(configDir string) (*Manager, error) {
	if err := os.MkdirAll(configDir, dirPerm); err != nil {
		return nil, fmt.Errorf("erro ao criar diretório de configuração: %v", err)
	}

	m := &Manager{
		configFile:       filepath.Join(configDir, "config.json"),
		templatesFile:    filepath.Join(configDir, "templates.json"),
		appConfig:        defaultAppConfig(),
		templates:        make(map[string]api.Template),
		corruptedBackups: []string{},
	}

	// MkdirAll e as gravações não corrigem o que já existia: instalações
	// antigas criaram o diretório com 0755 e os arquivos podem ter sido
	// copiados com permissões abertas.
	tightenPermissions(configDir, dirPerm)
	tightenPermissions(m.configFile, filePerm)
	tightenPermissions(m.templatesFile, filePerm)

	if err := m.Load(); err != nil {
		return nil, err
	}

	return m, nil
}

// tightenPermissions aplica perm a um caminho existente. Falhar aqui não
// impede o app de abrir; no Windows o Chmod só mexe no atributo somente leitura.
func tightenPermissions(path string, perm os.FileMode) {
	if _, err := os.Stat(path); err != nil {
		return
	}
	if err := os.Chmod(path, perm); err != nil {
		fmt.Printf("Aviso: não foi possível ajustar permissões de %s: %v\n", path, err)
	}
}

// LegacyCredentialPurged informa se uma credencial no formato antigo foi
// encontrada e removida do disco, para que a UI possa orientar o usuário a
// gerar um token de API e trocar a senha comprometida.
func (m *Manager) LegacyCredentialPurged() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.legacyCredentialPurged
}

// CorruptedConfigBackups devolve os caminhos para onde foram movidos arquivos
// de configuração corrompidos encontrados na carga (vazio se não houve nenhum).
// Nunca devolve nil, para que o frontend receba [] e não null.
func (m *Manager) CorruptedConfigBackups() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	backups := make([]string, len(m.corruptedBackups))
	copy(backups, m.corruptedBackups)
	return backups
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
//
// O token é aparado aqui (além de em security.StoreToken) para que a cópia em
// memória seja idêntica à do cofre. Se gravar config.json falhar depois de o
// cofre já ter o token novo, o cofre volta ao token anterior e a memória não é
// alterada, para que cofre, disco e memória não divirjam.
func (m *Manager) SetConnection(host string, userID int, token string) error {
	normalizedHost, err := api.NormalizeHost(host)
	if err != nil {
		return err
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("token de API vazio")
	}

	m.mutex.RLock()
	previousToken := m.appConfig.TeamworkConfig.AuthToken
	m.mutex.RUnlock()

	// O cofre é acessado fora do lock: no Linux ele pode pedir o desbloqueio
	// do chaveiro e travaria todas as leituras de configuração enquanto isso.
	if err := storeToken(token); err != nil {
		return err
	}

	m.mutex.Lock()
	previous := m.appConfig.TeamworkConfig
	m.appConfig.TeamworkConfig.ApiHost = normalizedHost
	m.appConfig.TeamworkConfig.UserID = userID
	m.appConfig.TeamworkConfig.AuthToken = token

	if err := m.saveLocked(); err != nil {
		m.appConfig.TeamworkConfig = previous
		m.mutex.Unlock()

		if rollbackErr := restoreToken(previousToken); rollbackErr != nil {
			return fmt.Errorf("%v (e não foi possível restaurar o token anterior no cofre: %v)", err, rollbackErr)
		}
		return err
	}

	m.legacyCredentialPurged = false
	m.mutex.Unlock()
	return nil
}

// restoreToken devolve o cofre ao estado anterior a um SetConnection que falhou.
func restoreToken(previous string) error {
	if previous == "" {
		return deleteToken()
	}
	return storeToken(previous)
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

	if m.templates == nil {
		m.templates = make(map[string]api.Template)
	}
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

// Load lê config.json e templates.json. Um arquivo corrompido não impede o app
// de abrir: ele é renomeado para <nome>.corrompido-<data-hora>, a configuração
// padrão é usada no lugar e CorruptedConfigBackups passa a apontar o backup.
func (m *Manager) Load() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if data, err := os.ReadFile(m.configFile); err == nil {
		// Decodifica sobre os padrões: campos ausentes (ou um "null") não
		// zeram a jornada nem o idioma.
		loadedConfig := defaultAppConfig()
		if err := json.Unmarshal(data, loadedConfig); err != nil {
			m.quarantineLocked(m.configFile, err)
		} else {
			if loadedConfig.TeamworkConfig.MinutosPorDia <= 0 {
				loadedConfig.TeamworkConfig.MinutosPorDia = 8 * 60
			}
			if loadedConfig.SavedTasks == nil {
				loadedConfig.SavedTasks = []api.Task{}
			}
			*m.appConfig = *loadedConfig

			if hasLegacyCredential(data) {
				// A credencial antiga é o par email:senha, protegido por uma
				// chave derivável do código-fonte. Deve ser tratada como
				// comprometida: apagamos do disco e exigimos um token de API.
				m.legacyCredentialPurged = true
				if err := m.saveLocked(); err != nil {
					return fmt.Errorf("erro ao remover credencial antiga do disco: %v", err)
				}
				fmt.Println("Aviso: credencial antiga (email:senha) removida de config.json. Gere um token de API e troque sua senha do Teamwork.")
			}
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
		var loaded map[string]api.Template
		if err := json.Unmarshal(data, &loaded); err != nil {
			m.quarantineLocked(m.templatesFile, err)
		} else if loaded != nil {
			// Um templates.json contendo "null" decodifica para um mapa nil,
			// que faria SaveTemplate entrar em panic ao escrever nele.
			m.templates = loaded
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("erro ao ler arquivo de templates: %v", err)
	}
	if m.templates == nil {
		m.templates = make(map[string]api.Template)
	}

	return nil
}

// quarantineLocked tira do caminho um arquivo que não pôde ser decodificado,
// preservando-o para inspeção. Exige m.mutex travado para escrita.
func (m *Manager) quarantineLocked(path string, cause error) {
	backup := fmt.Sprintf("%s.corrompido-%s", path, time.Now().Format("20060102-150405"))
	if err := os.Rename(path, backup); err != nil {
		// Sem conseguir renomear, a próxima gravação substitui o arquivo
		// corrompido; ainda assim avisamos o usuário de onde ele estava.
		fmt.Printf("Aviso: %s está corrompido (%v) e não pôde ser renomeado: %v\n", path, cause, err)
		backup = path
	} else {
		fmt.Printf("Aviso: %s está corrompido (%v); movido para %s e substituído pela configuração padrão.\n", path, cause, backup)
	}
	m.corruptedBackups = append(m.corruptedBackups, backup)
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

	if err := writeFileAtomic(m.configFile, data, filePerm); err != nil {
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

	if err := writeFileAtomic(m.templatesFile, data, filePerm); err != nil {
		return fmt.Errorf("erro ao salvar templates: %v", err)
	}

	return nil
}

// writeFileAtomic grava data em path sem nunca deixar um arquivo pela metade:
// escreve num temporário do mesmo diretório, força para o disco e o renomeia
// por cima do destino. Uma queda de energia ou um crash no meio da escrita
// deixa o arquivo antigo intacto em vez de um JSON truncado.
func writeFileAtomic(path string, data []byte, perm os.FileMode) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	closed := false
	defer func() {
		if err != nil {
			if !closed {
				_ = tmp.Close()
			}
			_ = os.Remove(tmpName)
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	closed = true
	if err = tmp.Close(); err != nil {
		return err
	}
	// O rename leva junto as permissões do temporário, de modo que um destino
	// antigo com permissões abertas passa a ter perm.
	if err = os.Chmod(tmpName, perm); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// CheckAndMoveConfigFromExecDir migra config.json/templates.json deixados ao
// lado do executável (versões antigas) para ~/.teamwork-logger. Devolve true
// se uma credencial antiga (email:senha) foi apagada no processo.
func CheckAndMoveConfigFromExecDir() (bool, error) {
	execPath, err := os.Executable()
	if err != nil {
		return false, err
	}

	configDir, err := getConfigDir()
	if err != nil {
		return false, err
	}

	return migrateLegacyFiles(filepath.Dir(execPath), configDir)
}

// migrateLegacyFiles move os arquivos de configuração de execDir para
// configDir. Só migra quando o destino ainda não existe: se o original não
// puder ser removido, a próxima inicialização não sobrescreve a configuração
// atual com a cópia velha. Recebe os diretórios para ser testável.
func migrateLegacyFiles(execDir, configDir string) (legacyPurged bool, err error) {
	var errs []error

	for _, name := range []string{"config.json", "templates.json"} {
		src := filepath.Join(execDir, name)
		dst := filepath.Join(configDir, name)

		if _, statErr := os.Stat(src); statErr != nil {
			if !os.IsNotExist(statErr) {
				errs = append(errs, statErr)
			}
			continue
		}

		if sameFile(src, dst) {
			continue
		}

		data, readErr := os.ReadFile(src)
		if readErr != nil {
			errs = append(errs, readErr)
			continue
		}

		if _, statErr := os.Stat(dst); statErr == nil {
			// Já existe configuração no destino: a cópia antiga não é migrada.
			// Se ela ainda guarda a credencial email:senha, é apagada mesmo
			// assim — mantê-la em disco é o risco que o expurgo evita.
			if name == "config.json" && hasLegacyCredential(data) {
				if rmErr := os.Remove(src); rmErr != nil {
					fmt.Printf("Aviso: %s contém credencial antiga (email:senha) e não pôde ser apagado: %v\n", src, rmErr)
				} else {
					legacyPurged = true
					fmt.Printf("Aviso: %s com credencial antiga (email:senha) apagado. Gere um token de API e troque sua senha do Teamwork.\n", src)
				}
				continue
			}
			fmt.Printf("Aviso: %s não foi migrado porque %s já existe; o arquivo antigo foi mantido.\n", src, dst)
			continue
		} else if !os.IsNotExist(statErr) {
			errs = append(errs, statErr)
			continue
		}

		if mkErr := os.MkdirAll(configDir, dirPerm); mkErr != nil {
			errs = append(errs, mkErr)
			continue
		}
		if wErr := writeFileAtomic(dst, data, filePerm); wErr != nil {
			errs = append(errs, wErr)
			continue
		}

		if rmErr := os.Remove(src); rmErr != nil {
			// O destino já existe, então isto não se repete a cada abertura.
			fmt.Printf("Aviso: %s foi migrado para %s, mas o original não pôde ser removido: %v\n", src, dst, rmErr)
		}
	}

	return legacyPurged, errors.Join(errs...)
}

// sameFile evita que a migração apague o próprio destino quando o executável
// roda de dentro do diretório de configuração.
func sameFile(a, b string) bool {
	infoA, errA := os.Stat(a)
	infoB, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(infoA, infoB)
}
