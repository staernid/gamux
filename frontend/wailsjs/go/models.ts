export namespace config {
	
	export class Config {
	    library_paths?: string[];
	    gbe_dir?: string;
	    steamless_dir?: string;
	    steam_userdata?: string;
	    steam_web_api_key?: string;
	    hubcap_api_key?: string;
	    steamgriddb_api_key?: string;
	    rawg_api_key?: string;
	    artwork_dir?: string;
	    enable_launch_notify: boolean;
	    operating_mode?: string;
	    default_runner?: string;
	    proton_path?: string;
	    steamless: boolean;
	    achievements: boolean;
	    normalize: boolean;
	    wine_prefix?: string;
	    mangohud?: boolean;
	    dxvk_hud?: string;
	    wine_fullscreen_fsr?: boolean;
	    proton_nvapi?: boolean;
	    transfer_speed_limit?: number;
	    global_env?: Record<string, string>;
	    language?: string;
	    disable_overlay?: boolean;
	    offline_mode?: boolean;
	    auto_backup_saves: boolean;
	    backup_retention_count?: number;
	    backup_dir?: string;
	    save_manifest_url?: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.library_paths = source["library_paths"];
	        this.gbe_dir = source["gbe_dir"];
	        this.steamless_dir = source["steamless_dir"];
	        this.steam_userdata = source["steam_userdata"];
	        this.steam_web_api_key = source["steam_web_api_key"];
	        this.hubcap_api_key = source["hubcap_api_key"];
	        this.steamgriddb_api_key = source["steamgriddb_api_key"];
	        this.rawg_api_key = source["rawg_api_key"];
	        this.artwork_dir = source["artwork_dir"];
	        this.enable_launch_notify = source["enable_launch_notify"];
	        this.operating_mode = source["operating_mode"];
	        this.default_runner = source["default_runner"];
	        this.proton_path = source["proton_path"];
	        this.steamless = source["steamless"];
	        this.achievements = source["achievements"];
	        this.normalize = source["normalize"];
	        this.wine_prefix = source["wine_prefix"];
	        this.mangohud = source["mangohud"];
	        this.dxvk_hud = source["dxvk_hud"];
	        this.wine_fullscreen_fsr = source["wine_fullscreen_fsr"];
	        this.proton_nvapi = source["proton_nvapi"];
	        this.transfer_speed_limit = source["transfer_speed_limit"];
	        this.global_env = source["global_env"];
	        this.language = source["language"];
	        this.disable_overlay = source["disable_overlay"];
	        this.offline_mode = source["offline_mode"];
	        this.auto_backup_saves = source["auto_backup_saves"];
	        this.backup_retention_count = source["backup_retention_count"];
	        this.backup_dir = source["backup_dir"];
	        this.save_manifest_url = source["save_manifest_url"];
	    }
	}

}

export namespace domain {
	
	export class AchievementItem {
	    id: string;
	    name: string;
	    description: string;
	    icon_url: string;
	    icon_gray_url: string;
	    hidden: boolean;
	    unlocked: boolean;
	    unlock_time?: number;
	
	    static createFrom(source: any = {}) {
	        return new AchievementItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.icon_url = source["icon_url"];
	        this.icon_gray_url = source["icon_gray_url"];
	        this.hidden = source["hidden"];
	        this.unlocked = source["unlocked"];
	        this.unlock_time = source["unlock_time"];
	    }
	}
	export class ArtworkOption {
	    id: string;
	    provider: string;
	    type: string;
	    url: string;
	    thumb_url: string;
	    width?: number;
	    height?: number;
	    style?: string;
	    score?: number;
	    author?: string;
	
	    static createFrom(source: any = {}) {
	        return new ArtworkOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.provider = source["provider"];
	        this.type = source["type"];
	        this.url = source["url"];
	        this.thumb_url = source["thumb_url"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.style = source["style"];
	        this.score = source["score"];
	        this.author = source["author"];
	    }
	}
	export class ArtworkQuery {
	    game_dir?: string;
	    title?: string;
	    steam_app_id?: number;
	    gog_id?: string;
	    sgdb_id?: string;
	    asset_type: string;
	
