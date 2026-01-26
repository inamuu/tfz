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

## Download

Go to release page and download binary.

## Run
```sh
mv tfz /usr/local/bin/
```

## If you get an error on mac

Remove extention attributes for mac from the binary.

```sh
xattr -rc tfz
```

## Usage
1. Type to filter the target list.
2. Press `Space` to toggle targets.
3. Press `Enter` to confirm, then choose `plan` or `apply`.

Notes:
- Selecting `all` ignores any other target selection and runs without `-target`.
- If nothing is selected, `all` is chosen by default.

## Screenshots
Main

<img width="835" height="231" alt="Image" src="https://github.com/user-attachments/assets/98cd9df4-1ed1-4f9a-a765-5157bcfe17d4" />

Filter (after typing)

<img width="814" height="109" alt="Image" src="https://github.com/user-attachments/assets/73476a2f-481a-4842-965a-3677d83191d6" />

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
