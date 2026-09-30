{
  inputs.flakelight.url = "github:nix-community/flakelight";
  outputs = {flakelight, ...}:
    flakelight ./. {
      systems = ["x86_64-linux" "aarch64-linux" "aarch64-darwin"];
      devShell.packages = pkgs: [
        pkgs.go
        pkgs.pre-commit
        pkgs.go-tools
        pkgs.govulncheck
        pkgs.gosec
        pkgs.golangci-lint
        # Docs tooling. uv reads the committed uv.lock and provisions its own
        # interpreter, so the Python version comes from pyproject.toml rather
        # than from nixpkgs. Use `uv run sphinx-build ...` so the venv is used.
        pkgs.uv
      ];
      # Auto-build the custom golangci-lint binary (stock linter set + NilAway
      # module plugin, see .custom-gcl.yml) when entering the shell, so that
      # `./custom-gcl run ./...` and the pre-commit custom-gcl hook work
      # out of the box. Re-run `golangci-lint custom` manually after changing
      # .custom-gcl.yml or the pinned golangci-lint version.
      devShell.shellHook = ''
        if [ ! -x "$PWD/custom-gcl" ]; then
          echo "Building custom-gcl (golangci-lint + NilAway plugin, one-time)..."
          golangci-lint custom
        fi
      '';
    };
}
