//go:build contract

package objects_test

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	ct "github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/contracttest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

type objectAPI struct {
	mu            sync.Mutex
	t             *testing.T
	content       []byte
	contentType   string
	exists        bool
	puts, deletes int
}

func (a *objectAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if r.URL.Path == "/contract-bucket/" && r.Method == "GET" && r.URL.Query().Has("location") {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, `<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">us-east-1</LocationConstraint>`)
		return
	}
	if r.URL.Path != "/contract-bucket/config.txt" {
		a.t.Errorf("unexpected S3 request: %s %s", r.Method, r.URL)
		w.WriteHeader(400)
		return
	}
	if r.Method == "GET" && r.URL.Query().Has("retention") {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, `<Retention xmlns="http://s3.amazonaws.com/doc/2006-03-01/"></Retention>`)
		return
	}
	switch r.Method {
	case "PUT":
		body, err := io.ReadAll(r.Body)
		if err != nil {
			a.t.Error(err)
			w.WriteHeader(400)
			return
		}
		a.content = body
		a.contentType = r.Header.Get("Content-Type")
		a.exists = true
		a.puts++
		w.Header().Set("ETag", fmt.Sprintf(`"%x"`, md5.Sum(body)))
	case "HEAD":
		if !a.exists {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", a.contentType)
		w.Header().Set("Content-Length", fmt.Sprint(len(a.content)))
		w.Header().Set("ETag", fmt.Sprintf(`"%x"`, md5.Sum(a.content)))
		w.Header().Set("Last-Modified", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Format(http.TimeFormat))
	case "DELETE":
		a.exists = false
		a.deletes++
		w.WriteHeader(204)
	default:
		a.t.Errorf("unexpected S3 request: %s %s", r.Method, r.URL)
		w.WriteHeader(400)
	}
}

func objectConfig(endpoint, content string) string {
	config := strings.Replace(ct.ProviderConfig(endpoint), fmt.Sprintf("object_storage = %q", endpoint), `object_storage = "https://br-se1.magaluobjects.com"`, 1)
	return config + fmt.Sprintf(`resource "mgc_object_storage_objects" "test" {
  bucket = "contract-bucket"
  key = "config.txt"
  content = %q
}`, content)
}

func (a *objectAPI) checkContent(want string, puts int) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		a.mu.Lock()
		defer a.mu.Unlock()
		if string(a.content) != want || a.puts != puts || a.deletes != 0 {
			return fmt.Errorf("remote object: content=%q puts=%d deletes=%d; want %q, %d puts, no deletion", a.content, a.puts, a.deletes, want, puts)
		}
		return nil
	}
}

func s3Server(t *testing.T, handler http.Handler) *httptest.Server {
	s := ct.Server(t, handler)
	// SDK v1.19.0 accepts only fixed S3 hosts. Alias redirects before DNS.
	ct.Alias(t, "https://br-se1.magaluobjects.com", s)
	return s
}
