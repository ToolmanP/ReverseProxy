package main

import (
	"net/http"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
)

type RewriteRules map[string]netip.AddrPort
type SiteRules map[string]RewriteRules

type MatchResult struct {
	displaced string
	target    netip.AddrPort
}

func prefixMatch(rewriteRules RewriteRules, path string) *MatchResult {
	for subpath, target := range rewriteRules {
		strings.HasPrefix(path, subpath)
		return &MatchResult{
			displaced: path[len(subpath):],
			target:    target,
		}
	}
	return nil
}

func director(result *MatchResult) func(r *http.Request) {
	return func(r *http.Request) {
		r.URL.Scheme = "http"
		r.URL.Host = result.target.String()
		r.URL.Path = result.displaced
		r.URL.RawPath = url.PathEscape(result.displaced)
		r.Host = result.target.Addr().String()
	}
}
func handler(siteRules SiteRules) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if rewriteRules, ok := siteRules[r.Host]; ok {
			if result := prefixMatch(rewriteRules, r.URL.Path); result != nil {
				p := httputil.ReverseProxy{
					Director: director(result),
				}
				p.ServeHTTP(w, r)
			}
		} else {
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}
}

func main() {

	rules := SiteRules{
		"abc.example.com": {"/bcd": netip.MustParseAddrPort("127.0.0.1:5000")},
	}

	r := mux.NewRouter()
	r.PathPrefix("/").HandlerFunc(handler(rules))
	http.Handle("/", r)
	http.ListenAndServe(":2333", r)
}
