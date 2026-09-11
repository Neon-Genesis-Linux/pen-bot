{
  inputs = {
    nixpkgs.url = "https://channels.nixos.org/nixpkgs-unstable/nixexprs.tar.xz";
  };

  outputs =
    inputs:
    let
      nixpkgs = inputs.nixpkgs;
      lib = nixpkgs.lib;
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
      version = "0.0.1-alpha";
      commit =
        if
          (builtins.hasAttr "self" inputs) && inputs.self != null && (builtins.hasAttr "rev" inputs.self)
        then
          inputs.self.rev
        else
          "dev";
      # Vendored dependency-tree hash. Obtained via vendorHash = lib.fakeHash;
      # then reading the reported value and confirming a full `nix build`.
      vendorHash = "sha256-zpa8XTHdbJqCJsJYNJK1/ICCCgEvTJ32ecydaf7zEPg=";

      # Only ship first-party Go code and its module files to the build.
      # `lib.fileset` is used instead of `builtins.filterSource` as it is unreliable here.
      src =
        let
          repo = ./.;
        in
        lib.fileset.toSource {
          root = repo;
          fileset = lib.fileset.unions [
            (lib.fileset.fromSource (repo + "/go.mod"))
            (lib.fileset.fromSource (repo + "/go.sum"))
            (lib.fileset.fromSource (repo + "/cmd"))
            (lib.fileset.fromSource (repo + "/internal"))
          ];
        };

      buildPen =
        pkgs:
        pkgs.buildGoModule {
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
    in
    {
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
            dockerfmt
          ];
          shellHook = ''
            if [ ! -x ./custom-gcl ]; then
              golangci-lint custom
            fi
          '';
        };
      });

      packages = forAllSystems (pkgs: {
        default = buildPen pkgs;
      });
    };
}
