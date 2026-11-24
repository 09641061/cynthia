{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  packages = [
    pkgs.go
    pkgs.gnumake
    pkgs.stdenv.cc
  ];

  shellHook = ''
    echo "Ready. Run verification with: make verify"
  '';
}
