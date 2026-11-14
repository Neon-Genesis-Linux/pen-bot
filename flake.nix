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
      # Vendored dependency-tree hash. Regenerate after changing dependencies:
      # set `vendorHash = lib.fakeHash`, run `nix build`, and take the value the
      # build reports, so a wrong hash fails the build rather than pulling network access.
      vendorHash = "sha256-zpa8XTHdbJqCJsJYNJK1/ICCCgEvTJ32ecydaf7zEPg=";

      # Only ship first-party Go code and its module files to the build.
      # `builtins.filterSource` is unreliable here.
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
          pname = "pen-bot";
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

      # NixOS deployment for the same binary as `packages.default`. DB_PATH
      # points at a persistent StateDirectory, so the SQLite database stays in
      # /var/lib and survives rebuilds, which change only the binary's path in
      # the immutable store.
      penBotModule =
        {
          config,
          pkgs,
          lib,
          ...
        }:
        let
          cfg = config.services."pen-bot";

          # The app reads the token from BOT_TOKEN at runtime. systemd reads
          # EnvironmentFile as root during activation, so a root-owned file
          # keeps it out of the service user. The store is world-readable, and
          # writeSecretFile is absent from some nixpkgs versions, so a plain
          # writeText file is sufficient.
          tokenFile = pkgs.writeText "pen-bot-token-env" "BOT_TOKEN=${builtins.readFile cfg.botTokenFile}\n";
        in
        {
          options.services."pen-bot" = {
            enable = lib.mkEnableOption "Pen Bot Discord service";

            botTokenFile = lib.mkOption {
              type = lib.types.path;
              description = "Path to a file containing the Discord bot token.";
              example = "/etc/secrets/pen-bot/token";
            };

            guildID = lib.mkOption {
              type = lib.types.str;
              default = "";
              description = "Discord guild ID to register commands against.";
            };

            dataFile = lib.mkOption {
              type = lib.types.path;
              default = "${config.statePrefix}/pen-bot/pen-bot.db";
              description = "Absolute path to the SQLite database file.";
            };
          };

          config = {
            systemd."pen-bot" = lib.mkIf cfg.enable {
              description = "Pen Bot Discord service";

              after = [ "network.target" ];
              wants = [ "network.target" ];
              wantedBy = [ "multi-user.target" ];

              EnvironmentFile = [ tokenFile ];

              Environment = [
                "ENV=production"
                "DB_PATH=${cfg.dataFile}"
              ]
              ++ (if cfg.guildID != "" then [ "GUILD_ID=${cfg.guildID}" ] else [ ]);

              serviceConfig = {
                ExecStart = "${buildPen pkgs}/bin/pen-bot";
                DynamicUser = true;
                StateDirectory = "pen-bot";
                StateDirectoryMode = "0750";
                ReadWritePaths = "${config.statePrefix}/pen-bot";
                Restart = "on-failure";
                RestartSec = "5s";
                PrivateTmp = true;
                ProtectSystem = "strict";
                ProtectHome = true;
                NoNewPrivileges = true;
              };
            };
          };
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

      nixosModules = {
        default = penBotModule;
      };
    };
}
