package repl

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/kakkky/gonsole/errs"
)

const VERSION = "v1.3"

type relesasesInfoResponse struct {
	LatestVersion string `json:"tag_name"`
}

type latestVersionCache struct {
	LastChecked   time.Time `json:"last_checked"`
	LatestVersion string    `json:"latest_version"`
}

func isLatestVersion() (bool, string, error) {
	cache, err := getLatestVersionCache()
	if err != nil {
		return false, "", err
	}
	// キャッシュが有効であれば、キャッシュにあるlatest_versionと比較する
	if time.Since(cache.LastChecked) < 24*time.Hour {
		return cache.LatestVersion == VERSION, cache.LatestVersion, nil
	}
	// キャッシュが無効な場合、最新バージョンを取得
	latestVersion, err := fetchLatestVersion()
	if err != nil {
		return false, "", err
	}
	// キャッシュを更新
	if err := updateVersionCache(cache, latestVersion); err != nil {
		return false, "", err
	}
	return latestVersion == VERSION, latestVersion, nil
}

func getLatestVersionCache() (*latestVersionCache, error) {
	cacheFilePath, err := getLatestVersionCacheFilePath()
	if err != nil {
		return nil, err
	}
	cacheFile, err := os.ReadFile(cacheFilePath)
	if err != nil {
		// ファイルが存在しない場合、初期ファイルを作成する
		if os.IsNotExist(err) {
			initialCache := &latestVersionCache{
				LastChecked:   time.Now(),
				LatestVersion: VERSION,
			}
			json, err := json.MarshalIndent(initialCache, "", "    ")
			if err != nil {
				return nil, errs.NewInternalError("failed to marshal initial version cache").Wrap(err)
			}
			if err := os.WriteFile(cacheFilePath, json, 0644); err != nil {
				return nil, errs.NewInternalError("failed to write initial version cache").Wrap(err)
			}
			// 初期キャッシュを返す
			return initialCache, nil
		}
		// その他の読み取りエラー
		return nil, errs.NewInternalError("failed to read version cache").Wrap(err)
	}

	var cache latestVersionCache
	if err := json.Unmarshal(cacheFile, &cache); err != nil {
		return nil, errs.NewInternalError("failed to unmarshal version cache").Wrap(err)
	}
	return &cache, nil
}

func updateVersionCache(cache *latestVersionCache, latestVersion string) error {
	cacheFilePath, err := getLatestVersionCacheFilePath()
	if err != nil {
		return err
	}

	cache.LastChecked = time.Now()
	cache.LatestVersion = latestVersion
	json, err := json.MarshalIndent(cache, "", "    ")
	if err != nil {
		return errs.NewInternalError("failed to marshal version cache").Wrap(err)
	}
	if err := os.WriteFile(cacheFilePath, json, 0644); err != nil {
		return errs.NewInternalError("failed to write version cache").Wrap(err)
	}
	return nil
}

func fetchLatestVersion() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/kakkky/gonsole/releases/latest")
	if err != nil {
		return "", errs.NewInternalError("failed to fetch latest release").Wrap(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errs.NewInternalError("failed to read response body").Wrap(err)
	}
	var releasesInfo relesasesInfoResponse
	if err := json.Unmarshal(body, &releasesInfo); err != nil {
		return "", errs.NewInternalError("failed to unmarshal response body").Wrap(err)
	}

	return releasesInfo.LatestVersion, nil
}

// getLatestVersionCacheFilePath returns the path to version.json in the user's config directory.
func getLatestVersionCacheFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", errs.NewInternalError("failed to get user config directory").Wrap(err)
	}
	gonsoleConfigDir := filepath.Join(configDir, "gonsole")
	if err := os.MkdirAll(gonsoleConfigDir, 0755); err != nil {
		return "", errs.NewInternalError("failed to create gonsole config directory").Wrap(err)
	}
	return filepath.Join(gonsoleConfigDir, "version.json"), nil
}

func printNoteLatestVersion(latestVersion string) {
	fmt.Println("┌─────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 💡NOTE                                   			  │")
	fmt.Println("│                                     			          │")
	fmt.Printf("│  🚀 New version available! Gonsole %s (you have %s)         │\n", latestVersion, VERSION)
	fmt.Println("│                                                                 │")
	fmt.Println("│     Please Update with:                                    	  │")
	fmt.Println("│                                                                 │")
	fmt.Println("│       go install github.com/kakkky/gonsole/cmd/gonsole@latest   │")
	fmt.Println("│                                                                 │")
	fmt.Println("└─────────────────────────────────────────────────────────────────┘")
}
