/**
 * Canonical option lists and enumerations for GUI settings, forms, and dialogs.
 * Centralized to prevent drift and shotgun edits across views.
 */

export const RUNNER_OPTIONS = [
  { value: 'umu', label: 'UMU (Unified Linux Wine/Proton Runtime - Recommended)' },
  { value: 'wine', label: 'Wine (System Wine Binary)' }
];

export const OPERATING_MODE_OPTIONS = [
  { value: 'loader', label: 'Loader Mode (Clean / Non-Invasive - Recommended)' },
  { value: 'portable', label: 'Portable Mode (Injected DLL Replacement)' },
  { value: 'none', label: 'DRM-Free / None' }
];

export const LANGUAGE_OPTIONS = [
  { value: 'auto', label: 'Auto (System Locale)' },
  { value: 'english', label: 'English' },
  { value: 'french', label: 'French' },
  { value: 'german', label: 'German' },
  { value: 'italian', label: 'Italian' },
  { value: 'spanish', label: 'Spanish' },
  { value: 'japanese', label: 'Japanese' },
  { value: 'schinese', label: 'Simplified Chinese' },
  { value: 'russian', label: 'Russian' },
  { value: 'brazilian', label: 'Brazilian Portuguese' },
  { value: 'koreana', label: 'Korean' },
  { value: 'polish', label: 'Polish' }
];

export const COUNTRY_OPTIONS = [
  { value: 'US', label: 'United States (US)' },
  { value: 'GB', label: 'United Kingdom (GB)' },
  { value: 'DE', label: 'Germany (DE)' },
  { value: 'FR', label: 'France (FR)' },
  { value: 'CA', label: 'Canada (CA)' },
  { value: 'AU', label: 'Australia (AU)' },
  { value: 'JP', label: 'Japan (JP)' },
  { value: 'KR', label: 'South Korea (KR)' },
  { value: 'CN', label: 'China (CN)' },
  { value: 'RU', label: 'Russia (RU)' },
  { value: 'BR', label: 'Brazil (BR)' },
  { value: 'ES', label: 'Spain (ES)' },
  { value: 'IT', label: 'Italy (IT)' },
  { value: 'NL', label: 'Netherlands (NL)' },
  { value: 'PL', label: 'Poland (PL)' },
  { value: 'SE', label: 'Sweden (SE)' },
  { value: 'NO', label: 'Norway (NO)' },
  { value: 'FI', label: 'Finland (FI)' },
  { value: 'DK', label: 'Denmark (DK)' },
  { value: 'CH', label: 'Switzerland (CH)' },
  { value: 'AT', label: 'Austria (AT)' },
  { value: 'MX', label: 'Mexico (MX)' },
  { value: 'AR', label: 'Argentina (AR)' },
  { value: 'TR', label: 'Turkey (TR)' },
  { value: 'UA', label: 'Ukraine (UA)' },
  { value: 'NZ', label: 'New Zealand (NZ)' }
];

export const PLATFORM_OPTIONS = [
  { value: 'win64', label: 'Windows' },
  { value: 'linux', label: 'Linux (Native)' }
];

export const UI_SCALE_OPTIONS = [
  { value: 14, label: '14px — Compact' },
  { value: 16, label: '16px — Standard' },
  { value: 18, label: '18px — Balanced (Default)' },
  { value: 20, label: '20px — Handheld (Deck)' },
  { value: 24, label: '24px — Large (TV / Couch)' }
];

export const LAUNCH_BEHAVIOR_OPTIONS = [
  { value: 'keep', label: 'Keep Window Open (Default)' },
  { value: 'minimize', label: 'Minimize to Taskbar' },
  { value: 'hide', label: 'Hide Window (Restore on Exit)' }
];

export const BACKUP_RETENTION_OPTIONS = [
  { value: 1, label: '1 backup' },
  { value: 2, label: '2 backups' },
  { value: 3, label: '3 backups' },
  { value: 5, label: '5 backups (Recommended)' },
  { value: 10, label: '10 backups' },
  { value: 15, label: '15 backups' },
  { value: 20, label: '20 backups' },
  { value: 30, label: '30 backups' }
];

export const THEME_OPTIONS = [
  { value: 'system', label: 'System', icon: 'system', desc: 'Follow OS preference' },
  { value: 'dark', label: 'Dark', icon: 'moon', desc: 'Monochrome dark' },
  { value: 'light', label: 'Light', icon: 'sun', desc: 'Monochrome light' }
];
