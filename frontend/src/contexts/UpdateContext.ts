import {createContext, useContext} from 'react';
import type {UseUpdateResult} from '../hooks/useUpdate';

// Estado único de atualização do app: o aviso global e a página de
// configuração enxergam a mesma verificação, o mesmo progresso e o mesmo
// "dispensar nesta sessão".
export const UpdateContext = createContext<UseUpdateResult | null>(null);

export const useUpdateContext = (): UseUpdateResult => {
    const valor = useContext(UpdateContext);
    if (!valor) {
        throw new Error('useUpdateContext precisa estar dentro de UpdateContext.Provider');
    }
    return valor;
};
