# Teamwork Time Logger

![Teamwork Time Logger](build/windows/icon.ico)

## 📝 Sobre o Projeto

O **Teamwork Time Logger** é uma aplicação desktop (Wails: Go + React) para lançar horas na plataforma Teamwork em lote, a partir de tarefas salvas e templates reutilizáveis, evitando o preenchimento manual dia a dia na interface web.

### Principais funcionalidades

- **Lançamento em lote**: distribui entradas de tempo por um intervalo de datas, pulando fins de semana e feriados
- **Tarefas salvas**: configure múltiplas entradas por tarefa, com horário, descrição e dias da semana aplicáveis
- **Templates**: salve conjuntos de tarefas e aplique-os ao módulo de lançamento
- **Calendário mensal**: visualize as horas já lançadas e os dias não úteis
- **Gerenciador de apontamentos**: liste, edite e exclua entradas de tempo de um período
- **Feriados brasileiros**: obtidos da BrasilAPI, com fallback local (inclui feriados móveis via algoritmo de Gauss)
- **Relatórios em PDF**: exportação do relatório de horas do Teamwork por período
- **Tema claro/escuro**

## 🔒 Segurança

O modelo de credenciais é deliberadamente simples:

- **Autenticação por token de API**, não por senha. O aplicativo nunca pede, transmite ou armazena a senha da sua conta Teamwork.
- **O token fica no cofre de credenciais do sistema operacional** — Credential Manager (Windows), Keychain (macOS), Secret Service (Linux). Nunca é gravado em `config.json`.
- **O token não atravessa para o frontend.** O processo Go monta o cabeçalho de autenticação; o webview recebe apenas dados já autenticados e um booleano de "configurado".
- **Somente HTTPS.** A autenticação é Basic — o token viaja em base64 em toda requisição. Endereços `http://` são recusados, e há uma verificação final antes de cada requisição sair.
- **Arquivos locais com permissão `0600`.**

### Migração de versões anteriores

Versões até a 1.0 guardavam `email:senha` em `config.json`, cifrado com AES-GCM cuja chave era derivada de uma constante presente no código-fonte — ou seja, reversível por qualquer pessoa com acesso ao arquivo.

Ao iniciar, o aplicativo **detecta e apaga** essa credencial, e exibe um aviso na tela de configuração. **Se você usou uma versão anterior, troque sua senha do Teamwork**: ela deve ser considerada exposta.

### Como obter um token de API

No Teamwork, acesse seu perfil → *Edit My Details* → aba *API & Mobile*. O caminho exato pode variar conforme a versão da sua instância.

## 🚀 Tecnologias

### Backend (Go 1.24)
- **Wails v2.10.1** — aplicação desktop híbrida
- **go-keyring** — cofre de credenciais do SO
- **HTTP client** com connection pooling, timeouts e repetição com backoff exponencial
- **Cache em memória** com TTL por tipo de dado
- **Goroutines com semáforo** para lançamentos concorrentes (limite de 3 simultâneos)

### Frontend (React 18)
- **React Router**, **TailwindCSS**, **React Icons (Feather)**
- **date-fns** com locale pt-BR
- **React Toastify**, **clsx**
- **Vite 7**

## 🛠️ Estrutura do Projeto

