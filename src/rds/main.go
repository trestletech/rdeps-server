package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	//r.Run(":8081")

	rules, err := downloadDB()
	if err != nil {
		log.Fatal(err)
	}

	acts := rules.FindActions(",libcurl,", "linux", "debian", "9", "amd64")
	log.Printf("Acts: %v", acts)

	log.Printf("Rules: %+v", rules)
}

var depRE = regexp.MustCompile(`/deps/.*\.json$`)

type versionConstraint struct {
	MinVer          string `json:"minVer"`
	MinVerExclusive string `json:"minVerExclusive"`
	MaxVer          string `json:"maxVer"`
	MaxVerExclusive string `json:"maxVerExclusive"`
}

type sysConstraint struct {
	OS     string `json:"os"`     // TODO: enum
	Flavor string `json:"flavor"` // TODO: enum
	Arch   string `json:"arch"`
}

type dependency struct {
	Runtime        bool              `json:"runtime"`
	SysConstraints []sysConstraint   `json:"sysConstraints"`
	PkgConstraint  versionConstraint `json:"pkgConstraint"`
	SysPkgs        []string          `json:"sysPkgs"`
	Scripts        []string          `json:"scripts"`
}

type rule struct {
	Description  string       `json:"description"`
	Regexp       string       `json:"regexp"`
	Dependencies []dependency `json:"dependencies"`
}

type ruleset []rule

type action struct {
	SystemPkg []string
	Scripts   []string
}

func (rs ruleset) FindActions(sysreqs, os, flavor, version, arch string) []action {
	acts := make([]action, 0)

	for _, r := range rs {
		// TODO: cache this
		re, err := regexp.Compile("(?i)" + r.Regexp)
		if err != nil {
			log.Printf("Ignoring invalid regular expression for rule: %s", r.Regexp)
			break
		}

		if !re.MatchString(sysreqs) {
			// No match, no need to further evaluate.
			log.Printf("No re match: %s on '%v'", sysreqs, r.Regexp)
			continue
		}

		for _, d := range r.Dependencies {
			for _, sc := range d.SysConstraints {
				if sc.Arch != "" && arch != "" && sc.Arch != arch {
					continue
				}
				if sc.Flavor != "" && flavor != "" && sc.Flavor != flavor {
					continue
				}
				/*if sc.Version != "" && version != "" && sc.Version != version {
					continue
				}*/
				if sc.OS != "" && os != "" && sc.OS != os {
					continue
				}

				// We have a match! Apply it
				act := action{
					d.SysPkgs,
					d.Scripts,
				}
				acts = append(acts, act)

				// We don't need to evaluate any more of the sysConstraints, we already matched.
				break
			}
		}
	}

	return acts
}

func downloadDB() (ruleset, error) {
	url := "https://github.com/trestletech/rdeps/archive/master.tar.gz"
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	deps := make([]rule, 0)

	gz, err := gzip.NewReader(res.Body)
	if err != nil {
		return nil, err
	}

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		if header.Typeflag == tar.TypeReg && depRE.MatchString(header.Name) {
			// dependency file
			dec := json.NewDecoder(tr)
			var dep rule
			dec.Decode(&dep)

			deps = append(deps, dep)
		}
	}

	return deps, nil
}
