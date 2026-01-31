package version

import (
	// go:embedディレクティブ用
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/kakkky/gonsole/errs"
)

// VERSION は現在のgonsoleのバージョンを表す
const VERSION = "v1.4"

// PrintVersion は現在のgonsoleのバージョンを表示する
func PrintVersion() {
	fmt.Println("   " + VERSION)
}

type relesasesInfoResponse struct {
	LatestVersion string `json:"tag_name"`
}

// IsLatestVersion は現在のgonsoleのバージョンが最新かどうかを判定する
func IsLatestVersion() (bool, string, error) {
	latestVersion, err := fetchLatestVersion()
	if err != nil {
		return false, "", err
	}
	return latestVersion == VERSION, latestVersion, nil
}

func fetchLatestVersion() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/kakkky/gonsole/releases/latest")
	if err != nil {
		return "", errs.NewInternalError("failed to fetch latest release").Wrap(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			errs.HandleError(err)
		}
	}()

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

//go:embed latest_ver_note_ascii.txt
var latestVerNoteASCII []byte

// PrintNoteLatestVersion は最新バージョンが存在する場合の通知を表示する
func PrintNoteLatestVersion(latestVersion string) {
	fmt.Printf(string(latestVerNoteASCII), latestVersion, VERSION)
}
