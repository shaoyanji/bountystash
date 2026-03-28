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
        version = "0.1.2";
        src = ./.;

        subPackages = ["cmd/web"];

        # Hash update ritual (must be repeated for this package when Go deps,
        # vendoring, or package inputs change):
        # 1) set vendorHash = pkgs.lib.fakeHash;
        # 2) run nix build .#default;
        # 3) replace with the hash shown in the build error.
        vendorHash = "sha256-zVLALLW4ZkwYD7bJ0UOZ206fRpBJ1FnlVy/ugZI1g8k=";

        env = {
          CGO_ENABLED = "0";
        };

        meta = with pkgs.lib; {
          description = "Thin server-rendered Go intake app";
          platforms = platforms.linux;
        };
      };

      tui = pkgs.buildGoModule {
        pname = "bountystash-tui";
        version = "0.1.2";
        src = ./.;

        subPackages = ["cmd/bountystash-tui"];

        # Hash update ritual (must be repeated for this package when Go deps,
        # vendoring, or package inputs change):
        # 1) set vendorHash = pkgs.lib.fakeHash;
        # 2) run nix build .#tui;
        # 3) replace with the hash shown in the build error.
        vendorHash = "sha256-zVLALLW4ZkwYD7bJ0UOZ206fRpBJ1FnlVy/ugZI1g8k=";
        ldflags = [
          "-X github.com/shaoyanji/bountystash/internal/version.Version=0.1.2"
          "-X github.com/shaoyanji/bountystash/internal/version.Commit=flake"
          "-X github.com/shaoyanji/bountystash/internal/version.Date=unknown"
        ];

        env = {
          CGO_ENABLED = "0";
        };

        meta = with pkgs.lib; {
          description = "Keyboard-first TUI for the Bountystash backend";
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

      tui = {
        type = "app";
        program = "${self.packages.${system}.tui}/bin/bountystash-tui";
        meta = {
          description = "Run the Bountystash terminal UI client";
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
