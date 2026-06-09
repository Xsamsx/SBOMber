package health

import "testing"

func TestExtractGitHubURLFromPOM(t *testing.T) {
	pom := `<project>
  <scm>
    <url>https://github.com/spring-projects/spring-framework</url>
  </scm>
</project>`

	got := extractGitHubURLFromPOM(pom)
	want := "https://github.com/spring-projects/spring-framework"
	if got != want {
		t.Fatalf("extractGitHubURLFromPOM() = %q, want %q", got, want)
	}
}

func TestMavenPOMURL(t *testing.T) {
	got := mavenPOMURL("org.springframework", "spring-core", "5.3.20")
	want := "https://repo1.maven.org/maven2/org/springframework/spring-core/5.3.20/spring-core-5.3.20.pom"
	if got != want {
		t.Fatalf("mavenPOMURL() = %q, want %q", got, want)
	}
}
