package envresolve

import (
	"strings"
	"testing"
)

func TestResolve_NoTemplates(t *testing.T) {
	appVars := []AppVar{
		{Key: "HOST", Value: "localhost"},
		{Key: "PORT", Value: "8080"},
	}

	result, err := Resolve(appVars, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["HOST"] != "localhost" {
		t.Errorf("HOST: got %q, want %q", result["HOST"], "localhost")
	}
	if result["PORT"] != "8080" {
		t.Errorf("PORT: got %q, want %q", result["PORT"], "8080")
	}
}

func TestResolve_SelfReference(t *testing.T) {
	appVars := []AppVar{
		{Key: "HOST", Value: "localhost"},
		{Key: "PORT", Value: "8080"},
		{Key: "URL", Value: "http://${{ HOST }}:${{ PORT }}"},
	}

	result, err := Resolve(appVars, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "http://localhost:8080"
	if result["URL"] != want {
		t.Errorf("URL: got %q, want %q", result["URL"], want)
	}
}

func TestResolve_SharedReference(t *testing.T) {
	appVars := []AppVar{
		{Key: "DB_URL", Value: "${{ shared.DATABASE_URL }}"},
	}
	sharedVars := []AppVar{
		{Key: "DATABASE_URL", Value: "postgres://shared-host:5432/mydb"},
	}

	result, err := Resolve(appVars, sharedVars, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "postgres://shared-host:5432/mydb"
	if result["DB_URL"] != want {
		t.Errorf("DB_URL: got %q, want %q", result["DB_URL"], want)
	}
}

func TestResolve_SiblingAppReference(t *testing.T) {
	appVars := []AppVar{
		{Key: "API_KEY", Value: "${{ auth-service.SECRET_KEY }}"},
	}
	siblingVars := []SiblingVar{
		{AppSlug: "auth-service", Key: "SECRET_KEY", Value: "super-secret-123"},
	}

	result, err := Resolve(appVars, nil, siblingVars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "super-secret-123"
	if result["API_KEY"] != want {
		t.Errorf("API_KEY: got %q, want %q", result["API_KEY"], want)
	}
}

func TestResolve_MultipleTemplatesInOneValue(t *testing.T) {
	appVars := []AppVar{
		{Key: "HOST", Value: "myhost"},
		{Key: "CONNECTION", Value: "postgres://${{ shared.DB_USER }}:${{ shared.DB_PASS }}@${{ HOST }}/db"},
	}
	sharedVars := []AppVar{
		{Key: "DB_USER", Value: "admin"},
		{Key: "DB_PASS", Value: "p@ssw0rd"},
	}

	result, err := Resolve(appVars, sharedVars, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "postgres://admin:p@ssw0rd@myhost/db"
	if result["CONNECTION"] != want {
		t.Errorf("CONNECTION: got %q, want %q", result["CONNECTION"], want)
	}
}

func TestResolve_MissingSelfReference(t *testing.T) {
	appVars := []AppVar{
		{Key: "URL", Value: "${{ MISSING_VAR }}"},
	}

	_, err := Resolve(appVars, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing self-reference, got nil")
	}

	if !strings.Contains(err.Error(), "MISSING_VAR") {
		t.Errorf("error should mention MISSING_VAR, got: %v", err)
	}
}

func TestResolve_MissingSharedReference(t *testing.T) {
	appVars := []AppVar{
		{Key: "SECRET", Value: "${{ shared.NONEXISTENT }}"},
	}

	_, err := Resolve(appVars, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing shared reference, got nil")
	}

	if !strings.Contains(err.Error(), "NONEXISTENT") {
		t.Errorf("error should mention NONEXISTENT, got: %v", err)
	}
}

func TestResolve_MissingSiblingReference(t *testing.T) {
	appVars := []AppVar{
		{Key: "KEY", Value: "${{ other-app.MISSING }}"},
	}

	_, err := Resolve(appVars, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing sibling reference, got nil")
	}

	if !strings.Contains(err.Error(), "other-app") {
		t.Errorf("error should mention other-app, got: %v", err)
	}
	if !strings.Contains(err.Error(), "MISSING") {
		t.Errorf("error should mention MISSING, got: %v", err)
	}
}

func TestResolve_EmptyInputs(t *testing.T) {
	result, err := Resolve(nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}

	result, err = Resolve([]AppVar{}, []AppVar{}, []SiblingVar{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}
}

func TestResolve_WhitespaceInTemplate(t *testing.T) {
	appVars := []AppVar{
		{Key: "HOST", Value: "localhost"},
		{Key: "URL", Value: "${{   HOST   }}"},
	}

	result, err := Resolve(appVars, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["URL"] != "localhost" {
		t.Errorf("URL: got %q, want %q", result["URL"], "localhost")
	}
}

func TestResolve_MixedScopes(t *testing.T) {
	appVars := []AppVar{
		{Key: "PORT", Value: "3000"},
		{Key: "FULL_URL", Value: "${{ shared.DOMAIN }}:${{ PORT }}/api/${{ gateway.API_VERSION }}"},
	}
	sharedVars := []AppVar{
		{Key: "DOMAIN", Value: "https://example.com"},
	}
	siblingVars := []SiblingVar{
		{AppSlug: "gateway", Key: "API_VERSION", Value: "v2"},
	}

	result, err := Resolve(appVars, sharedVars, siblingVars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "https://example.com:3000/api/v2"
	if result["FULL_URL"] != want {
		t.Errorf("FULL_URL: got %q, want %q", result["FULL_URL"], want)
	}
}