	    static createFrom(source: any = {}) {
	        return new ArtworkQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game_dir = source["game_dir"];
	        this.title = source["title"];
	        this.steam_app_id = source["steam_app_id"];
	        this.gog_id = source["gog_id"];
	        this.sgdb_id = source["sgdb_id"];
	        this.asset_type = source["asset_type"];
	    }
	}
	export class DepotSummary {
	    depot_id: number;
	    name: string;
	    category: string;
	    installed: boolean;
	    size: number;
	    file_count: number;
	    manifest_id: number;
	
	    static createFrom(source: any = {}) {
	        return new DepotSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.depot_id = source["depot_id"];
	        this.name = source["name"];
	        this.category = source["category"];
	        this.installed = source["installed"];
	        this.size = source["size"];
	        this.file_count = source["file_count"];
	        this.manifest_id = source["manifest_id"];
	    }
	}
	export class NewsItem {
	    id: string;
	    title: string;
	    url: string;
	    author?: string;
	    contents: string;
	    feedlabel?: string;
	    date: number;
	    feedname?: string;
	    appid?: number;
	
	    static createFrom(source: any = {}) {
	        return new NewsItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.author = source["author"];
	        this.contents = source["contents"];
	        this.feedlabel = source["feedlabel"];
	        this.date = source["date"];
	        this.feedname = source["feedname"];
	        this.appid = source["appid"];
	    }
	}
	export class LaunchProfile {
	    game_dir: string;
	    app_id?: string;
	    name?: string;
	    platform?: string;
	    exe_path: string;
	    exe_args?: string;
	    operating_mode?: string;
	    runner?: string;
	    wine_prefix?: string;
	    proton_path?: string;
	    language?: string;
	    mangohud?: boolean;
	    dxvk_hud?: string;
	    wine_fullscreen_fsr?: boolean;
	    proton_nvapi?: boolean;
	    dll_overrides?: string;
	    custom_env?: Record<string, string>;
	    working_dir?: string;
	    store?: string;
	    store_app_id?: string;
	    source_override?: string;
	    metadata_provider?: string;
	    metadata_id?: string;
	    gog_id?: string;
	    sgdb_id?: string;
	    custom_cover_url?: string;
	    custom_banner_url?: string;
	    custom_hero_url?: string;
	    custom_logo_url?: string;
	    custom_icon_url?: string;
	    locked_artwork?: boolean;
	    playtime_seconds?: number;
	    last_played?: number;
	
