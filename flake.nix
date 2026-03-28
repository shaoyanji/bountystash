{
  description = "Bountystash - thin server-rendered Go intake app";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = {
    self,
    nixpkgs,
  }: let
    systems = ["x86_64-linux" "aarch64-linux"];
    forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f system);
  in {
    packages = forAllSystems (system: let
      pkgs = import nixpkgs {inherit system;};
    in {
      default = pkgs.buildGoModule {
        pname = "bountystash";
        version = "0.1.0";
        src = ./.;

        subPackages = ["cmd/web"];

        # If dependencies change, set this to pkgs.lib.fakeHash once,
        # run nix build, then replace with the printed real hash.
        vendorHash = "sha256-MCbuaf7FSNDhNJAQxyT6DSGPy7zbYlKGrya2FWaC8x8=";
        # vendorHash = pkgs.lib.fakeHash;

        env = {
          CGO_ENABLED = "0";
        };

        meta = with pkgs.lib; {
          description = "Thin server-rendered Go intake app";
          platforms = platforms.linux;
        };
      };
    });

    apps = forAllSystems (system: {
      default = {
        type = "app";
        program = "${self.packages.${system}.default}/bin/web";
        meta = {
          description = "Run the Bountystash web server";
        };
      };
    });

    devShells = forAllSystems (system: let
      pkgs = import nixpkgs {inherit system;};
    in {
      default = pkgs.mkShell {
        packages = with pkgs; [
          go
          gopls
          sqlc
          gcc
        ];
      };
    });
  };
}
