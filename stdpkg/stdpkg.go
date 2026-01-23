package stdpkg

import "github.com/kakkky/gonsole/types"

// IsStandardPackage は与えられたパッケージ名が標準パッケージかどうかを判定し、
// 標準パッケージであればそのインポートパスを返す
func IsStandardPackage(pkgName types.PkgName) (types.ImportPath, bool) {
	// 各カテゴリの標準パッケージをチェック
	if importPath, found := getCorePackages()[pkgName]; found {
		return importPath, true
	}
	if importPath, found := getNetworkPackages()[pkgName]; found {
		return importPath, true
	}
	if importPath, found := getEncodingPackages()[pkgName]; found {
		return importPath, true
	}
	if importPath, found := getCryptoPackages()[pkgName]; found {
		return importPath, true
	}
	if importPath, found := getIOPackages()[pkgName]; found {
		return importPath, true
	}
	if importPath, found := getMathPackages()[pkgName]; found {
		return importPath, true
	}
	if importPath, found := getTestingPackages()[pkgName]; found {
		return importPath, true
	}
	if importPath, found := getSystemPackages()[pkgName]; found {
		return importPath, true
	}
	if importPath, found := getUtilityPackages()[pkgName]; found {
		return importPath, true
	}
	return "", false
}

// getCorePackages は基本的なパッケージのマップを返す
func getCorePackages() map[types.PkgName]types.ImportPath {
	return map[types.PkgName]types.ImportPath{
		"fmt":     `"fmt"`,
		"errors":  `"errors"`,
		"context": `"context"`,
		"sort":    `"sort"`,
		"reflect": `"reflect"`,
		"unsafe":  `"unsafe"`,
		"embed":   `"embed"`,
	}
}

// getIOPackages はIO関連パッケージのマップを返す
func getIOPackages() map[types.PkgName]types.ImportPath {
	return map[types.PkgName]types.ImportPath{
		"io":     `"io"`,
		"ioutil": `"io/ioutil"`,
		"fs":     `"io/fs"`,
		"bufio":  `"bufio"`,
		"bytes":  `"bytes"`,
		"os":     `"os"`,
	}
}

// getNetworkPackages はネットワーク関連パッケージのマップを返す
func getNetworkPackages() map[types.PkgName]types.ImportPath {
	return map[types.PkgName]types.ImportPath{
		"http":      `"net/http"`,
		"url":       `"net/url"`,
		"mail":      `"net/mail"`,
		"rpc":       `"net/rpc"`,
		"smtp":      `"net/smtp"`,
		"textproto": `"net/textproto"`,
		"httputil":  `"net/http/httputil"`,
		"httptrace": `"net/http/httptrace"`,
		"httptest":  `"net/http/httptest"`,
		"cookiejar": `"net/http/cookiejar"`,
		"fcgi":      `"net/http/fcgi"`,
		"pprof":     `"net/http/pprof"`,
		"jsonrpc":   `"net/rpc/jsonrpc"`,
	}
}

// getEncodingPackages はエンコーディング関連パッケージのマップを返す
func getEncodingPackages() map[types.PkgName]types.ImportPath {
	return map[types.PkgName]types.ImportPath{
		"json":    `"encoding/json"`,
		"xml":     `"encoding/xml"`,
		"csv":     `"encoding/csv"`,
		"base64":  `"encoding/base64"`,
		"base32":  `"encoding/base32"`,
		"hex":     `"encoding/hex"`,
		"ascii85": `"encoding/ascii85"`,
		"binary":  `"encoding/binary"`,
		"gob":     `"encoding/gob"`,
		"pem":     `"encoding/pem"`,
		"asn1":    `"encoding/asn1"`,
	}
}

// getCryptoPackages は暗号化関連パッケージのマップを返す
func getCryptoPackages() map[types.PkgName]types.ImportPath {
	return map[types.PkgName]types.ImportPath{
		"crypto":   `"crypto"`,
		"md5":      `"crypto/md5"`,
		"sha1":     `"crypto/sha1"`,
		"sha256":   `"crypto/sha256"`,
		"sha512":   `"crypto/sha512"`,
		"aes":      `"crypto/aes"`,
		"cipher":   `"crypto/cipher"`,
		"des":      `"crypto/des"`,
		"dsa":      `"crypto/dsa"`,
		"ecdsa":    `"crypto/ecdsa"`,
		"ed25519":  `"crypto/ed25519"`,
		"elliptic": `"crypto/elliptic"`,
		"hmac":     `"crypto/hmac"`,
		"rc4":      `"crypto/rc4"`,
		"rsa":      `"crypto/rsa"`,
		"subtle":   `"crypto/subtle"`,
		"tls":      `"crypto/tls"`,
		"x509":     `"crypto/x509"`,
		"hash":     `"hash"`,
		"adler32":  `"hash/adler32"`,
		"crc32":    `"hash/crc32"`,
		"crc64":    `"hash/crc64"`,
		"fnv":      `"hash/fnv"`,
		"maphash":  `"hash/maphash"`,
	}
}

