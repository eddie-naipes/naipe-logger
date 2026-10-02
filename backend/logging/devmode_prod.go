//go:build !dev

package logging

// Build de produção: o app não tem terminal, o log vai só para o arquivo.
const devBuild = false
