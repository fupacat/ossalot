// TRACEABILITY_MAPPING: REQ-036
export function generateHygieneConfigs(language, repoFullName) {
  const lang = (language || "generic").toLowerCase();
  const repo = repoFullName || "project";

  const editorconfig = `root = true

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true
indent_style = space
indent_size = 2

[*.go]
indent_style = tab
indent_size = 4

[*.py]
indent_size = 4

[*.java]
indent_size = 4

[*.md]
trim_trailing_whitespace = false
`;

  const gitattributes = `* text=auto eol=lf
*.go text eol=lf
*.java text eol=lf
*.py text eol=lf
*.ts text eol=lf
*.js text eol=lf
*.json text eol=lf
*.md text eol=lf
*.png binary
*.jpg binary
*.jar binary
`;

  const preCommit = `repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.6.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
      - id: check-added-large-files
        args: ['--maxkb=1000']
`;

  let linterConfig = null;
  if (lang === "go" || lang === "golang") {
    linterConfig = {
      filename: ".golangci.yml",
      description: "GolangCI-Lint configuration for Go static analysis and hygiene.",
      content: `run:
  timeout: 5m
linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
`,
    };
  } else if (lang === "python") {
    linterConfig = {
      filename: "ruff.toml",
      description: "Ruff linter and code formatter configuration for Python.",
      content: `line-length = 88
target-version = "py311"

[lint]
select = ["E", "F", "I", "UP", "B", "C4"]
ignore = []
`,
    };
  } else if (lang === "java") {
    linterConfig = {
      filename: "spotless.xml",
      description: "Spotless code hygiene and formatting configuration for Java.",
      content: `<?xml version="1.0" encoding="UTF-8"?>
<configuration>
  <java>
    <googleJavaFormat>
      <version>1.22.0</version>
      <style>GOOGLE</style>
    </googleJavaFormat>
  </java>
</configuration>
`,
    };
  } else {
    linterConfig = {
      filename: ".prettierrc.json",
      description: "Prettier code formatter configuration for JS/TS/JSON/Markdown.",
      content: `{
  "semi": true,
  "singleQuote": false,
  "tabWidth": 2,
  "trailingComma": "es5"
}
`,
    };
  }

  return {
    editorconfig: {
      filename: ".editorconfig",
      description: "Cross-editor consistency for indentation, charset, and newlines.",
      content: editorconfig,
    },
    gitattributes: {
      filename: ".gitattributes",
      description: "Deterministic line ending (LF) and binary file handling across platforms.",
      content: gitattributes,
    },
    pre_commit: {
      filename: ".pre-commit-config.yaml",
      description: "Pre-commit git hooks preventing hygiene drift before commit.",
      content: preCommit,
    },
    linter: linterConfig,
  };
}
