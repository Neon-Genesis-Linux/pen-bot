{
  inputs.flakelight.url = "github:nix-community/flakelight";
  outputs =
    { flakelight, self, ... }:
    let
      version = "0.0.1-alpha";
      vendorHash = "sha256-ARMoq5m16oPQEHkM8FfQBDFOGUhRxgLbonTYxQh9NV8=";
    in
    flakelight ./. {
      inputs.self = self;
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];
      devShell.packages = pkgs: [
        pkgs.go
        pkgs.pre-commit
        pkgs.go-tools
        pkgs.govulncheck
        pkgs.gosec
        pkgs.golangci-lint
      ];

      packages = {
        default =
          { pkgs, inputs }:
          pkgs.buildGoModule {
            pname = "pen-fun";
            version = version;
            src = ./.;
            subPackages = [ "cmd/pen-fun" ];
            vendorHash = vendorHash;
            ldflags = [
              "-s"
              "-w"
              "-X"
              "github.com/Neon-Genesis-Linux/pen-bot/internal/core.Version=${version}"
              "-X"
              "github.com/Neon-Genesis-Linux/pen-bot/internal/core.Commit=${inputs.self.rev or "dev"}"
            ];
          };
      };
    };
}
