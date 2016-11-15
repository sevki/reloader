package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"runtime"

	"os"
	"os/exec"
	"path/filepath"
	"time"
)

var (
	start = time.Now()
)

func doe(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func status(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK\n"))
	w.Write([]byte(fmt.Sprintf("go:\t%s\n", runtime.Version())))
	w.Write([]byte(fmt.Sprintf("uptime:\t%s\n", time.Since(start).String())))

}

func main() {
	cmd := logCmd("apachectl", "-DFOREGROUND")
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	go func() {
		for {
			resp, err := http.Get(os.Getenv("REPOS_URL"))
			doe(err)
			type repo struct {
				Name, Url string
			}
			dec := json.NewDecoder(resp.Body)
			defer resp.Body.Close()
			for {
				var m repo
				if err := dec.Decode(&m); err == io.EOF {
					break
				} else if err != nil {
					log.Fatal(err)
				}
				if err := checkout(m.Url, path.Join("/git", m.Name)); err != nil {
					log.Fatal(err)
				}
				time.Sleep(40 * time.Second)
			}

		}
	}()

	u, err := url.Parse("http://localhost")
	doe(err)
	cgit := httputil.NewSingleHostReverseProxy(u)

	http.Handle("/", AddPrefix("cgit.cgi", cgit))

	http.HandleFunc("/_status", status)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
func AddPrefix(prefix string, h http.Handler) http.Handler {
	if prefix == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL.Path )
		r.URL.Path = path.Join(prefix, r.URL.Path)
		h.ServeHTTP(w, r)
	})
}
func checkout(repo, path string) error {
	// Clone git repo if it doesn't exist.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := runErr(logCmd("git", "clone", "--mirror", repo, path)); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Pull down changes and update to hash.
	log.Print(path)
	cmd := logCmd("git", "fetch", "--all")
	cmd.Dir = path
	return runErr(cmd)
}

func runErr(cmd *exec.Cmd) error {
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) == 0 {
			return err
		}
		return fmt.Errorf("%s\n%v", out, err)
	}
	return nil
}
func logCmd(s ...string) *exec.Cmd {
	log.Println(s)
	return exec.Command(s[0], s[1:]...)
}