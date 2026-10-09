package modeluse

import (
	"path/filepath"
	"strings"

	"tgo/internal/outputname"
	"tgo/internal/sourceanalysis"
)

// setGeneratedOutputs records the compiler output for integrity checks.
func (c *checker) setGeneratedOutputs(analysis *sourceanalysis.Package) {
	for _, file := range c.files {
		path := c.pass.Fset.Position(file.Package).Filename
		if outputname.Reserved(path) {
			c.generated[file] = true
		}
	}
	if analysis == nil {
		return
	}
	for _, source := range analysis.Sources {
		c.outputs[source.Name] = source.Output
		for _, file := range c.files {
			path := c.pass.Fset.Position(file.Package).Filename
			if outputname.Matches(source.Name, path) {
				c.generated[file] = true
				c.generatedSource[file] = source.Name
			}
		}
	}
}

// analyzeTGoPackage loads TGo source once for all source checks.
func (c *checker) analyzeTGoPackage() *sourceanalysis.Package {
	directory := c.packageDirectory()
	if directory == "" {
		return nil
	}
	test, external := c.tgoTestPackage()
	path := c.pass.Pkg.Path()
	if external {
		path = strings.TrimSuffix(path, "_test")
	}
	var analysis *sourceanalysis.Package = nil
	var err error = nil
	if test {
		analysis, err = sourceanalysis.AnalyzeTestPackage(
			directory, path, external, c.pass.Fset,
		)
	} else {
		analysis, err = sourceanalysis.AnalyzePackage(
			directory, path, c.pass.Fset,
		)
	}
	if err != nil {
		c.reportResult(c.pass.Files[0].Package, "analyze TGo source: %v", err)
		return nil
	}
	return analysis
}

// tgoTestPackage reports whether the pass contains generated TGo tests.
func (c *checker) tgoTestPackage() (bool, bool) {
	test := false
	production := false
	for _, file := range c.files {
		path := c.pass.Fset.Position(file.Package).Filename
		if !outputname.Reserved(path) {
			continue
		}
		if strings.HasSuffix(path, "_test.go") {
			test = true
		} else {
			production = true
		}
	}
	return test, test && !production
}

// packageDirectory returns the directory that owns the loaded package.
func (c *checker) packageDirectory() string {
	for _, file := range c.pass.Files {
		name := c.pass.Fset.Position(file.Package).Filename
		if name == "" {
			continue
		}
		directory := filepath.Dir(name)
		sources, err := filepath.Glob(filepath.Join(directory, "*.tgo"))
		if err == nil && len(sources) != 0 {
			return directory
		}
	}
	return ""
}
