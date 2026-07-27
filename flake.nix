{
  description = "walkr — renders a hand-authored markdown walkthrough into a static site";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-26.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    let
      # The overlay adds `walkr` to pkgs. Pure Go, no CGO, so it is
      # system-independent wherever Go itself builds.
      overlay = final: prev: {
        walkr = final.callPackage ./nix/walkr.nix { };
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
          walkr = pkgs.walkr;
          default = pkgs.walkr;
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
      # can do `inputs.walkr.overlays.default` and reference `pkgs.walkr`.
      overlays.default = overlay;
    };
}
