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

          vendorHash = "sha256-n0nO4JNfWx6FfNs0HN6lFlJYCgOIgg5+P5lMhwJRqik=";
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

              mode = mkOption {
                type = types.str;
                default = "agent";
                description = "Mode in which to run Compute Flock (agent/controller).";
              };
            };

            config = lib.mkIf cfg.enable {
              networking.firewall = {
                  allowedTCPPorts = [ 
                    6443   # Kubernetes API
                    10250  # Kubelet Metrics
                    9000   # Compute Flock Controller
                  ];
                  allowedUDPPorts = [ 
                    8472   # Flannel VXLAN
                    5353   # mDNS
                  ];
                };

              systemd.services.compute-flock = {
                description = "Compute Flock Agent";
                after = [ "network-online.target" "k3s.service" ];
                wants = [ "k3s.service" ];
                wantedBy = [ "multi-user.target" ];

                path = with pkgs; [
                  procps    # for pgrep
                  iptables  # for iptables checks
                  k3s       # for k3s binary
                ];

                serviceConfig = {
                  ExecStart = "${cfg.package}/bin/compute-flock --mode ${cfg.mode}";
                  Environment = [ 

                  ];

                  DynamicUser = false;

                  # 2. Explicitly set User to root (Optional, as it defaults to root when DynamicUser is false)
                  User = "root";
                  Group = "root";

                  # Keep your restart policies
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