	    static createFrom(source: any = {}) {
	        return new LaunchProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game_dir = source["game_dir"];
	        this.app_id = source["app_id"];
	        this.name = source["name"];
	        this.platform = source["platform"];
	        this.exe_path = source["exe_path"];
	        this.exe_args = source["exe_args"];
	        this.operating_mode = source["operating_mode"];
	        this.runner = source["runner"];
	        this.wine_prefix = source["wine_prefix"];
	        this.proton_path = source["proton_path"];
	        this.language = source["language"];
	        this.mangohud = source["mangohud"];
	        this.dxvk_hud = source["dxvk_hud"];
	        this.wine_fullscreen_fsr = source["wine_fullscreen_fsr"];
	        this.proton_nvapi = source["proton_nvapi"];
	        this.dll_overrides = source["dll_overrides"];
	        this.custom_env = source["custom_env"];
	        this.working_dir = source["working_dir"];
	        this.store = source["store"];
	        this.store_app_id = source["store_app_id"];
	        this.source_override = source["source_override"];
	        this.metadata_provider = source["metadata_provider"];
	        this.metadata_id = source["metadata_id"];
	        this.gog_id = source["gog_id"];
	        this.sgdb_id = source["sgdb_id"];
	        this.custom_cover_url = source["custom_cover_url"];
	        this.custom_banner_url = source["custom_banner_url"];
	        this.custom_hero_url = source["custom_hero_url"];
	        this.custom_logo_url = source["custom_logo_url"];
	        this.custom_icon_url = source["custom_icon_url"];
	        this.locked_artwork = source["locked_artwork"];
	        this.playtime_seconds = source["playtime_seconds"];
	        this.last_played = source["last_played"];
	    }
	}
	export class LaunchCandidate {
	    name: string;
	    executable: string;
	    arguments?: string;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new LaunchCandidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.executable = source["executable"];
	        this.arguments = source["arguments"];
	        this.description = source["description"];
	    }
	}
	export class Game {
	    game_dir: string;
	    app_id: string;
	    numeric_app_id: number;
	    name: string;
	    install_dir?: string;
	    store: string;
	    source: string;
	    platform: string;
	    steam_appid?: number;
	    gog_id?: string;
	    epic_app?: string;
	    sgdb_id?: string;
	    rawg_id?: string;
	    igdb_id?: string;
	    disk_size_bytes: number;
	    file_count: number;
	    dlc_count?: number;
	    achievement_count?: number;
	    launch_candidates?: LaunchCandidate[];
	    depots?: DepotSummary[];
	    available_platforms?: string[];
	    supported_languages?: string[];
	    target_lib_path?: string;
	    manifest_path?: string;
	    raw_depot_layout?: boolean;
	    exe_path?: string;
	    exe_args?: string;
	    launch_profile?: LaunchProfile;
	    state: string;
	    is_patched: boolean;
	    is_portable: boolean;
	    has_updates_or_issues: boolean;
	    is_ready: boolean;
	    original_backups?: string[];
	    settings_dir_exists: boolean;
	    steam_appid_txt_found: boolean;
	    current_language?: string;
	    cover_url?: string;
	    banner_url?: string;
	    hero_url?: string;
	    logo_url?: string;
	    icon_url?: string;
	    local_cover?: string;
	    local_banner?: string;
	    local_hero?: string;
	    local_logo?: string;
	    local_icon?: string;
	    local_cover_url?: string;
	    local_banner_url?: string;
	    local_hero_url?: string;
	    local_logo_url?: string;
	    custom_cover_url?: string;
	    custom_banner_url?: string;
	    custom_hero_url?: string;
	    custom_logo_url?: string;
	    custom_icon_url?: string;
	    locked_artwork?: boolean;
	    manifest_id?: string;
	    build_id?: string;
	    remote_manifest_id?: string;
	    official_file_count: number;
	    modified_files?: string[];
	    missing_files?: string[];
	    untracked_files?: string[];
	    redist_modified?: string[];
	    redist_missing?: string[];
	    has_update: boolean;
	    recent_patch_note?: string;
	    news_items?: NewsItem[];
	
