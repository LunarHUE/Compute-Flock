{
  description = "Compute Flock - Distributed Computing Agent";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };

        recipe = { lib, ... }: pkgs.buildGoModule {
          pname = "compute-flock";
          version = "0.0.2"; # You can also read this from a file if you want

          src = lib.fileset.toSource {
            root = ./.;
            fileset = lib.fileset.unions [
              ./cmd
              ./pkg
              ./go.mod
              ./go.sum
            ];
          };

          vendorHash = "sha256-PIZr3ABTF/V5X95p2NMJHyX23ypzMkboN8F/kAhowk8=";
          env.CGO_ENABLED = 0;
          ldflags = [
            "-s" "-w"
            "-X main.Version=0.0.1"
          ];

          meta = with lib; {
            description = "Compute Flock Agent";
            homepage = "https://github.com/lunarhue/compute-flock";
            license = licenses.mit;
          };
        };

      in rec {
        # 2. PACKAGES
        packages.compute-flock = pkgs.callPackage recipe {};
        packages.default = packages.compute-flock;

        # 3. DEV SHELL
        devShells.default = pkgs.mkShell {
          inputsFrom = [ packages.default ];
          packages = with pkgs; [
            go
            gopls
            gotools
            golangci-lint
          ];
        };
      })
      
      # 4. NIXOS MODULE (Systemd Configuration)
      // flake-utils.lib.eachDefaultSystemPassThrough (system: {
        nixosModules.default = { pkgs, lib, config, ...}:
          let cfg = config.services.compute-flock; in {
            
            options.services.compute-flock = with lib; {
              enable = mkEnableOption "Compute Flock Service";

              package = mkOption {
                type = types.package;
                default = self.packages.${pkgs.system}.default;
                description = "The package to use.";
              };

              apiToken = mkOption {
                type = types.str;
                default = "";
                description = "Token for authentication (passed as env var).";
              };
              
              logLevel = mkOption {
                type = types.enum [ "info" "debug" "warn" ];
                default = "info";
              };
            };

            config = lib.mkIf cfg.enable {
              systemd.services.compute-flock = {
                description = "Compute Flock Agent";
                wants = [ "network-online.target" ];
                after = [ "network-online.target" ];
                wantedBy = [ "multi-user.target" ];

                serviceConfig = {
                  ExecStart = "${cfg.package}/bin/compute-flock";
                  Environment = [ 
                    "FLOCK_TOKEN=${cfg.apiToken}"
                    "FLOCK_LOG_LEVEL=${cfg.logLevel}"
                  ];
                  
                  # Hardening
                  DynamicUser = true;
                  Restart = "always";
                  RestartSec = "5s";
                  StateDirectory = "compute-flock";
                  CacheDirectory = "compute-flock";
                };
              };
            };
          };
      });
}
