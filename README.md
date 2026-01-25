# TFZ

A small, fast TUI for running Terraform `plan` and `apply` with optional targets.

## Features
- Scans `.tf` files in the current directory and lists `module` / `resource` targets
- Multi-select targets (fzf-style filtering)
- `all` option runs Terraform without `-target`
- Choose `plan` or `apply` after selecting targets
- Dracula-inspired color theme and lazygit-style header

## Requirements
- Go 1.22+
- Terraform in your `PATH`

## Install
```sh
go build -o tfz .
```

## Run
```sh
./tfz
```

## Usage
1. Type to filter the target list.
2. Press `Space` to toggle targets.
3. Press `Enter` to confirm, then choose `plan` or `apply`.

Notes:
- Selecting `all` ignores any other target selection and runs without `-target`.
- If nothing is selected, `all` is chosen by default.

## Development
```sh
go run .
```

## Roadmap
- Preview command before execution
- Highlight fuzzy match positions
- Optional key guide footer

## License
MIT
