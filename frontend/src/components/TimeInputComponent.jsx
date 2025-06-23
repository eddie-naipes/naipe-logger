import React from 'react';
import { FiClock } from 'react-icons/fi';

const TimeInputComponent = ({
                                hours = 0,
                                minutes = 0,
                                onTimeChange,
                                label = "Tempo",
                                disabled = false,
                                className = "",
                                showTotalMinutes = true
                            }) => {
    const handleHoursChange = (newHours) => {
        const validHours = Math.max(0, Math.min(23, parseInt(newHours) || 0));
        onTimeChange(validHours, minutes);
    };

    const handleMinutesChange = (newMinutes) => {
        const validMinutes = Math.max(0, Math.min(59, parseInt(newMinutes) || 0));
        onTimeChange(hours, validMinutes);
    };

    const getTotalMinutes = () => {
        return (hours * 60) + minutes;
    };

    const formatTotalTime = () => {
        const totalMins = getTotalMinutes();
        const h = Math.floor(totalMins / 60);
        const m = totalMins % 60;
        return `${h}h ${m}min (${totalMins} min total)`;
    };

    return (
        <div className={className}>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                <FiClock className="inline w-4 h-4 mr-1" />
                {label}
            </label>

            <div className="flex items-center space-x-2">
                <div className="flex-1">
                    <input
                        type="number"
                        min="0"
                        max="23"
                        value={hours}
                        onChange={(e) => handleHoursChange(e.target.value)}
                        className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                        placeholder="0"
                        disabled={disabled}
                    />
                    <span className="text-xs text-gray-500 dark:text-gray-400 mt-1 block">Horas</span>
                </div>

                <span className="text-gray-500 dark:text-gray-400 pb-6">:</span>

                <div className="flex-1">
                    <input
                        type="number"
                        min="0"
                        max="59"
                        value={minutes}
                        onChange={(e) => handleMinutesChange(e.target.value)}
                        className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                        placeholder="0"
                        disabled={disabled}
                    />
                    <span className="text-xs text-gray-500 dark:text-gray-400 mt-1 block">Minutos</span>
                </div>
            </div>

            {showTotalMinutes && (
                <p className="text-xs text-blue-600 dark:text-blue-400 mt-2">
                    {formatTotalTime()}
                </p>
            )}

            {/* Atalhos rápidos */}
            <div className="mt-2">
                <span className="text-xs text-gray-500 dark:text-gray-400 mb-1 block">Atalhos:</span>
                <div className="flex flex-wrap gap-1">
                    {[
                        { label: '15min', h: 0, m: 15 },
                        { label: '30min', h: 0, m: 30 },
                        { label: '1h', h: 1, m: 0 },
                        { label: '2h', h: 2, m: 0 },
                        { label: '4h', h: 4, m: 0 },
                        { label: '8h', h: 8, m: 0 }
                    ].map((shortcut) => (
                        <button
                            key={shortcut.label}
                            type="button"
                            onClick={() => onTimeChange(shortcut.h, shortcut.m)}
                            disabled={disabled}
                            className={`px-2 py-1 text-xs rounded border transition-colors ${
                                hours === shortcut.h && minutes === shortcut.m
                                    ? 'bg-primary-100 border-primary-300 text-primary-700 dark:bg-primary-900/30 dark:border-primary-700 dark:text-primary-300'
                                    : 'bg-gray-50 border-gray-200 text-gray-600 hover:bg-gray-100 dark:bg-gray-700 dark:border-gray-600 dark:text-gray-400 dark:hover:bg-gray-600'
                            } disabled:opacity-50 disabled:cursor-not-allowed`}
                        >
                            {shortcut.label}
                        </button>
                    ))}
                </div>
            </div>
        </div>
    );
};

export default TimeInputComponent;