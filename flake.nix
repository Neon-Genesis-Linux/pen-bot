{
  inputs = {
    nixpkgs.url = "https://channels.nixos.org/nixpkgs-unstable/nixexprs.tar.xz";
  };

  outputs = {nixpkgs, ...}: let
    systems = ["x86_64-linux" "aarch64-linux" "aarch64-darwin"];
    forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    version = "0.0.1-alpha";
    vendorHash = "sha256-ARMoq5m16oPQEHkM8FfQBDFOGUhRxgLbonTYxQh9NV8=";
  in {
    devShells = forAllSystems (pkgs: {
      default = pkgs.mkShell {
        packages = with pkgs; [
          go_1_26
          golangci-lint
          gofumpt
          gotestsum
          just
          hadolint
          yamlfmt
        ];
      };
    });
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
