package upload

import (
	"net/http"
	"os"
	"regexp"
)

const publicImageCacheControl = "public, max-age=31536000, immutable"

var publicImagePathPattern = regexp.MustCompile(`^/[0-9]{4}/(0[1-9]|1[0-2])/img_[0-9]+_[0-9a-f]{12}\.(gif|jpg|png|webp)$`)

// NewPublicImageHandler serves immutable public uploads without exposing
// directory contents or unrelated files from the upload directory.
func NewPublicImageHandler(uploadDir string) http.Handler {
	fileServer := http.FileServer(publicImageFileSystem{root: http.Dir(uploadDir)})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
		w.Header().Set("Cross-Origin-Resource-Policy", "cross-origin")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		fileServer.ServeHTTP(&publicImageResponseWriter{ResponseWriter: w}, r)
	})
}

type publicImageFileSystem struct {
	root http.FileSystem
}

func (fs publicImageFileSystem) Open(name string) (http.File, error) {
	if !publicImagePathPattern.MatchString(name) {
		return nil, os.ErrNotExist
	}

	file, err := fs.root.Open(name)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, os.ErrNotExist
	}
	return file, nil
}

type publicImageResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *publicImageResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	if (status >= 200 && status < 300) || status == http.StatusNotModified {
		w.Header().Set("Cache-Control", publicImageCacheControl)
	} else {
		w.Header().Set("Cache-Control", "no-store")
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *publicImageResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}