	    static createFrom(source: any = {}) {
	        return new Game(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game_dir = source["game_dir"];
	        this.app_id = source["app_id"];
	        this.numeric_app_id = source["numeric_app_id"];
	        this.name = source["name"];
	        this.install_dir = source["install_dir"];
	        this.store = source["store"];
	        this.source = source["source"];
	        this.platform = source["platform"];
	        this.steam_appid = source["steam_appid"];
	        this.gog_id = source["gog_id"];
	        this.epic_app = source["epic_app"];
	        this.sgdb_id = source["sgdb_id"];
	        this.rawg_id = source["rawg_id"];
	        this.igdb_id = source["igdb_id"];
	        this.disk_size_bytes = source["disk_size_bytes"];
	        this.file_count = source["file_count"];
	        this.dlc_count = source["dlc_count"];
	        this.achievement_count = source["achievement_count"];
	        this.launch_candidates = this.convertValues(source["launch_candidates"], LaunchCandidate);
	        this.depots = this.convertValues(source["depots"], DepotSummary);
	        this.available_platforms = source["available_platforms"];
	        this.supported_languages = source["supported_languages"];
	        this.target_lib_path = source["target_lib_path"];
	        this.manifest_path = source["manifest_path"];
	        this.raw_depot_layout = source["raw_depot_layout"];
	        this.exe_path = source["exe_path"];
	        this.exe_args = source["exe_args"];
	        this.launch_profile = this.convertValues(source["launch_profile"], LaunchProfile);
	        this.state = source["state"];
	        this.is_patched = source["is_patched"];
	        this.is_portable = source["is_portable"];
	        this.has_updates_or_issues = source["has_updates_or_issues"];
	        this.is_ready = source["is_ready"];
	        this.original_backups = source["original_backups"];
	        this.settings_dir_exists = source["settings_dir_exists"];
	        this.steam_appid_txt_found = source["steam_appid_txt_found"];
	        this.current_language = source["current_language"];
	        this.cover_url = source["cover_url"];
	        this.banner_url = source["banner_url"];
	        this.hero_url = source["hero_url"];
	        this.logo_url = source["logo_url"];
	        this.icon_url = source["icon_url"];
	        this.local_cover = source["local_cover"];
	        this.local_banner = source["local_banner"];
	        this.local_hero = source["local_hero"];
	        this.local_logo = source["local_logo"];
	        this.local_icon = source["local_icon"];
	        this.local_cover_url = source["local_cover_url"];
	        this.local_banner_url = source["local_banner_url"];
	        this.local_hero_url = source["local_hero_url"];
	        this.local_logo_url = source["local_logo_url"];
	        this.custom_cover_url = source["custom_cover_url"];
	        this.custom_banner_url = source["custom_banner_url"];
	        this.custom_hero_url = source["custom_hero_url"];
	        this.custom_logo_url = source["custom_logo_url"];
	        this.custom_icon_url = source["custom_icon_url"];
	        this.locked_artwork = source["locked_artwork"];
	        this.manifest_id = source["manifest_id"];
	        this.build_id = source["build_id"];
	        this.remote_manifest_id = source["remote_manifest_id"];
	        this.official_file_count = source["official_file_count"];
	        this.modified_files = source["modified_files"];
	        this.missing_files = source["missing_files"];
	        this.untracked_files = source["untracked_files"];
	        this.redist_modified = source["redist_modified"];
	        this.redist_missing = source["redist_missing"];
	        this.has_update = source["has_update"];
	        this.recent_patch_note = source["recent_patch_note"];
	        this.news_items = this.convertValues(source["news_items"], NewsItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GameArtwork {
	    cover_url?: string;
	    banner_url?: string;
	    hero_url?: string;
	    logo_url?: string;
	    icon_url?: string;
	    local_cover?: string;
	    local_banner?: string;
	    local_hero?: string;
	    local_logo?: string;
	    local_icon?: string;
	    local_cover_url?: string;
	    local_banner_url?: string;
	    local_hero_url?: string;
	    local_logo_url?: string;
	    custom_cover_url?: string;
	    custom_banner_url?: string;
	    custom_hero_url?: string;
	    custom_logo_url?: string;
	    custom_icon_url?: string;
	    locked_artwork?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GameArtwork(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cover_url = source["cover_url"];
	        this.banner_url = source["banner_url"];
	        this.hero_url = source["hero_url"];
	        this.logo_url = source["logo_url"];
	        this.icon_url = source["icon_url"];
	        this.local_cover = source["local_cover"];
	        this.local_banner = source["local_banner"];
	        this.local_hero = source["local_hero"];
	        this.local_logo = source["local_logo"];
	        this.local_icon = source["local_icon"];
	        this.local_cover_url = source["local_cover_url"];
	        this.local_banner_url = source["local_banner_url"];
	        this.local_hero_url = source["local_hero_url"];
	        this.local_logo_url = source["local_logo_url"];
	        this.custom_cover_url = source["custom_cover_url"];
	        this.custom_banner_url = source["custom_banner_url"];
	        this.custom_hero_url = source["custom_hero_url"];
	        this.custom_logo_url = source["custom_logo_url"];
	        this.custom_icon_url = source["custom_icon_url"];
	        this.locked_artwork = source["locked_artwork"];
	    }
	}
	

}

export namespace engine {
	
	export class DownloadGameOptions {
	    app_id: number;
	    target_dir: string;
	    platform: string;
	    lua_path: string;
	    dry_run: boolean;
	    worker_limit?: number;
	    auto_apply: boolean;
	    apply_options?: pipeline.Options;
	
	    static createFrom(source: any = {}) {
	        return new DownloadGameOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.app_id = source["app_id"];
	        this.target_dir = source["target_dir"];
	        this.platform = source["platform"];
	        this.lua_path = source["lua_path"];
	        this.dry_run = source["dry_run"];
	        this.worker_limit = source["worker_limit"];
	        this.auto_apply = source["auto_apply"];
	        this.apply_options = this.convertValues(source["apply_options"], pipeline.Options);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace gbe {
	
	export class GSEUserConfig {
	    account_name: string;
	    account_steamid: string;
	    language: string;
	    ip_country: string;
	
	    static createFrom(source: any = {}) {
	        return new GSEUserConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.account_name = source["account_name"];
	        this.account_steamid = source["account_steamid"];
	        this.language = source["language"];
	        this.ip_country = source["ip_country"];
	    }
	}

}

export namespace gui {
	
	export class GameMetadataDetails {
	    name: string;
	    short_description: string;
	    detailed_description: string;
	    about_the_game: string;
	    header_image: string;
	    developers: string[];
	    publishers: string[];
	    genres: string[];
	    release_date: string;
	
	    static createFrom(source: any = {}) {
	        return new GameMetadataDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.short_description = source["short_description"];
	        this.detailed_description = source["detailed_description"];
	        this.about_the_game = source["about_the_game"];
	        this.header_image = source["header_image"];
	        this.developers = source["developers"];
	        this.publishers = source["publishers"];
	        this.genres = source["genres"];
	        this.release_date = source["release_date"];
	    }
	}

}

export namespace manifest {
	
	export class IntegrityReport {
	    official_count: number;
	    intact_count: number;
	    modified_files?: string[];
	    missing_files?: string[];
	    untracked_files?: string[];
	    redist_modified?: string[];
	    redist_missing?: string[];
	
	    static createFrom(source: any = {}) {
	        return new IntegrityReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.official_count = source["official_count"];
	        this.intact_count = source["intact_count"];
	        this.modified_files = source["modified_files"];
	        this.missing_files = source["missing_files"];
	        this.untracked_files = source["untracked_files"];
	        this.redist_modified = source["redist_modified"];
	        this.redist_missing = source["redist_missing"];
	    }
	}

}

export namespace pipeline {
	
	export class Options {
	    path: string;
	    operating_mode?: string;
	    normalize_dir: boolean;
	    dry_run: boolean;
	    auto_yes: boolean;
	    no_steamless: boolean;
	    fetch_achievements: boolean;
	    language?: string;
	    wine_prefix?: string;
	    exe_path?: string;
	    exe_args?: string;
	
	    static createFrom(source: any = {}) {
	        return new Options(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.operating_mode = source["operating_mode"];
	        this.normalize_dir = source["normalize_dir"];
	        this.dry_run = source["dry_run"];
	        this.auto_yes = source["auto_yes"];
	        this.no_steamless = source["no_steamless"];
	        this.fetch_achievements = source["fetch_achievements"];
	        this.language = source["language"];
	        this.wine_prefix = source["wine_prefix"];
	        this.exe_path = source["exe_path"];
	        this.exe_args = source["exe_args"];
	    }
	}

}

export namespace saves {
	
	export class BackupInfo {
	    id: string;
	    zip_path: string;
	    app_id?: string;
	    game_title: string;
	    created_at: number;
	    created_by: string;
	    note?: string;
	    file_count: number;
	    size_bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new BackupInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.zip_path = source["zip_path"];
	        this.app_id = source["app_id"];
	        this.game_title = source["game_title"];
	        this.created_at = source["created_at"];
	        this.created_by = source["created_by"];
	        this.note = source["note"];
	        this.file_count = source["file_count"];
	        this.size_bytes = source["size_bytes"];
	    }
	}
	export class SaveLocation {
	    path: string;
	    display_path: string;
	    category: string;
	    tags?: string[];
	    exists: boolean;
	    is_dir: boolean;
	    file_count: number;
	    size_bytes: number;
	    source: string;
	    shared_with?: string[];
	
	    static createFrom(source: any = {}) {
	        return new SaveLocation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.display_path = source["display_path"];
	        this.category = source["category"];
	        this.tags = source["tags"];
	        this.exists = source["exists"];
	        this.is_dir = source["is_dir"];
	        this.file_count = source["file_count"];
	        this.size_bytes = source["size_bytes"];
	        this.source = source["source"];
	        this.shared_with = source["shared_with"];
	    }
	}
	export class SaveLocationsResult {
	    game_dir: string;
	    game_name: string;
	    app_id?: string;
	    platform?: string;
	    locations: SaveLocation[];
	    total_save_bytes: number;
	    total_config_files: number;
	    has_cloud_saves: boolean;
	    primary_save_path?: string;
	    primary_config_path?: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveLocationsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game_dir = source["game_dir"];
	        this.game_name = source["game_name"];
	        this.app_id = source["app_id"];
	        this.platform = source["platform"];
	        this.locations = this.convertValues(source["locations"], SaveLocation);
	        this.total_save_bytes = source["total_save_bytes"];
	        this.total_config_files = source["total_config_files"];
	        this.has_cloud_saves = source["has_cloud_saves"];
	        this.primary_save_path = source["primary_save_path"];
	        this.primary_config_path = source["primary_config_path"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace store {
	
	export class ProviderCapabilities {
	    can_search: boolean;
	    can_fetch_metadata: boolean;
	    can_provide_artwork: boolean;
	    is_game_store: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProviderCapabilities(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.can_search = source["can_search"];
	        this.can_fetch_metadata = source["can_fetch_metadata"];
	        this.can_provide_artwork = source["can_provide_artwork"];
	        this.is_game_store = source["is_game_store"];
	    }
	}
	export class ProviderDescriptor {
	    id: string;
	    display_name: string;
	    description?: string;
	    capabilities: ProviderCapabilities;
	
	    static createFrom(source: any = {}) {
	        return new ProviderDescriptor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.display_name = source["display_name"];
	        this.description = source["description"];
	        this.capabilities = this.convertValues(source["capabilities"], ProviderCapabilities);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SearchResult {
	    id: string;
	    store: string;
	    name: string;
	    poster_url?: string;
	    release_date?: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.store = source["store"];
	        this.name = source["name"];
	        this.poster_url = source["poster_url"];
	        this.release_date = source["release_date"];
	    }
	}

}

export namespace tasks {
	
	export class Progress {
	    current_bytes: number;
	    total_bytes: number;
	    current_items: number;
	    total_items: number;
	    bytes_per_sec: number;
	    eta_seconds: number;
	    percent: number;
	
	    static createFrom(source: any = {}) {
	        return new Progress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current_bytes = source["current_bytes"];
	        this.total_bytes = source["total_bytes"];
	        this.current_items = source["current_items"];
	        this.total_items = source["total_items"];
	        this.bytes_per_sec = source["bytes_per_sec"];
	        this.eta_seconds = source["eta_seconds"];
	        this.percent = source["percent"];
	    }
	}
	export class Task {
	    id: string;
	    type: string;
	    title: string;
	    game_name?: string;
	    game_dir?: string;
	    app_id?: string;
	    status: string;
	    stage: string;
	    details: string;
	    capabilities: number;
	    progress: Progress;
	    error?: string;
	    created_at: number;
	    started_at?: number;
	    finished_at?: number;
	    metadata?: Record<string, string>;
	    game?: domain.Game;
	
	    static createFrom(source: any = {}) {
	        return new Task(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.title = source["title"];
	        this.game_name = source["game_name"];
	        this.game_dir = source["game_dir"];
	        this.app_id = source["app_id"];
	        this.status = source["status"];
	        this.stage = source["stage"];
	        this.details = source["details"];
	        this.capabilities = source["capabilities"];
	        this.progress = this.convertValues(source["progress"], Progress);
	        this.error = source["error"];
	        this.created_at = source["created_at"];
	        this.started_at = source["started_at"];
	        this.finished_at = source["finished_at"];
	        this.metadata = source["metadata"];
	        this.game = this.convertValues(source["game"], domain.Game);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace tools {
	
	export class ToolStatus {
	    name: string;
	    key: string;
	    installed_path: string;
	    installed_tag: string;
	    latest_tag: string;
	    release_name: string;
	    release_notes: string;
	    release_url: string;
	    published_at: string;
	    has_update: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ToolStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.key = source["key"];
	        this.installed_path = source["installed_path"];
	        this.installed_tag = source["installed_tag"];
	        this.latest_tag = source["latest_tag"];
	        this.release_name = source["release_name"];
	        this.release_notes = source["release_notes"];
	        this.release_url = source["release_url"];
	        this.published_at = source["published_at"];
	        this.has_update = source["has_update"];
	    }
	}

}

export namespace util {
	
	export class DiskSpaceInfo {
	    path: string;
	    total_bytes: number;
	    free_bytes: number;
	    used_bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new DiskSpaceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.total_bytes = source["total_bytes"];
	        this.free_bytes = source["free_bytes"];
	        this.used_bytes = source["used_bytes"];
	    }
	}

}

