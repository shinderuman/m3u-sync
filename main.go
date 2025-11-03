package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	tempDirPrefix     = "music_sync_*"
	usbMusicDir       = "Music"
	maxPreviewEntries = 5
)

type Config struct {
	PlaylistGlob string
	USBRoot      string
	DryRun       bool
}

func main() {
	var cfg Config
	flag.StringVar(&cfg.PlaylistGlob, "playlist", "", "playlist glob pattern (required)")
	flag.StringVar(&cfg.USBRoot, "usbRoot", "", "USB root directory (required)")
	flag.BoolVar(&cfg.DryRun, "dryrun", false, "dry run mode (no actual sync)")
	flag.Parse()

	if err := validateArgs(cfg); err != nil {
		log.Fatal(err)
	}

	unique, perPlaylistLines, err := collectUniqueFiles(cfg)
	if err != nil {
		log.Fatal(err)
	}

	tmpDir, err := createSymlinks(unique)
	if err != nil {
		log.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := syncWithRsync(cfg, tmpDir); err != nil {
		log.Printf("rsync error: %v", err)
	}

	if err := cleanupOldPlaylists(cfg, perPlaylistLines); err != nil {
		log.Printf("cleanup error: %v", err)
	}

	writeAllPlaylists(cfg, perPlaylistLines)
}

func validateArgs(cfg Config) error {
	if cfg.PlaylistGlob == "" {
		return fmt.Errorf("playlist glob required (e.g. --playlist \"~/Playlists/*.m3u8\")")
	}
	if cfg.USBRoot == "" {
		return fmt.Errorf("usbRoot required (e.g. --usbRoot /Volumes/UNTITLED)")
	}
	return nil
}

func collectUniqueFiles(cfg Config) (map[string]struct{}, map[string][]string, error) {
	playlists, err := filepath.Glob(cfg.PlaylistGlob)
	if err != nil || len(playlists) == 0 {
		return nil, nil, fmt.Errorf("no playlists matched: %s", cfg.PlaylistGlob)
	}

	unique := map[string]struct{}{}
	perPlaylistLines := map[string][]string{}

	for _, playlist := range playlists {
		lines, tracks, err := parsePlaylistFile(playlist)
		perPlaylistLines[playlist] = lines
		if err != nil {
			log.Printf("warning: parsePlaylistFile %s: %v (will still attempt to write converted playlist)", playlist, err)
		}
		for _, track := range tracks {
			unique[filepath.Clean(track)] = struct{}{}
		}
	}
	return unique, perPlaylistLines, nil
}

func parsePlaylistFile(path string) ([]string, []string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	text := strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(string(content))
	allLines := strings.Split(text, "\n")

	var lines []string
	var tracks []string

	for len(allLines) > 0 {
		line := strings.TrimSpace(allLines[0])
		allLines = allLines[1:] // 処理した行を削除

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#EXTINF") {
			lines = append(lines, line)
			if len(allLines) == 0 {
				break
			}
			filePath := filepath.Clean(cleanPath(allLines[0]))
			allLines = allLines[1:] // ファイルパス行も削除
			lines = append(lines, filePath)
			if filepath.IsAbs(filePath) {
				tracks = append(tracks, filePath)
			}
			continue
		}

		lines = append(lines, line)
		if !strings.HasPrefix(line, "#") && filepath.IsAbs(line) {
			tracks = append(tracks, cleanPath(line))
		}
	}

	return lines, tracks, nil
}

func cleanPath(path string) string {
	// Remove any non-printable characters and normalize
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1 // Remove control characters
		}
		return r
	}, strings.TrimSpace(path)))
}

func createSymlinks(unique map[string]struct{}) (string, error) {
	tmpDir, err := os.MkdirTemp("", tempDirPrefix)
	if err != nil {
		return "", err
	}

	for src := range unique {
		if _, err := os.Stat(src); err != nil {
			log.Printf("src stat error %s: %v", src, err)
			continue
		}

		linkPath := filepath.Join(tmpDir, strings.TrimPrefix(src, "/"))

		if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
			log.Printf("error creating dir for %s: %v", linkPath, err)
			continue
		}

		if err := os.Symlink(src, linkPath); err != nil {
			log.Printf("error creating symlink %s -> %s: %v", linkPath, src, err)
			continue
		}
	}
	return tmpDir, nil
}

func syncWithRsync(cfg Config, srcDir string) error {
	musicDir := getMusicDir(cfg)
	if err := os.MkdirAll(musicDir, 0755); err != nil {
		return err
	}

	args := buildRsyncArgs(cfg.DryRun, srcDir, musicDir)
	cmd := exec.Command("rsync", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func buildRsyncArgs(dryRun bool, srcDir, destDir string) []string {
	args := []string{"-avL", "--progress", "--delete"}
	if dryRun {
		args = append(args, "--dry-run")
	}
	return append(args, srcDir+"/", destDir+"/")
}

func cleanupOldPlaylists(cfg Config, perPlaylistLines map[string][]string) error {
	musicDir := getMusicDir(cfg)
	expectedPlaylists := createExpectedPlaylistsMap(musicDir, perPlaylistLines)

	entries, err := os.ReadDir(musicDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".m3u8") {
			continue
		}

		if _, exists := expectedPlaylists[filepath.Join(musicDir, entry.Name())]; exists {
			continue
		}

		playlistPath := filepath.Join(musicDir, entry.Name())
		if cfg.DryRun {
			log.Printf("[DRYRUN] would delete old playlist: %s", playlistPath)
			continue
		}

		if err := os.Remove(playlistPath); err != nil {
			log.Printf("error deleting old playlist %s: %v", playlistPath, err)
		}
	}

	return nil
}

func createExpectedPlaylistsMap(musicDir string, perPlaylistLines map[string][]string) map[string]struct{} {
	expected := make(map[string]struct{}, len(perPlaylistLines))
	for playlist := range perPlaylistLines {
		expected[filepath.Join(musicDir, filepath.Base(playlist))] = struct{}{}
	}
	return expected
}

func getMusicDir(cfg Config) string {
	return filepath.Join(cfg.USBRoot, usbMusicDir)
}

func writeAllPlaylists(cfg Config, perPlaylistLines map[string][]string) {
	for playlist, lines := range perPlaylistLines {
		if err := writeConvertedPlaylist(cfg, playlist, lines); err != nil {
			log.Printf("playlist write error %s: %v", playlist, err)
		}
	}
}

func writeConvertedPlaylist(cfg Config, srcPlaylist string, originalLines []string) error {
	musicDir := getMusicDir(cfg)
	outPath := filepath.Join(musicDir, filepath.Base(srcPlaylist))

	if cfg.DryRun {
		previewPlaylist(outPath, originalLines)
		return nil
	}

	if err := os.MkdirAll(musicDir, 0o755); err != nil {
		return err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	var builder strings.Builder
	for _, l := range originalLines {
		builder.WriteString(formatPlaylistLine(l) + "\n")
	}

	_, err = f.WriteString(builder.String())
	return err
}

func formatPlaylistLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	} else if strings.HasPrefix(trimmed, "#") {
		return trimmed
	} else {
		return "./" + strings.TrimPrefix(trimmed, "/")
	}
}

func previewPlaylist(outPath string, originalLines []string) {
	log.Printf("[DRYRUN] would write playlist: %s (preview up to %d entries)", outPath, maxPreviewEntries)
	count := 0
	for _, l := range originalLines {
		line := formatPlaylistLine(l)
		log.Printf("  %s", line)

		// Only count non-empty lines for the limit
		if line != "" {
			count++
			if count >= maxPreviewEntries {
				break
			}
		}
	}
}
