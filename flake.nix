{
  description = "Compute Flock - Pre-compiled Binary Deployment";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };

        # 1. Determine Architecture for download URL
        # We need to map NixOS system names to your Release binary names
        arch = if system == "x86_64-linux" then "amd64"
               else if system == "aarch64-linux" then "arm64"
               else throw "Unsupported system: ${system}";

        version = "0.0.1"; # Update this to your release tag

        # 2. Define the Package (Downloads Binary, doesn't build)
        compute-flock-bin = pkgs.stdenv.mkDerivation {
          pname = "compute-flock";
          inherit version;

          # DOWNLOADS THE BINARY DIRECTLY
          src = pkgs.fetchurl {
            # Assumes your release binary is named "compute-flock_linux_amd64"
            url = "https://github.com/lunarhue/compute-flock/releases/download/v${version}/compute-flock_linux_${arch}";
            
            # YOU MUST UPDATE THESE HASHES WHEN YOU UPDATE THE VERSION
            # Nix needs to know the hash of the file before downloading it.
            sha256 = if system == "x86_64-linux" 
                     then "sha256-0000000000000000000000000000000000000000000=" # Replace with actual hash
                     else "sha256-0000000000000000000000000000000000000000000="; # Replace with actual hash
          };

          dontUnpack = true; # It's a binary, not a tarball, so don't try to unzip it

          installPhase = ''
            mkdir -p $out/bin
            cp $src $out/bin/compute-flock
            chmod +x $out/bin/compute-flock
          '';
        };

      in {
        # Export the package
        packages.default = compute-flock-bin;
        packages.compute-flock = compute-flock-bin;
      }
    ) // {
      # 3. Define the NixOS Module
      # This part runs on the server to configure Systemd
      nixosModules.default = { config, lib, pkgs, ... }:
        let
          cfg = config.services.compute-flock;
        in {
          options.services.compute-flock = with lib; {
            enable = mkEnableOption "Compute Flock Agent";

            package = mkOption {
              type = types.package;
              default = self.packages.${pkgs.system}.default;
              description = "The compute-flock package to use.";
            };

            # Example Config Option: The API Token
            apiToken = mkOption {
              type = types.str;
              default = "";
              description = "The authentication token for the flock.";
            };
            
            # Example Config Option: Extra args
            extraArgs = mkOption {
              type = types.listOf types.str;
              default = [];
              description = "Extra arguments to pass to the binary.";
            };
          };

          config = lib.mkIf cfg.enable {
            systemd.services.compute-flock = {
              description = "Compute Flock Agent Service";
              wantedBy = [ "multi-user.target" ];
              wants = [ "network-online.target" ];
              after = [ "network-online.target" ];

              serviceConfig = {
                ExecStart = "${cfg.package}/bin/compute-flock ${lib.escapeShellArgs cfg.extraArgs}";
                Restart = "always";
                RestartSec = "10s";
                
                # Security Hardening (Similar to Himmelblau)
                DynamicUser = true; # Runs as a restricted, ephemeral user
                StateDirectory = "compute-flock"; # Creates /var/lib/compute-flock
                CacheDirectory = "compute-flock"; # Creates /var/cache/compute-flock
                
                # Pass secrets via Environment variables to avoid them showing in 'ps'
                Environment = [
                  "FLOCK_TOKEN=${cfg.apiToken}"
                ];
              };
            };
          };
        };
    };
}
