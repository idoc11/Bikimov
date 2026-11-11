// Лаунчер «Шторма»: игра вшита в exe, отдаётся с локального адреса
// и открывается в отдельном окне без адресной строки.
package main

import (
	"bytes"
	"embed"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"
)

//go:embed web
var web embed.FS

const addr = "127.0.0.1:47815" // постоянный порт, чтобы сохранения не терялись между запусками
const url = "http://" + addr + "/"

var lastPing atomic.Int64

func main() {
	ln, err := net.Listen("tcp", addr)
	if err != nil { // игра уже запущена — просто показываем её окно
		if c := openWindow(); c != nil {
			c.Wait()
		}
		return
	}
	sub, _ := fs.Sub(web, "web")
	page, _ := fs.ReadFile(sub, "index.html")
	// страница сообщает, что она открыта: по этому сигналу лаунчер понимает, когда пора завершиться
	page = bytes.Replace(page, []byte("</body>"), []byte("<script>setInterval(()=>fetch('/__ping').catch(()=>{}),3000)</script></body>"), 1)
	files := http.FileServer(http.FS(sub))
	mux := http.NewServeMux()
	mux.HandleFunc("/__ping", func(w http.ResponseWriter, r *http.Request) { lastPing.Store(time.Now().Unix()) })
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.Write(page)
			return
		}
		files.ServeHTTP(w, r)
	})
	go http.Serve(ln, mux)

	if c := openWindow(); c != nil {
		began := time.Now()
		c.Wait()
		// браузер мог сразу передать окно уже запущенному процессу и выйти — тогда игра ещё открыта.
		// Если же окно так и не появилось, открываем игру в обычном браузере.
		if lastPing.Load() == 0 && time.Since(began) < 8*time.Second {
			openDefault()
		}
	} else {
		openDefault()
	}
	// работаем, пока страница игры подаёт сигналы; после закрытия окна завершаемся
	start := time.Now()
	for {
		time.Sleep(2 * time.Second)
		p := lastPing.Load()
		if p == 0 && time.Since(start) > 90*time.Second {
			return
		}
		if p != 0 && time.Now().Unix()-p > 12 {
			return
		}
	}
}

// openWindow запускает Edge или Chrome в режиме приложения с отдельным профилем.
func openWindow() *exec.Cmd {
	var paths []string
	if runtime.GOOS == "windows" {
		for _, base := range []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles"), os.Getenv("LocalAppData")} {
			if base == "" {
				continue
			}
			paths = append(paths, filepath.Join(base, "Microsoft", "Edge", "Application", "msedge.exe"), filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe"))
		}
	} else {
		for _, n := range []string{"microsoft-edge", "google-chrome", "chromium", "chromium-browser"} {
			if p, err := exec.LookPath(n); err == nil {
				paths = append(paths, p)
			}
		}
	}
	dir, _ := os.UserCacheDir()
	profile := filepath.Join(dir, "Shtorm", "profile")
	os.MkdirAll(profile, 0o755)
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		c := exec.Command(p, "--app="+url, "--user-data-dir="+profile, "--window-size=1280,760", "--no-first-run", "--no-default-browser-check", "--disable-features=Translate")
		if c.Start() == nil {
			return c
		}
	}
	return nil
}

func openDefault() {
	if runtime.GOOS == "windows" {
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	} else {
		exec.Command("xdg-open", url).Start()
	}
}
