export const SETTINGS_TABS = ['general', 'storage', 'compatibility', 'profiles', 'advanced'];

export const DEFAULT_CFG_DRAFT = {
  library_paths: [],
  default_runner: 'umu',
  operating_mode: 'loader',
  language: 'auto',
  mangohud: false,
  steamless: true,
  normalize: true,
  auto_backup_saves: false,
  backup_retention_count: 5,
  steam_web_api_key: '',
  hubcap_api_key: '',
  steamgriddb_api_key: '',
  rawg_api_key: '',
  artwork_dir: '',
  backup_dir: '',
  wine_prefix: '',
  enable_launch_notify: true,
  launch_window_behavior: 'keep',
  close_to_tray: false,
  disable_overlay: false,
  offline_mode: false,
  achievements: true,
  transfer_speed_limit: 0,
  save_manifest_url: '',
  global_env: {},
  multiseat: {
    player2_profile_id: 'p2',
    default_seat1_mode: 'local',
    default_seat2_mode: 'local',
    default_layout: 'dual_monitor',
    seat1_gamepad: 'auto',
    seat2_gamepad: 'auto',
    seat1_keyboard: 'auto',
    seat2_keyboard: 'auto',
    seat1_mouse: 'auto',
    seat2_mouse: 'auto',
    seat2_force_grab_cursor: false,
    seat2_grab_keyboard: false,
    seat2_show_cursor: true,
    seat2_compositor: 'auto',
    seat1_display_index: 0,
    seat2_display_index: 1,
    sunshine_port_p1: 47989,
    sunshine_port_p2: 48010,
  }
};

export const DEFAULT_USER_CFG_DRAFT = {
  account_name: 'GamuxPlayer',
  account_steamid: '76561197960287930',
  language: 'english',
  ip_country: 'US'
};

/**
 * Parses URL tab parameter into activeTab and optional advancedSubtab.
 * @param {string | null | undefined} tabParam
 * @returns {{ activeTab: string, advancedSubtab?: string } | null}
 */
export function parseTabParam(tabParam) {
  if (!tabParam) return null;
  const p = tabParam.toLowerCase().trim();
  if (['general', 'display', 'ui', 'network', 'bandwidth'].includes(p)) {
    return { activeTab: 'general' };
  }
  if (['storage', 'library', 'paths', 'credentials', 'media', 'api', 'apis', 'key', 'keys'].includes(p)) {
    return { activeTab: 'storage' };
  }
  if (['compatibility', 'compat', 'runner'].includes(p)) {
    return { activeTab: 'compatibility' };
  }
  if (['profiles', 'profile', 'steam', 'persona', 'player', 'user', 'emu', 'drm'].includes(p)) {
    return { activeTab: 'profiles' };
  }
  if (['advanced'].includes(p)) {
    return { activeTab: 'advanced' };
  }
  if (['coop', 'co-op', 'multiseat', 'split'].includes(p)) {
    return { activeTab: 'advanced', advancedSubtab: 'coop' };
  }
  if (['maintenance', 'tools', 'components'].includes(p)) {
    return { activeTab: 'advanced', advancedSubtab: 'maintenance' };
  }
  if (['logs', 'log', 'activity'].includes(p)) {
    return { activeTab: 'advanced', advancedSubtab: 'logs' };
  }
  if (['schema', 'all', 'properties'].includes(p)) {
    return { activeTab: 'advanced', advancedSubtab: 'schema' };
  }
  return null;
}

/**
 * Returns the adjacent tab when navigating via keyboard arrows.
 * @param {string} currentTab
 * @param {'next'|'prev'|number} direction
 * @param {string[]} [tabs]
 * @returns {string}
 */
export function getAdjacentTab(currentTab, direction, tabs = SETTINGS_TABS) {
  const currentIndex = tabs.indexOf(currentTab);
  if (currentIndex === -1) return tabs[0];
  if (direction === 'next' || direction === 1) {
    return tabs[(currentIndex + 1) % tabs.length];
  }
  return tabs[(currentIndex - 1 + tabs.length) % tabs.length];
}
