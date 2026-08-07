package gbe

import (
	"context"
	"fmt"
	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ApplyGBE applies the GBE emulator.
// If portable is true, it performs direct DLL/SO replacement (copying emulated libraries into the game dir).
// If portable is false (loader mode), it leaves original game libraries untouched and sets up Wine registry/loader config.
func ApplyGBE(ctx context.Context, cfg *config.Config, targetDir, platform, appID string, dryRun, portable bool, winePrefix string, exePath string) error {


	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	if targetDir == "" {
		targetDir = "."
	}

	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("resolve target dir: %w", err)
	}

	platformCfg, ok := config.PlatformConfig[platform]
	if !ok {
		var validPlatforms []string
		for p := range config.PlatformConfig {
			validPlatforms = append(validPlatforms, p)
		}
		return fmt.Errorf("invalid platform: '%s'. Valid platforms: %s", platform, strings.Join(validPlatforms, ", "))
	}

	gbeBase := cfg.GbeDir
	if !filepath.IsAbs(gbeBase) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		gbeBase = filepath.Join(homeDir, gbeBase)
	}

	gbePath := filepath.Join(gbeBase, platformCfg.Subdir, "experimental", "x"+platformCfg.Arch)

	var targetFiles []string
	walkErr := filepath.WalkDir(absTargetDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == platformCfg.Target {
			targetFiles = append(targetFiles, path)
		}
		return nil
	})
	if walkErr != nil {
		return fmt.Errorf("failed to search for files in %s: %w", absTargetDir, walkErr)
	}

	if len(targetFiles) == 0 {
		slog.Warn("No target files found", "platform", platform, "dir", absTargetDir)
	}

	sourceFile := filepath.Join(gbePath, platformCfg.Target)
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		return fmt.Errorf("source file not found: '%s'", sourceFile)
	}

	sourceHash, err := util.GetHash(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to get hash of source file: %w", err)
	}

	for _, file := range targetFiles {
		slog.Info("Found target library", "file", file)

		if portable {
			targetHash, err := util.GetHash(file)
			if err != nil {
				slog.Error("Failed to get hash", "file", file, "error", err)
				continue
			}

			if targetHash == sourceHash {
				slog.Info("File is already up-to-date", "file", file)
			} else {
				if dryRun {
					slog.Info("[DRY RUN] Would replace file with Goldberg emulator", "file", file)
				} else if err := util.BackupAndReplace(sourceFile, file); err != nil {
					slog.Error("Failed to replace file", "file", file, "error", err)
					continue
				}
			}

			if platformCfg.Additional != "" {
				additionalSource := filepath.Join(gbePath, platformCfg.Additional)
				additionalDest := filepath.Join(filepath.Dir(file), platformCfg.Additional)
				if _, err := os.Stat(additionalSource); err == nil {
					if _, err := os.Stat(additionalDest); err == nil {
						if dryRun {
							slog.Info("[DRY RUN] Would replace additional file", "dest", additionalDest)
						} else if err := util.BackupAndReplace(additionalSource, additionalDest); err != nil {
							slog.Warn("Failed to replace additional file", "dest", additionalDest, "error", err)
						}
					}
				}
			}
		} else {
			slog.Info("[LOADER MODE] Keeping original library intact (zero file replacement)", "file", file)
			if platform == "win64" || platform == "win32" {
				targetExe := exePath
				if targetExe == "" {
					targetExe = file
				}
				if err := SetupLoaderMode(gbeBase, absTargetDir, platform, appID, targetExe, dryRun); err != nil {
					slog.Warn("Failed to setup Goldberg loader mode", "error", err)
				}
				dllPath := filepath.Join(gbePath, platformCfg.Additional)
				ConfigureWineRegistry(winePrefix, platform, dllPath, dryRun)
			}
		}



		generatorPath := filepath.Join(gbeBase, platformCfg.Subdir, "tools", "generate_interfaces", platformCfg.Generator)
		if _, err := os.Stat(generatorPath); err == nil {
			if dryRun {
				slog.Info("[DRY RUN] Would run generator", "generator", platformCfg.Generator)
			} else {
				slog.Info("Running generator", "generator", platformCfg.Generator)
				var cmd *exec.Cmd
				if strings.HasSuffix(platformCfg.Generator, ".exe") && runtime.GOOS != "windows" {
					winePath, err := exec.LookPath("wine")
					if err != nil {
						slog.Warn("Wine binary not found in PATH; skipping generator execution", "generator", platformCfg.Generator)
					} else {
						cmd = exec.Command(winePath, generatorPath, filepath.Base(file))
					}
				} else {
					if runtime.GOOS != "windows" {
						if err := os.Chmod(generatorPath, 0755); err != nil {
							slog.Warn("Failed to set executable permissions", "path", generatorPath, "error", err)
						}
					}
					cmd = exec.Command(generatorPath, filepath.Base(file))
				}

				if cmd != nil {
					cmd.Dir = filepath.Dir(file)
					if out, err := cmd.CombinedOutput(); err != nil {
						slog.Error("Generator failed", "error", err, "output", string(out))
					}
				}
			}
		}
	}

	// After applying GBE, fetch and configure DLCs
	for _, file := range targetFiles {
		libraryPath := filepath.Dir(file)
		if err := steam.FetchDLCs(ctx, appID, libraryPath, dryRun); err != nil {
			slog.Warn("Failed to fetch and configure DLCs", "appID", appID, "libraryPath", libraryPath, "error", err)
		}
	}

	slog.Info("GBE application process completed")
	return nil
}

