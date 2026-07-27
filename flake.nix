{
  description = "repo-walker — renders a hand-authored markdown walkthrough into a static site";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-26.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    let
      # The overlay adds `repo-walker` to pkgs. Pure Go, no CGO, so it is
      # system-independent wherever Go itself builds.
      overlay = final: prev: {
        repo-walker = final.callPackage ./nix/repo-walker.nix { };
      };
    in
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ overlay ];
        };
      in
      {
        packages = {
          repo-walker = pkgs.repo-walker;
          default = pkgs.repo-walker;
        };

        devShells.default = pkgs.mkShell {
          nativeBuildInputs = with pkgs; [
            go
            git
            gh
            just
          ];
        };
      }
    ) // {
      # Overlay is system-independent; expose it at the top level so consumers
      # can do `inputs.repo-walker.overlays.default` and reference `pkgs.repo-walker`.
      overlays.default = overlay;
    };
}
