// Gera em .wails-types/ uma cópia, só para o TypeScript, dos tipos que o Wails
// gera em wailsjs/go/ (models.ts e backend/App.d.ts).
//
// Por que existe: o gerador do Wails v2 transforma structs anônimas do Go
// (ex.: os campos Tags e Assignees de api.TeamworkTask) em `export class  {`
// sem nome e em chamadas `convertValues(x, )` com um argumento a menos. Isso é
// TypeScript inválido, e como App.d.ts importa ../models, o `tsc` acusa esses
// erros mesmo com skipLibCheck. wailsjs/ é regenerado pelo Wails e não pode ser
// editado à mão, então a correção é feita numa cópia:
//
//   - classes sem nome são removidas (nada consegue referenciá-las mesmo);
//   - `this.convertValues(source["x"], )` vira `source["x"]`.
//
// O resto é copiado sem alteração, então os tipos acompanham cada nova geração
// dos bindings. Em tempo de execução o app continua usando wailsjs/ (alias
// @wailsjs no vite.config.ts); esta cópia só é lida pelo tsc e pelo ESLint.
// Quando as structs do Go ganharem nome, o script vira uma cópia simples.
import {mkdirSync, readFileSync, writeFileSync} from 'node:fs';
import {dirname, join} from 'node:path';
import {fileURLToPath} from 'node:url';

const raiz = join(dirname(fileURLToPath(import.meta.url)), '..');
const origem = join(raiz, 'wailsjs', 'go');
const destino = join(raiz, '.wails-types', 'go');

const AVISO = '// Arquivo gerado por scripts/wails-types.mjs a partir de wailsjs/go. Não edite.\n';

const sanitizarModels = (texto) => {
    const linhas = texto.split(/\r?\n/);
    const saida = [];
    let classesRemovidas = 0;
    let dentroDeClasseSemNome = false;

    for (const linha of linhas) {
        if (dentroDeClasseSemNome) {
            // A classe termina no `}` com a mesma indentação (um tab) da abertura.
            if (/^\t\}\s*$/.test(linha)) dentroDeClasseSemNome = false;
            continue;
        }
        if (/^\s*export class\s*\{\s*$/.test(linha)) {
            dentroDeClasseSemNome = true;
            classesRemovidas++;
            continue;
        }
        saida.push(linha);
    }

    let chamadasCorrigidas = 0;
    const corrigido = saida.join('\n').replace(
        /this\.convertValues\((source\["[^"]+"\]),\s*\)/g,
        (_, campo) => {
            chamadasCorrigidas++;
            return campo;
        }
    );

    return {texto: corrigido, classesRemovidas, chamadasCorrigidas};
};

const models = sanitizarModels(readFileSync(join(origem, 'models.ts'), 'utf8'));
const app = readFileSync(join(origem, 'backend', 'App.d.ts'), 'utf8');

mkdirSync(join(destino, 'backend'), {recursive: true});
writeFileSync(join(destino, 'models.ts'), AVISO + models.texto);
writeFileSync(join(destino, 'backend', 'App.d.ts'), AVISO + app);

console.log(
    `wails-types: ${models.classesRemovidas} classe(s) sem nome removida(s), `
    + `${models.chamadasCorrigidas} convertValues corrigido(s).`
);
