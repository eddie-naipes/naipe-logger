import React from 'react';
import { Link } from 'react-router-dom';
import { FiHome, FiAlertCircle } from 'react-icons/fi';

const NotFound = () => {
    return (
        <div className="flex flex-col items-center justify-center h-full">
            <div className="text-center">
                <div className="flex justify-center mb-4">
                    <FiAlertCircle className="w-20 h-20 text-gray-400 dark:text-gray-600" />
                </div>
                <h1 className="text-4xl font-bold text-gray-900 dark:text-white mb-4">
                    404
                </h1>
                <p className="text-lg text-gray-600 dark:text-gray-400 mb-6">
                    Página não encontrada
                </p>
                <p className="text-gray-500 dark:text-gray-500 mb-8">
                    A página que você está procurando não existe ou foi movida.
                </p>
                <Link
                    to="/"
                    className="btn-primary inline-flex items-center px-6 py-3"
                >
                    <FiHome className="w-5 h-5 mr-2" />
                    Voltar para o Dashboard
                </Link>
            </div>
        </div>
    );
};

export default NotFound;