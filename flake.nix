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
        ];
    };
}
