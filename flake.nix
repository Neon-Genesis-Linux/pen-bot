{
  inputs = {
    nixpkgs.url = "https://channels.nixos.org/nixpkgs-unstable/nixexprs.tar.xz";
  };

  outputs = inputs: let
    nixpkgs = inputs.nixpkgs;
    lib = nixpkgs.lib;
    systems = ["x86_64-linux" "aarch64-linux" "aarch64-darwin"];
    version = "0.0.1-alpha";
    commit =
      if (builtins.hasAttr "self" inputs) && inputs.self != null
      && (builtins.hasAttr "rev" inputs.self)
      then inputs.self.rev
      else "dev";
    # Vendored dependency-tree hash. Obtained via vendorHash = lib.fakeHash;
    # then reading the reported value and confirming a full `nix build`.
    vendorHash = "sha256-QMHsT94FBrbhFOmfHdu8+EmqG+eMLlYZHGKLxuW9Fmk=";

    # Only ship first-party Go code and its module files to the build. The
    # gitignored dev artifacts `custom-gcl` (~55 MB) and `.venv` are excluded so
    # the hermetic derivation stays small. `lib.fileset` is used instead of
    # `filterSource`: the `builtins.filterSource` primop is unreliable here.
    src =
      let
        repo = ./.;
      in
        lib.fileset.toSource {
          root = repo;
          fileset =
            lib.fileset.unions [
              (lib.fileset.fromSource (repo + "/go.mod"))
              (lib.fileset.fromSource (repo + "/go.sum"))
              (lib.fileset.fromSource (repo + "/cmd"))
              (lib.fileset.fromSource (repo + "/internal"))
            ];
        };

    buildPen = pkgs: pkgs.buildGoModule {
      pname = "pen-fun";
      version = version;
      src = src;
      subPackages = [ "cmd/pen-fun" ];
      vendorHash = vendorHash;
      ldflags = [
        "-s"
        "-w"
        "-X"
        "github.com/Neon-Genesis-Linux/pen-bot/internal/core.Version=${version}"
        "-X"
        "github.com/Neon-Genesis-Linux/pen-bot/internal/core.Commit=${commit}"
      ];
    };
  in {
    devShells = lib.genAttrs systems (system: {
      default = nixpkgs.legacyPackages.${system}.mkShell {
        packages = with nixpkgs.legacyPackages.${system}; [
          go_1_26
          golangci-lint
          gofumpt
          gotestsum
          just
          hadolint
          yamlfmt
        ];
        # custom-gcl (golangci-lint + NilAway) is built from .custom-gcl.yml at
        # activation, mirroring .github/workflows/ci.yml. Network is available in
        # a dev shell, and the guard skips rebuilding when it already exists.
        shellHook = ''
          if [ ! -x ./custom-gcl ]; then
            golangci-lint custom
          fi
        '';
      };
    });

    packages = lib.genAttrs systems (system: {
      default = buildPen nixpkgs.legacyPackages.${system};
    });
  };
}