```
teamwork-logger/
├── backend/
│   ├── api/                # Integração com a API do Teamwork
│   │   ├── auth.go         # Validação de token e identificação do usuário
│   │   ├── client.go       # HTTP client, autenticação e barreira de HTTPS
│   │   ├── host.go         # Normalização e validação do host
│   │   ├── cache.go        # Cache com TTL
│   │   ├── conflicts.go    # Detecção de lançamentos duplicados
│   │   ├── dates.go        # Parsing das datas devolvidas pela API
│   │   ├── tasks.go        # Tarefas
│   │   ├── projects.go     # Projetos
│   │   ├── reports.go      # Relatórios e exportação PDF
│   │   ├── retry.go        # Política de repetição (rate limit e falhas de rede)
│   │   ├── time_entries.go # CRUD de apontamentos e distribuição
│   │   ├── Holiday.go      # Feriados (BrasilAPI + fallback)
│   │   └── types.go
│   ├── config/
│   │   └── config.go       # Persistência de config, tarefas e templates
│   ├── security/
│   │   └── credentials.go  # Token no cofre do SO
│   └── app.go              # Bindings expostos ao frontend
├── frontend/
│   ├── src/
│   │   ├── components/     # Sidebar, Header, MonthlyTimeCalendar,
│   │   │                   # TimeEntryManager, HolidayManager, UserProfile,
│   │   │                   # TimeInputComponent
│   │   ├── pages/          # Dashboard, Config, Task, TimeLog, Templates,
│   │   │                   # ReportPeriodModal, NotFound
│   │   └── contexts/       # ThemeContext
│   └── index.html
└── main.go
```

## 🖥️ Funcionalidades

### 📊 Dashboard

![Dashboard](frontend/src/assets/dashboard.png)

- Horas lançadas no mês, com comparação percentual ao mês anterior
- Contagem de dias úteis do mês (decorridos e restantes)
- Tarefas pendentes e projetos ativos
- **Atividades recentes**: seus últimos 5 lançamentos de tempo dos últimos 30 dias
- **Próximos prazos**: as 5 tarefas atribuídas a você com vencimento mais próximo. Tarefas sem prazo definido no Teamwork não aparecem
- Calendário visual do mês
- Exportação do relatório PDF do mês corrente

Se um desses cards falhar ao carregar, ele fica vazio e o restante do dashboard continua funcionando.

### ⏰ Lançamento de Horas

![Lançamento de Horas](frontend/src/assets/hours.png)

Fluxo:

1. **Selecione as tarefas** salvas (ou aplique um template)
2. **Defina o período** por data inicial/final, ou clicando num dia do calendário
3. **Gere o plano** — o backend expande as tarefas pelos dias úteis do intervalo, respeitando os `workingDays` de cada tarefa
4. **Revise** o plano completo antes de enviar
5. **Execute** — os lançamentos são enviados em paralelo (3 simultâneos, com pausa entre eles)
6. **Confira** o resultado individual de cada entrada

Fins de semana e feriados são excluídos automaticamente do plano.

**Detecção de duplicatas:** ao gerar o plano, o aplicativo consulta os lançamentos já existentes no período e destaca os dias que já possuem tempo registrado, indicando quais colidem com a *mesma tarefa* do plano. Se houver colisão, o botão de envio muda de cor e exige confirmação explícita. Se a verificação em si falhar, o envio também pede confirmação — em vez de seguir em silêncio.

**Desfazer lançamento:** após executar, o painel de resultados oferece um botão que apaga do Teamwork as entradas criadas por aquele lote. Útil quando parte das entradas falha, ou quando o plano estava errado.

**Repetição automática:** um lote grande costuma esbarrar no rate limit da API. Requisições recusadas com `429` são repetidas até 3 vezes, com backoff exponencial (500 ms, 1 s, 2 s…, teto de 8 s), respeitando o cabeçalho `Retry-After` quando o servidor o envia. Falhas de rede e erros `5xx` só são repetidos em métodos idempotentes — **um `POST` que falha nunca é reenviado**, porque não há como saber se o lançamento chegou a ser criado, e repetir duplicaria horas. Nesses casos a entrada aparece como falha no painel de resultados e pode ser reenviada manualmente.

Fechar o aplicativo cancela o que estiver em voo: as requisições carregam o contexto da aplicação, e a espera do backoff é interrompida em vez de segurar o encerramento por até 8 segundos.

> **Atenção:** o desfazer depende do identificador que o Teamwork devolve ao criar cada entrada. Se algum lançamento vier sem esse identificador, o painel informa quantos ficaram de fora — esses precisam ser removidos pelo Gerenciador de Apontamentos.
>
> O aviso de duplicata não bloqueia o envio: lançamentos são **somados**, não substituídos.

### 📋 Gerenciamento de Tarefas

