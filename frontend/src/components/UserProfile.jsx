// src/components/UserProfile.jsx
import React, { useState, useEffect } from 'react';
import { FiUser, FiMail } from 'react-icons/fi';
import mascotImage from '../assets/mascot.png';

const UserProfile = () => {
    const [profile, setProfile] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    useEffect(() => {
        const loadProfile = async () => {
            try {
                setLoading(true);
                const userProfile = await window.go.backend.App.GetUserProfile();
                setProfile(userProfile);
                setError(null);
            } catch (error) {
                console.error('Erro ao carregar perfil:', error);
                setError(error.message || 'Erro ao carregar perfil');
            } finally {
                setLoading(false);
            }
        };

        loadProfile();
    }, []);

    if (loading) {
        return (
            <div className="p-4 border-t border-gray-200 dark:border-gray-700">
                <div className="flex items-center space-x-3">
                    <div className="w-10 h-10 bg-gray-200 dark:bg-gray-600 rounded-full animate-pulse"></div>
                    <div className="flex-1">
                        <div className="h-4 bg-gray-200 dark:bg-gray-600 rounded animate-pulse mb-1"></div>
                        <div className="h-3 bg-gray-200 dark:bg-gray-600 rounded animate-pulse w-2/3"></div>
                    </div>
                </div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="p-4 border-t border-gray-200 dark:border-gray-700">
                <div className="flex items-center space-x-3">
                    <div className="w-10 h-10 bg-gray-300 dark:bg-gray-600 rounded-full flex items-center justify-center">
                        <FiUser className="w-5 h-5 text-gray-500 dark:text-gray-400" />
                    </div>
                    <div className="flex-1">
                        <p className="text-sm text-gray-500 dark:text-gray-400">Usuário</p>
                        <p className="text-xs text-gray-400 dark:text-gray-500">Não conectado</p>
                    </div>
                </div>
            </div>
        );
    }

    return (
        <div className="p-4 border-t border-gray-200 dark:border-gray-700">
            <div className="flex items-center space-x-3">
                <div className="relative">
                    <img
                        src={profile?.avatarURL || mascotImage}
                        alt="Avatar"
                        className="w-10 h-10 rounded-full object-cover border-2 border-gray-200 dark:border-gray-600"
                        onError={(e) => {
                            e.target.src = mascotImage;
                        }}
                    />
                    <div className="absolute -bottom-1 -right-1 w-4 h-4 bg-green-400 border-2 border-white dark:border-gray-800 rounded-full"></div>
                </div>
                <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-gray-900 dark:text-white truncate">
                        {profile?.fullName || 'Usuário'}
                    </p>
                    <div className="flex items-center text-xs text-gray-500 dark:text-gray-400">
                        <FiMail className="w-3 h-3 mr-1" />
                        <span className="truncate">{profile?.email || 'email@exemplo.com'}</span>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default UserProfile;