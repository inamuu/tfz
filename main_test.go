package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestQDoesNotQuitWhileFiltering(t *testing.T) {
	m := model{
		step:    stepTargets,
		targets: []targetItem{{Label: allTarget}, {Label: "resource.aws_instance.example"}},
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd != nil {
		t.Fatal("q should not quit the program")
	}

	got := updated.(model)
	if got.filter != "q" {
		t.Fatalf("expected filter to contain q, got %q", got.filter)
	}
}

func TestEscQuits(t *testing.T) {
	m := model{
		step:    stepTargets,
		targets: []targetItem{{Label: allTarget}, {Label: "resource.aws_instance.example"}},
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc should quit the program")
	}
}

func TestTargetsForLinesFindsEnclosingBlocks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.tf")
	content := `resource "aws_instance" "example" {
  ami = "ami-123"
  instance_type = "t3.micro"
}

module "network" {
  source = "./modules/network"
}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write tf file: %v", err)
	}

	targets, err := targetsForLines(path, map[int]struct{}{
		3: {},
		6: {},
	})
	if err != nil {
		t.Fatalf("targetsForLines: %v", err)
	}

	want := []string{
		"module.network",
		"resource.aws_instance.example",
	}
	if !reflect.DeepEqual(targets, want) {
		t.Fatalf("unexpected targets: got %#v want %#v", targets, want)
	}
}

func TestParseGitDiffIncludesChangedAndDeletedTargets(t *testing.T) {
	diff := `diff --git a/main.tf b/main.tf
index 1111111..2222222 100644
--- a/main.tf
+++ b/main.tf
@@ -2 +2 @@ resource "aws_instance" "example" {
-  instance_type = "t3.micro"
+  instance_type = "t3.small"
@@ -10,4 +9,0 @@ module "obsolete" {
-module "obsolete" {
-  source = "./modules/obsolete"
-}
-
`

	changed, explicit, err := parseGitDiff(diff)
	if err != nil {
		t.Fatalf("parseGitDiff: %v", err)
	}

	if _, ok := changed["main.tf"][2]; !ok {
		t.Fatalf("expected changed line 2 for main.tf, got %#v", changed["main.tf"])
	}
	if _, ok := explicit["module.obsolete"]; !ok {
		t.Fatalf("expected deleted module target, got %#v", explicit)
	}
}

func TestParseGitDiffTracksNestedTerraformFiles(t *testing.T) {
	diff := `diff --git a/terraform/jm-maskdata/main.tf b/terraform/jm-maskdata/main.tf
index e2236ca53..f17cb98cf 100644
--- a/terraform/jm-maskdata/main.tf
+++ b/terraform/jm-maskdata/main.tf
@@ -754,1 +754,1 @@ resource "aws_scheduler_schedule" "db_restore" {
-    mode = "OFF"
+    mode = "ON"
`

	changed, explicit, err := parseGitDiff(diff)
	if err != nil {
		t.Fatalf("parseGitDiff: %v", err)
	}

	if _, ok := changed["terraform/jm-maskdata/main.tf"][754]; !ok {
		t.Fatalf("expected changed line 754 for nested tf file, got %#v", changed)
	}
	if len(explicit) != 0 {
		t.Fatalf("expected no explicit deleted targets, got %#v", explicit)
	}
}