![Gerenciamento de Tarefas](frontend/src/assets/manager-task.png)

- Importação de projetos e tarefas do Teamwork
- Busca e filtro por projeto
- Múltiplas entradas por tarefa (horário de início, duração, descrição, billable)
- Seleção dos dias da semana em que cada tarefa se aplica
- Cálculo do total de minutos configurado

### 🎯 Templates

![Templates](frontend/src/assets/templates.png)

- Salve o conjunto atual de tarefas como um template nomeado
- Aplique um template para carregá-lo no módulo de lançamento
- Exclua templates

Templates são salvos em `templates.json`. Não há versionamento nem exportação/importação entre máquinas.

### 🔧 Configuração

![Configurações](frontend/src/assets/config.png)

- Domínio da empresa e token de API
- Validação do token contra a API antes de salvar
- Logout, que remove o token do cofre do sistema

### 🗂️ Gerenciador de Apontamentos

- Listagem das entradas de tempo por período
- Filtros por projeto, tarefa, faixa de horas e billable
- Edição de uma entrada (duração, horário, descrição, billable)
- **Exclusão em lote**: selecione as entradas pela caixa de marcação (ou "Selecionar todas") e apague de uma vez, com um painel mostrando o resultado de cada uma

A exclusão em lote roda no backend (`DeleteMultipleTimeEntries`), com 3 exclusões simultâneas, respiro entre elas e a mesma política de repetição em rate limit das demais chamadas. Entradas já deletadas não são selecionáveis.

### 📅 Calendário Mensal

- Horas lançadas por dia
- Marcação de fins de semana e feriados
- Clique num dia para carregá-lo no módulo de lançamento

## 💾 Armazenamento Local

```
~/.teamwork-logger/
├── config.json      # host, userId, jornada diária, tarefas salvas, preferências (0600)
└── templates.json   # templates de trabalho (0600)
```

O token de API **não** fica nesses arquivos — ele reside no cofre de credenciais do sistema operacional.

O cache (projetos, tarefas, feriados, estatísticas) é mantido apenas em memória e se perde ao fechar o aplicativo.

## 🔧 Desenvolvimento

### Requisitos
- Go 1.24+
- Node.js 20+
- [Wails CLI v2.10.1](https://wails.io/docs/gettingstarted/installation)
- Linux: `libgtk-3-dev`, `libwebkit2gtk-4.0-dev` (ou 4.1)

### Rodando

```bash
git clone <repo>
cd teamwork-logger

go mod download
cd frontend && npm ci && cd ..

wails dev
```

### Testes e verificações

```bash
gofmt -l backend/ main.go   # deve não listar nada
go vet ./...
go test ./...
go test -race ./...         # requer CGO_ENABLED=1 e um compilador C

cd frontend && npm audit --audit-level=high
```

### Build de produção

```bash
wails build

# Executável gerado em:
# Windows: ./build/bin/teamwork-logger.exe
# macOS:   ./build/bin/teamwork-logger.app
# Linux:   ./build/bin/teamwork-logger
```

O CI (`.github/workflows/build.yml`) roda as verificações antes de compilar para as três plataformas e, em tags `v*`, publica uma release com checksums SHA-256.

> O instalador Windows **não é assinado digitalmente**. O SmartScreen exibirá um aviso na primeira execução.

## 🗺️ Limitações conhecidas

- O desfazer de lote depende do ID devolvido pela API; entradas sem ID precisam ser removidas manualmente
- A leitura de apontamentos de um período pagina até 50 páginas (25 mil entradas). O teto existe para evitar laço infinito caso a API devolva `hasMore` indefinidamente; períodos reais ficam muito abaixo disso
- Um `POST` que falha por rede ou erro do servidor não é reenviado automaticamente (evita duplicar horas) — só o rate limit `429` dispara repetição
- Sem auto-update
- Sem modo offline — toda operação requer conexão
- Sem backup automático das configurações
- Interface disponível apenas em português

## 📜 Licença

MIT.

---

*Ferramenta interna para reduzir o trabalho manual de lançamento de horas no Teamwork.*