// ConfigureWineRegistry configures the Wine registry to load Goldberg's steamclient DLL for non-portable loader mode.
func ConfigureWineRegistry(winePrefix, platform, dllPath string, dryRun bool) {
	winePath, err := exec.LookPath("wine")
	if err != nil {
		slog.Warn("Wine binary not found in PATH; skipping Wine registry configuration", "platform", platform)
		return
	}

	regKey := "SteamClientDll64"
	if platform == "win32" {
		regKey = "SteamClientDll"
	}

	if dryRun {
		slog.Info("[DRY RUN] Would configure Wine registry key", "key", regKey, "path", dllPath)
		return
	}

	cmd := exec.Command(winePath, "reg", "add", `HKCU\Software\Valve\Steam`, "/v", regKey, "/t", "REG_SZ", "/d", dllPath, "/f")
	if winePrefix != "" {
		cmd.Env = append(os.Environ(), "WINEPREFIX="+winePrefix)
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		slog.Warn("Failed to set Wine registry key", "key", regKey, "error", err, "output", string(out))
	} else {
		slog.Info("Successfully configured Wine registry key for Goldberg loader", "key", regKey, "path", dllPath)
	}
}

// SetupLoaderMode configures Goldberg's official helper loader (steamclient_loader_x64/x86.exe and ColdClientLoader.ini)
// by creating relative symlinks to central GBE binaries in ~/.local/share/gbe_fork/ and generating ColdClientLoader.ini.
func SetupLoaderMode(gbeBase, absTargetDir, platform, appID, exePath string, dryRun bool) error {
	if platform != "win64" && platform != "win32" {
		return nil
	}

	loaderDir := filepath.Join(gbeBase, "win_release", "steamclient_experimental")
	loaderExeName := "steamclient_loader_x64.exe"
	if platform == "win32" {
		loaderExeName = "steamclient_loader_x86.exe"
	}

	filesToSymlink := []string{
		loaderExeName,
		"steamclient64.dll",
		"steamclient.dll",
		"GameOverlayRenderer64.dll",
		"GameOverlayRenderer.dll",
	}

	for _, fname := range filesToSymlink {
		src := filepath.Join(loaderDir, fname)
		dst := filepath.Join(absTargetDir, fname)
		if _, err := os.Stat(src); os.IsNotExist(err) {
			slog.Warn("GBE loader asset not found", "src", src)
			continue
		}

		if dryRun {
			slog.Info("[DRY RUN] Would symlink loader asset", "from", src, "to", dst)
			continue
		}

		_ = os.Remove(dst)
		if err := os.Symlink(src, dst); err != nil {
			slog.Warn("Failed to symlink loader asset, falling back to copy", "from", src, "to", dst, "error", err)
			if copyErr := util.CopyFile(src, dst); copyErr != nil {
				slog.Error("Failed to copy loader asset fallback", "from", src, "to", dst, "error", copyErr)
			}
		} else {
			slog.Info("Symlinked GBE loader asset", "file", fname, "dst", dst)
		}
	}

	relExe := exePath
	if filepath.IsAbs(exePath) {
		if r, err := filepath.Rel(absTargetDir, exePath); err == nil {
			relExe = r
		} else {
			relExe = filepath.Base(exePath)
		}
	}

	iniContent := fmt.Sprintf(`[SteamClient]
Exe=%s
ExeRunDir=
ExeCommandLine=
AppId=%s
SteamClientDll=steamclient.dll
SteamClient64Dll=steamclient64.dll

[Injection]
ForceInjectSteamClient=1
ForceInjectGameOverlayRenderer=0
IgnoreInjectionError=0
IgnoreLoaderArchDifference=0
`, relExe, appID)

	iniPath := filepath.Join(absTargetDir, "ColdClientLoader.ini")
	if dryRun {
		slog.Info("[DRY RUN] Would write ColdClientLoader.ini", "path", iniPath)
	} else {
		if err := os.WriteFile(iniPath, []byte(iniContent), 0644); err != nil {
			return fmt.Errorf("write ColdClientLoader.ini: %w", err)
		}
		slog.Info("Successfully wrote ColdClientLoader.ini", "path", iniPath, "appID", appID, "exe", relExe)
	}

	return nil
}