// getMathPackages は数学関連パッケージのマップを返す
func getMathPackages() map[types.PkgName]types.ImportPath {
	return map[types.PkgName]types.ImportPath{
		"math":  `"math"`,
		"rand":  `"math/rand"`,
		"big":   `"math/big"`,
		"bits":  `"math/bits"`,
		"cmplx": `"math/cmplx"`,
	}
}

// getTestingPackages はテスト関連パッケージのマップを返す
func getTestingPackages() map[types.PkgName]types.ImportPath {
	return map[types.PkgName]types.ImportPath{
		"testing": `"testing"`,
		"quick":   `"testing/quick"`,
		"iotest":  `"testing/iotest"`,
		"fstest":  `"testing/fstest"`,
	}
}

// getSystemPackages はシステム関連パッケージのマップを返す
func getSystemPackages() map[types.PkgName]types.ImportPath {
	return map[types.PkgName]types.ImportPath{
		"runtime": `"runtime"`,
		"cgo":     `"runtime/cgo"`,
		"debug":   `"runtime/debug"`,
		"metrics": `"runtime/metrics"`,
		"race":    `"runtime/race"`,
		"trace":   `"runtime/trace"`,
		"syscall": `"syscall"`,
		"plugin":  `"plugin"`,
		"sync":    `"sync"`,
		"atomic":  `"sync/atomic"`,
	}
}

// getUtilityPackages はユーティリティ関連パッケージのマップを返す
func getUtilityPackages() map[types.PkgName]types.ImportPath {
	return map[types.PkgName]types.ImportPath{
		// 文字列・テキスト処理
		"strings":   `"strings"`,
		"strconv":   `"strconv"`,
		"regexp":    `"regexp"`,
		"scanner":   `"text/scanner"`,
		"template":  `"text/template"`,
		"tabwriter": `"text/tabwriter"`,

		// パス・ファイル操作
		"pkgName":     `"pkgName"`,
		"filepkgName": `"pkgName/filepkgName"`,

		// 時間
		"time": `"time"`,

		// コレクション・データ構造
		"slices":    `"slices"`,
		"container": `"container"`,
		"heap":      `"container/heap"`,
		"list":      `"container/list"`,
		"ring":      `"container/ring"`,

		// 圧縮・アーカイブ
		"compress": `"compress"`,
		"bzip2":    `"compress/bzip2"`,
		"flate":    `"compress/flate"`,
		"gzip":     `"compress/gzip"`,
		"lzw":      `"compress/lzw"`,
		"zlib":     `"compress/zlib"`,
		"archive":  `"archive"`,
		"tar":      `"archive/tar"`,
		"zip":      `"archive/zip"`,

		// データベース
		"sql":    `"database/sql"`,
		"driver": `"database/sql/driver"`,

		// イメージ処理
		"image": `"image"`,
		"color": `"image/color"`,
		"draw":  `"image/draw"`,
		"gif":   `"image/gif"`,
		"jpeg":  `"image/jpeg"`,
		"png":   `"image/png"`,

		// Go言語関連
		"go":       `"go"`,
		"ast":      `"go/ast"`,
		"build":    `"go/build"`,
		"constant": `"go/constant"`,
		"doc":      `"go/doc"`,
		"format":   `"go/format"`,
		"importer": `"go/importer"`,
		"parser":   `"go/parser"`,
		"printer":  `"go/printer"`,
		"token":    `"go/token"`,
		"types":    `"go/types"`,

		// HTML・MIME
		"html":            `"html"`,
		"mime":            `"mime"`,
		"multipart":       `"mime/multipart"`,
		"quotedprintable": `"mime/quotedprintable"`,

		// Unicode
		"unicode": `"unicode"`,
		"utf8":    `"unicode/utf8"`,
		"utf16":   `"unicode/utf16"`,
	}
}
