package backend

import (
	"encoding/json"
	"strings"
)

// DevVersion é a versão de um binário sem versão conhecida (wails.json
// ilegível). O sufixo de pré-release faz o updater só checar, nunca instalar.
const DevVersion = "0.0.0-dev"

// ParseProductVersion extrai info.productVersion do wails.json embutido em
// main.go. No CI esse campo é sobrescrito com a versão da tag antes do build,
// então é a fonte única da versão: a mesma que o instalador NSIS e as
// propriedades do .exe mostram.
func ParseProductVersion(wailsJSON []byte) string {
	var cfg struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if err := json.Unmarshal(wailsJSON, &cfg); err != nil {
		return DevVersion
	}
	v := strings.TrimPrefix(strings.TrimSpace(cfg.Info.ProductVersion), "v")
	if v == "" {
		return DevVersion
	}
	return v
}

// MarkDevVersion acrescenta "-dev" a builds de `wails dev`, que leem o mesmo
// wails.json de uma versão publicada mas não são aquela versão. Assim o
// updater nunca troca um ambiente de desenvolvimento por um instalador.
func MarkDevVersion(version string, dev bool) string {
	if !dev || strings.Contains(version, "-") {
		return version
	}
	return version + "-dev"
}

// GetAppVersion devolve a versão do aplicativo em execução (ex.: "1.4.0").
func (a *App) GetAppVersion() string {
	if a.version == "" {
		return DevVersion
	}
	return a.version
}
