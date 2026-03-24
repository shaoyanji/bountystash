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

        # First build will fail with the real hash. Replace fakeHash
        # with the value printed by nix build.
        vendorHash = "sha256-SmtR/uqEv56LfDZGRxgH4XbG9xvxTntVBIWnTey00GU=";

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
