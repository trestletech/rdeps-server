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

	err := downloadDB()
	if err != nil {
		log.Fatal(err)
	}
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

func downloadDB() error {
	url := "https://github.com/trestletech/rdeps/archive/master.tar.gz"
	res, err := http.Get(url)
	if err != nil {
		return err
	}

	deps := make([]rule, 0)

	gz, err := gzip.NewReader(res.Body)
	if err != nil {
		return err
	}

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if header.Typeflag == tar.TypeReg && depRE.MatchString(header.Name) {
			log.Println("\tis a dep!")
			// dependency file
			dec := json.NewDecoder(tr)
			var dep rule
			dec.Decode(&dep)

			deps = append(deps, dep)
		}
	}

	log.Printf("Deps: %+v", deps)
	return nil
}
