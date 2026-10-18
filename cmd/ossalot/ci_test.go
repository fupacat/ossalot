package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TRACEABILITY_MAPPING: REQ-054
func TestCIBuildsFrontendBeforeEvidence(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read CI workflow: %v", err)
	}
	wf := string(raw)
	at := func(marker string) int {
		i := strings.Index(wf, marker)
		if i < 0 {
			t.Fatalf("CI workflow has no %q", marker)
		}
		return i
	}
	node := at("actions/setup-node")
	build := at("npm run build")
	npmCI := at("npm ci")
	if !strings.Contains(wf[node:build], "working-directory: web") {
		t.Fatalf("the frontend build step must run in web/")
	}
	if !(node < npmCI && npmCI < build && build < at("run_evidence.py test") && build < at("run_evidence.py coverage") && build < at("go build")) {
		t.Fatalf("the frontend build must follow the Node.js setup and precede the evidence steps and go build")
	}
	t.Logf("frontend_build_step=present order=node<build<evidence<go_build")
}

// TRACEABILITY_MAPPING: REQ-057
func TestCIPublishesImageAndDeploysDev(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read CI workflow: %v", err)
	}
	wf := string(raw)
	// job returns the lines of one job under jobs:, up to the next two-space-indented key
	job := func(name string) string {
		start := strings.Index(wf, "\n  "+name+":\n")
		if start < 0 {
			t.Fatalf("CI workflow has no %s job", name)
		}
		body := wf[start+len(name)+4:]
		for i, line := range strings.SplitAfter(body, "\n") {
			if i > 0 && len(line) > 2 && line[:2] == "  " && line[2] != ' ' && line[2] != '#' {
				return body[:strings.Index(body, line)]
			}
		}
		return body
	}
	need := func(block, name string, markers ...string) {
		for _, m := range markers {
			if !strings.Contains(block, m) {
				t.Fatalf("%s job must contain %q", name, m)
			}
		}
	}
	mainPushOnly := "if: github.event_name == 'push' && github.ref == 'refs/heads/main'"

	publish := job("publish-image")
	need(publish, "publish-image", "needs: test-and-build", mainPushOnly, "packages: write",
		"ghcr.io/fupacat/ossalot:main", "ghcr.io/fupacat/ossalot:${{ github.sha }}", "push: true")

	deploy := job("deploy-dev")
	need(deploy, "deploy-dev", "needs: publish-image", mainPushOnly, "name: dev",
		"runs-on: [self-hosted, ossalot-dev]", "working-directory: deploy/dev", "OSSALOT_TAG: ${{ github.sha }}")
	pull, up := strings.Index(deploy, "docker compose pull"), strings.Index(deploy, "docker compose up -d")
	if pull < 0 || up < pull {
		t.Fatalf("deploy-dev must run docker compose pull, then docker compose up -d")
	}
	if !(strings.Index(wf, "\n  test-and-build:") < strings.Index(wf, "\n  publish-image:") &&
		strings.Index(wf, "\n  publish-image:") < strings.Index(wf, "\n  deploy-dev:")) {
		t.Fatalf("jobs must run test-and-build, then publish-image, then deploy-dev")
	}
	t.Logf("publish=main_push_only tags=main,sha deploy=dev,self-hosted,ossalot-dev order=test<publish<deploy")
}
