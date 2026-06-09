package remote

import (
	"testing"

	"github.com/Xsamsx/SBOMber/internal/deps"
)

func TestParsePackageJSON(t *testing.T) {
	content := []byte(`{
		"dependencies": {
			"express": "^4.18.2",
			"lodash": "4.17.21"
		},
		"devDependencies": {
			"jest": "^29.0.0"
		}
	}`)

	summary, err := parsePackageJSON(content)
	if err != nil {
		t.Fatalf("parsePackageJSON failed: %v", err)
	}

	if len(summary.Direct) != 3 {
		t.Errorf("expected 3 dependencies, got %d", len(summary.Direct))
	}

	found := make(map[string]deps.Dependency)
	for _, dep := range summary.Direct {
		found[dep.Name] = dep
	}

	if dep, ok := found["express"]; !ok {
		t.Error("missing express dependency")
	} else if dep.Version != "4.18.2" {
		t.Errorf("express version = %s, want 4.18.2", dep.Version)
	} else if dep.Ecosystem != "npm" {
		t.Errorf("express ecosystem = %s, want npm", dep.Ecosystem)
	}

	if dep, ok := found["jest"]; !ok {
		t.Error("missing jest dependency")
	} else if dep.Scope != deps.ScopeDev {
		t.Errorf("jest scope = %s, want development", dep.Scope)
	}
}

func TestParseRequirementsTxt(t *testing.T) {
	content := []byte(`
# Comment line
requests==2.28.1
flask>=2.0.0
django[argon2]>=4.0
-r base.txt
numpy
`)

	summary, err := parseRequirementsTxt(content)
	if err != nil {
		t.Fatalf("parseRequirementsTxt failed: %v", err)
	}

	if len(summary.Direct) != 4 {
		t.Errorf("expected 4 dependencies, got %d", len(summary.Direct))
	}

	found := make(map[string]deps.Dependency)
	for _, dep := range summary.Direct {
		found[dep.Name] = dep
	}

	if dep, ok := found["requests"]; !ok {
		t.Error("missing requests dependency")
	} else if dep.Version != "==2.28.1" {
		t.Errorf("requests version = %s, want ==2.28.1", dep.Version)
	}

	if _, ok := found["django"]; !ok {
		t.Error("missing django dependency")
	}
}

func TestParseGoMod(t *testing.T) {
	content := []byte(`
module example.com/myapp

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/spf13/cobra v1.7.0
	golang.org/x/text v0.14.0 // indirect
)

require github.com/single/dep v1.0.0
`)

	summary, err := parseGoMod(content)
	if err != nil {
		t.Fatalf("parseGoMod failed: %v", err)
	}

	if len(summary.Direct) != 3 {
		t.Errorf("expected 3 direct dependencies, got %d", len(summary.Direct))
	}

	if len(summary.Transitive) != 1 {
		t.Errorf("expected 1 transitive dependency, got %d", len(summary.Transitive))
	}

	found := make(map[string]deps.Dependency)
	for _, dep := range summary.Direct {
		found[dep.Name] = dep
	}

	if dep, ok := found["github.com/gin-gonic/gin"]; !ok {
		t.Error("missing gin dependency")
	} else if dep.Version != "v1.9.1" {
		t.Errorf("gin version = %s, want v1.9.1", dep.Version)
	} else if dep.Ecosystem != "golang" {
		t.Errorf("gin ecosystem = %s, want golang", dep.Ecosystem)
	}

	if _, ok := found["golang.org/x/text"]; ok {
		t.Error("indirect dependency should not be listed as direct")
	}

	transitive := make(map[string]deps.Dependency)
	for _, dep := range summary.Transitive {
		transitive[dep.Name] = dep
	}

	if dep, ok := transitive["golang.org/x/text"]; !ok {
		t.Error("missing indirect golang.org/x/text dependency")
	} else if dep.Version != "v0.14.0" {
		t.Errorf("text version = %s, want v0.14.0", dep.Version)
	}
}

func TestParsePomXML(t *testing.T) {
	content := []byte(`
<?xml version="1.0"?>
<project>
	<dependencies>
		<dependency>
			<groupId>org.springframework</groupId>
			<artifactId>spring-core</artifactId>
			<version>5.3.20</version>
		</dependency>
		<dependency>
			<groupId>junit</groupId>
			<artifactId>junit</artifactId>
			<version>4.13.2</version>
		</dependency>
	</dependencies>
</project>
`)

	summary, err := parsePomXML(content)
	if err != nil {
		t.Fatalf("parsePomXML failed: %v", err)
	}

	if len(summary.Direct) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(summary.Direct))
	}

	found := make(map[string]deps.Dependency)
	for _, dep := range summary.Direct {
		found[dep.Name] = dep
	}

	if dep, ok := found["org.springframework:spring-core"]; !ok {
		t.Error("missing spring-core dependency")
	} else if dep.Version != "5.3.20" {
		t.Errorf("spring-core version = %s, want 5.3.20", dep.Version)
	}
}

func TestParseGemfileLock(t *testing.T) {
	content := []byte(`
GEM
  remote: https://rubygems.org/
  specs:
    actioncable (7.0.4)
    activerecord (7.0.4)
    rails (7.0.4)

PLATFORMS
  ruby
`)

	summary, err := parseGemfileLock(content)
	if err != nil {
		t.Fatalf("parseGemfileLock failed: %v", err)
	}

	if len(summary.Direct) != 3 {
		t.Errorf("expected 3 dependencies, got %d", len(summary.Direct))
	}

	found := make(map[string]deps.Dependency)
	for _, dep := range summary.Direct {
		found[dep.Name] = dep
	}

	if dep, ok := found["rails"]; !ok {
		t.Error("missing rails dependency")
	} else if dep.Version != "7.0.4" {
		t.Errorf("rails version = %s, want 7.0.4", dep.Version)
	}
}
