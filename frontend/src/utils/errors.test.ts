// @vitest-environment node
import {describe, expect, it} from 'vitest';
import {errMsg} from './errors';

describe('errMsg', () => {
    it('devolve a string como veio (erros dos bindings do Wails chegam assim)', () => {
        expect(errMsg('token inválido')).toBe('token inválido');
    });

    it('usa a mensagem de um Error', () => {
        expect(errMsg(new Error('falhou'))).toBe('falhou');
    });

    it('usa o fallback para Error sem mensagem', () => {
        expect(errMsg(new Error(''), 'padrão')).toBe('padrão');
    });

    it.each([null, undefined, ''])('usa o fallback para %j', (valor) => {
        expect(errMsg(valor)).toBe('Erro desconhecido');
        expect(errMsg(valor, 'outro')).toBe('outro');
    });

    it('lê message ou error de objetos simples', () => {
        expect(errMsg({message: 'via message'})).toBe('via message');
        expect(errMsg({error: 'via error'})).toBe('via error');
        expect(errMsg({message: '', error: 'segunda opção'})).toBe('segunda opção');
    });

    it('serializa outros objetos em JSON', () => {
        expect(errMsg({codigo: 42})).toBe('{"codigo":42}');
    });

    it('usa o fallback para objeto vazio ou que não serializa', () => {
        expect(errMsg({})).toBe('Erro desconhecido');
        const circular: Record<string, unknown> = {};
        circular.eu = circular;
        expect(errMsg(circular)).toBe('Erro desconhecido');
    });

    it('converte primitivos', () => {
        expect(errMsg(404)).toBe('404');
        expect(errMsg(false)).toBe('false');
        expect(errMsg(10n)).toBe('10');
        expect(errMsg(Symbol('x'))).toBe('Symbol(x)');
    });

    it('nunca devolve "[object Object]" nem "undefined"', () => {
        expect(errMsg(() => 1)).toBe('Erro desconhecido');
    });
});
