{
  inputs = {
    nixpkgs.url = "https://channels.nixos.org/nixpkgs-unstable/nixexprs.tar.xz";
  };

  outputs =
    inputs:
    let
      version = "0.0.1-alpha";
      # Vendored dependency-tree hash. Regenerate after changing dependencies:
      # set `vendorHash = lib.fakeHash`, run `nix build`, and take the value the
      # build reports, so a wrong hash fails the build rather than pulling network access.
      vendorHash = "sha256-zpa8XTHdbJqCJsJYNJK1/ICCCgEvTJ32ecydaf7zEPg=";
      nixpkgs = inputs.nixpkgs;
      lib = nixpkgs.lib;
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
      commit =
        if
          (builtins.hasAttr "self" inputs) && inputs.self != null && (builtins.hasAttr "rev" inputs.self)
        then
          inputs.self.rev
        else
          "dev";

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
        in
        {
          options.services."pen-bot" = {
            enable = lib.mkEnableOption "Pen Bot Discord bot service";

            botTokenFile = lib.mkOption {
              type = lib.types.str;
              default = "";
              description = ''
                Absolute path to a root-owned file containing the Discord bot
                token in systemd `EnvironmentFile` format, i.e. a single line
                `BOT_TOKEN=<token>`. systemd reads the file as root at
                activation time and passes it straight to the service, so the
                token never enters the world-readable Nix store. Required when
                the service is enabled.
              '';
              example = "/etc/secrets/pen-bot/token-env";
            };

            guildID = lib.mkOption {
              type = lib.types.str;
              default = "";
              description = "Discord guild ID to register commands against.";
            };
          };

          config = {
            systemd.services."pen-bot" = lib.mkIf cfg.enable (
              lib.mkAssert (cfg.botTokenFile != "")
                ''
                  services."pen-bot".botTokenFile is required when the service is
                  enabled. Point it at a root-owned file containing the Discord bot
                  token in systemd `EnvironmentFile` format, i.e. a single line
                  `BOT_TOKEN=<token>`.
                ''
                {
                  description = "Pen Bot Discord service";

                  wants = [ "network-online.target" ];
                  after = [
                    "network-online.target"
                    "local-fs.target"
                  ];
                  wantedBy = [ "multi-user.target" ];

                  serviceConfig = {
                    ExecStart = lib.getExe' (buildPen pkgs) "pen-fun";

                    Environment = [
                      "ENV=production"
                      # The database lives inside the service's StateDirectory
                      # (/var/lib/pen-bot): that is the only writable location a
                      # DynamicUser can reach, so its path is fixed rather than
                      # configurable.
                      "DB_PATH=/var/lib/pen-bot/pen-bot.db"
                    ]
                    ++ (if cfg.guildID != "" then [ "GUILD_ID=${cfg.guildID}" ] else [ ]);

                    # The bot reads BOT_TOKEN at runtime. systemd reads the file as
                    # root at activation time, so the token never enters the
                    # world-readable Nix store: pass the path directly rather than
                    # reading the file's contents into the derivation.
                    EnvironmentFile = lib.optionals (cfg.botTokenFile != "") [ cfg.botTokenFile ];

                    DynamicUser = true;
                    StateDirectory = "pen-bot";
                    StateDirectoryMode = "0750";

                    # No separate ReadWritePaths is needed: StateDirectory already
                    # makes /var/lib/pen-bot writable and owned by the (dynamic)
                    # user, and that is where the database lives.

                    Restart = "on-failure";
                    RestartSec = "5s";

                    # Sandboxing. DynamicUser already confines the service to an
                    # ephemeral, unprivileged namespace; these flags tighten it
                    # further. The qbittorrent service module lists the full set.
                    PrivateTmp = true;
                    ProtectSystem = "strict";
                    ProtectHome = true;
                    NoNewPrivileges = true;
                    ProtectKernelModules = true;
                    ProtectKernelTunables = true;
                    ProtectKernelLogs = true;
                    ProtectControlGroups = true;
                    ProtectClock = true;
                    ProtectHostname = true;
                    ProtectProc = "invisible";
                    ProcSubset = "pid";
                    RestrictNamespaces = true;
                    RestrictRealtime = true;
                    RestrictSUIDSGID = true;
                    LockPersonality = true;
                    MemoryDenyWriteExecute = true;
                    # Keep only what a network client needs: IPv4/IPv6 sockets and
                    # the local sockets systemd/journald use.
                    RestrictAddressFamilies = [
                      "AF_INET"
                      "AF_INET6"
                      "AF_UNIX"
                      "AF_NETLINK"
                    ];
                    # Only allow syscalls for the local architecture.
                    SystemCallArchitectures = "native";
                    CapabilityBoundingSet = "";
                  };
                }
            );
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
